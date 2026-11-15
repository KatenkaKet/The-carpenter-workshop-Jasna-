package app

import "time"

// All callers run inside the same serialized transaction as booking/payment changes.
func (r *R) reserve(b, l M, kind string) error {
	if kind == "single_lesson" {
		return nil
	}
	day := at(l, "starts_at").In(zone).Format("2006-01-02")
	s, e := one(r.ctx(), r.DB, `SELECT s.* FROM subscriptions s WHERE child_id=$1 AND group_id=$2 AND valid_from<=$3::date AND valid_until>=$3::date AND suspended_at IS NULL AND lessons_remaining>(SELECT count(*) FROM subscription_reservations x WHERE x.subscription_id=s.id AND x.status='reserved') ORDER BY valid_until,id LIMIT 1`, b["child_id"], l["group_id"], day)
	if e != nil {
		if ae, ok := e.(*APIError); !ok || ae.Status != 404 {
			return e
		}
		var exists bool
		if e = r.DB.QueryRow(r.ctx(), `SELECT EXISTS(SELECT 1 FROM subscriptions WHERE child_id=$1 AND group_id=$2)`, b["child_id"], l["group_id"]).Scan(&exists); e != nil {
			return e
		}
		if kind == "subscription" || exists {
			return conflict("NO_AVAILABLE_LESSONS")
		}
		return nil // compatibility: pre-existing API clients can still book a single lesson
	}
	if _, e = r.insert("subscription_reservations", M{"booking_id": b["id"], "subscription_id": s["id"]}); e != nil {
		return e
	}
	_, e = r.update("bookings", str(b, "id"), M{"funding_kind": "subscription"})
	return e
}

func (r *R) effectiveRule(b M) (M, error) {
	rule := clone(object(b["rule_snapshot"]))
	p, e := one(r.ctx(), r.DB, `SELECT p.* FROM group_policies p JOIN lessons l ON l.group_id=p.group_id WHERE l.id=$1`, b["lesson_id"])
	if e != nil {
		if ae, ok := e.(*APIError); ok && ae.Status == 404 {
			return rule, nil
		}
		return nil, e
	}
	for _, k := range []string{"missed_charge_lesson", "free_cancel_before_hours", "transfer_allowed"} {
		rule[k] = p[k]
	}
	// Saving the free-cancellation period explicitly enables charging after that period.
	rule["parent_cancel_charge_lesson"] = true
	rule["policy_version"] = p["version"]
	return rule, nil
}
func groupPolicy(r *R) (any, error) {
	id := r.C.Param("id")
	if _, e := r.get("groups", id); e != nil {
		return nil, e
	}
	if r.C.Request.Method == "GET" {
		p, e := one(r.ctx(), r.DB, "SELECT * FROM group_policies WHERE group_id=$1", id)
		if e == nil {
			return p, nil
		}
		if ae, ok := e.(*APIError); !ok || ae.Status != 404 {
			return nil, e
		}
		rule, e := r.rule(id, r.now().In(zone).Format("2006-01-02"))
		if e != nil {
			return nil, e
		}
		return M{"group_id": id, "version": 0, "missed_charge_lesson": rule["missed_charge_lesson"], "free_cancel_before_hours": rule["free_cancel_before_hours"], "transfer_allowed": rule["transfer_allowed"]}, nil
	}
	if e := r.fields("expected_version missed_charge_lesson free_cancel_before_hours transfer_allowed", ""); e != nil {
		return nil, e
	}
	if e := integer(r.Body, "expected_version", 0, 100000000); e != nil {
		return nil, e
	}
	for _, k := range []string{"missed_charge_lesson", "transfer_allowed"} {
		if e := boolean(r.Body, k); e != nil {
			return nil, e
		}
	}
	if e := integer(r.Body, "free_cancel_before_hours", 0, 8760); e != nil {
		return nil, e
	}
	old, e := one(r.ctx(), r.DB, "SELECT * FROM group_policies WHERE group_id=$1", id)
	if e != nil {
		if ae, ok := e.(*APIError); !ok || ae.Status != 404 {
			return nil, e
		}
		old = M{"version": 0}
	}
	if e = r.version(old); e != nil {
		return nil, e
	}
	m := clean(r.Body, "expected_version")
	m["group_id"] = id
	var v M
	if str(old, "id") == "" {
		v, e = r.insert("group_policies", m)
	} else {
		v, e = r.update("group_policies", str(old, "id"), m)
	}
	if e != nil {
		return nil, e
	}
	return v, r.audit("group.policy.updated", "groups", id, old, v, "Настройки применяются к следующим операциям")
}
func cancellationQuote(r *R) (any, error) {
	b, e := r.owned("bookings", r.C.Param("id"))
	if e != nil {
		return nil, e
	}
	l, e := r.get("lessons", str(b, "lesson_id"))
	if e != nil {
		return nil, e
	}
	rule, e := r.effectiveRule(b)
	if e != nil {
		return nil, e
	}
	start := at(l, "starts_at")
	allowed := str(b, "status") == "active" && start.After(r.now()) && yes(rule, "parent_cancel_allowed") && start.Sub(r.now()).Hours() >= float64(num(rule, "cancel_deadline_hours"))
	deadline := start.Add(-time.Duration(num(rule, "free_cancel_before_hours")) * time.Hour)
	if !yes(rule, "parent_cancel_charge_lesson") {
		deadline = start
	}
	return M{"booking_id": b["id"], "allowed": allowed, "server_time": r.now(), "timezone": "Asia/Yekaterinburg", "free_cancel_until": deadline, "will_charge": chargeFor(rule, "cancelled", "parent", start, r.now()), "rules": rule}, nil
}

// Purchase and cash/transfer acknowledgement are one idempotent transaction.
func manualPurchase(r *R) (any, error) {
	if e := r.fields("child_id plan_id valid_from method", "comment"); e != nil {
		return nil, e
	}
	method := str(r.Body, "method")
	if method != "cash" && method != "transfer" {
		return nil, invalid("PAYMENT_METHOD")
	}
	input := r.Body
	r.Body = M{"child_id": input["child_id"], "plan_id": input["plan_id"], "valid_from": input["valid_from"], "kind": "subscription"}
	inv, e := invoiceCreate(r)
	if e != nil {
		return nil, e
	}
	r.Body = M{"invoice_id": object(inv)["id"], "method": method}
	if input["comment"] != nil {
		r.Body["comment"] = input["comment"]
	}
	return manualPayment(r)
}
func paymentVoid(r *R) (any, error) {
	if e := r.fields("reason expected_version", ""); e != nil {
		return nil, e
	}
	if e := validText(r.Body, "reason", 1000); e != nil {
		return nil, e
	}
	p, e := r.get("payments", r.C.Param("id"))
	if e != nil {
		return nil, e
	}
	if str(p, "status") == "voided" {
		return paymentPublic(p), nil
	}
	if e = r.version(p); e != nil {
		return nil, e
	}
	if str(p, "status") != "succeeded" {
		return nil, conflict("PAYMENT_NOT_SUCCEEDED")
	}
	if str(p, "method") != "cash" && str(p, "method") != "transfer" {
		return nil, conflict("PROVIDER_REFUND_REQUIRED")
	}
	entry, e := one(r.ctx(), r.DB, "SELECT * FROM money_ledger WHERE payment_id=$1 AND reason='payment'", p["id"])
	if e != nil {
		return nil, e
	}
	if _, e = r.insert("money_ledger", M{"family_id": p["family_id"], "invoice_id": p["invoice_id"], "payment_id": p["id"], "amount_minor": -num(p, "amount_minor"), "reason": "payment_void", "source_id": p["id"], "reverses_entry_id": entry["id"]}); e != nil {
		return nil, e
	}
	v, e := r.update("payments", str(p, "id"), M{"status": "voided", "voided_at": r.now(), "void_reason": r.Body["reason"]})
	if e != nil {
		return nil, e
	}
	inv, e := r.get("invoices", str(p, "invoice_id"))
	if e != nil {
		return nil, e
	}
	if !yes(inv, "compensated") {
		if _, e = r.update("invoices", str(inv, "id"), M{"status": "open"}); e != nil {
			return nil, e
		}
	}
	if e = r.exec("UPDATE subscriptions SET suspended_at=now(),version=version+1 WHERE invoice_id=$1", p["invoice_id"]); e != nil {
		return nil, e
	}
	if e = r.notify(str(p, "family_id"), "payment.voided", str(p, "id")+":voided", M{"payment_id": p["id"], "invoice_id": p["invoice_id"], "message": "Мастер аннулировал ошибочную оплату. Проверьте счёт."}); e != nil {
		return nil, e
	}
	return paymentPublic(v), r.audit("payment.voided", "payments", str(p, "id"), paymentPublic(p), paymentPublic(v), str(r.Body, "reason"))
}
func receiptDelete(r *R) (any, error) {
	if e := r.fields("reason expected_version", ""); e != nil {
		return nil, e
	}
	if e := validText(r.Body, "reason", 1000); e != nil {
		return nil, e
	}
	old, e := r.get("receipts", r.C.Param("id"))
	if e != nil {
		return nil, e
	}
	if old["deleted_at"] != nil {
		return nil, nil
	}
	if e = r.version(old); e != nil {
		return nil, e
	}
	if e = r.exec(`INSERT INTO receipt_versions(receipt_id,snapshot,content,comment,actor_id) SELECT id,to_jsonb(receipts)-'content',content,$2,$3 FROM receipts WHERE id=$1`, old["id"], str(r.Body, "reason"), r.User["user_id"]); e != nil {
		return nil, e
	}
	_, e = r.update("receipts", str(old, "id"), M{"deleted_at": r.now()})
	if e != nil {
		return nil, e
	}
	return nil, r.audit("receipt.deleted", "receipts", str(old, "id"), clean(old, "content"), M{"deleted": true}, str(r.Body, "reason"))
}
