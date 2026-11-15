package app

import (
	"time"
)

func (r *R) balance(id string) (M, error) {
	return one(r.ctx(), r.DB, `SELECT coalesce(sum(total),0)::bigint balance_minor,coalesce(sum(greatest(-total,0)),0)::bigint debt_minor,'RUB' currency FROM (SELECT sum(amount_minor) total FROM money_ledger WHERE family_id=$1 GROUP BY invoice_id) s`, id)
}

func balance(r *R) (any, error) {
	id := r.C.Param("id")
	if id == "" {
		id = str(r.User, "family_id")
	}
	if _, e := r.get("families", id); e != nil {
		return nil, e
	}
	return r.balance(id)
}

func moneyList(r *R) (any, error) {
	id := r.C.Param("id")
	if id == "" {
		id = str(r.User, "family_id")
	}
	from, to, e := r.rangeDates(false)
	if e != nil {
		return nil, e
	}
	return r.list(`SELECT * FROM money_ledger WHERE family_id=$1 AND ($2='' OR created_at>=nullif($2,'')::timestamptz) AND ($3='' OR created_at<nullif($3,'')::timestamptz) ORDER BY created_at,id`, id, from, to)
}

func (r *R) invoiceCreate(m M) (M, error) {
	v, e := r.insert("invoices", m)
	if e != nil {
		return nil, e
	}
	if _, e = r.insert("money_ledger", M{"family_id": v["family_id"], "invoice_id": v["id"], "amount_minor": -num(v, "amount_minor"), "reason": "invoice", "source_id": v["id"]}); e != nil {
		return nil, e
	}
	if e = r.notify(str(v, "family_id"), "payment.required", str(v, "id")+":required", M{"invoice_id": v["id"]}); e != nil {
		return nil, e
	}
	v["due_minor"] = v["amount_minor"]
	return v, r.audit("invoice.created", "invoices", str(v, "id"), nil, v, "")
}

func (r *R) singleInvoice(b, l M) (M, error) {
	return r.invoiceCreate(M{"family_id": b["family_id"], "child_id": b["child_id"], "group_id": l["group_id"], "kind": "single_lesson", "booking_id": b["id"], "amount_minor": object(b["rule_snapshot"])["single_lesson_price_minor"]})
}

func invoiceCreate(r *R) (any, error) {
	kind := str(r.Body, "kind")
	switch kind {
	case "subscription":
		if e := r.fields("child_id kind plan_id valid_from", ""); e != nil {
			return nil, e
		}
		child, e := r.owned("children", str(r.Body, "child_id"))
		if e != nil {
			return nil, e
		}
		if !yes(child, "is_active") {
			return nil, conflict("CHILD_INACTIVE")
		}
		plan, e := r.get("plans", str(r.Body, "plan_id"))
		if e != nil {
			return nil, e
		}
		if !yes(plan, "is_active") {
			return nil, conflict("PLAN_INACTIVE")
		}
		from, e := date(str(r.Body, "valid_from"))
		if e != nil {
			return nil, e
		}
		until := from.AddDate(0, 0, int(num(plan, "validity_days"))-1)
		if until.Before(func() time.Time { t, _ := date(r.now().In(zone).Format("2006-01-02")); return t }()) {
			return nil, invalid("SUBSCRIPTION_EXPIRED")
		}
		return r.invoiceCreate(M{"family_id": child["family_id"], "child_id": child["id"], "group_id": plan["group_id"], "kind": kind, "amount_minor": plan["price_minor"], "plan_snapshot": plan, "valid_from": from.Format("2006-01-02"), "valid_until": until.Format("2006-01-02")})
	case "single_lesson":
		if e := r.fields("child_id kind booking_id", ""); e != nil {
			return nil, e
		}
		b, e := r.owned("bookings", str(r.Body, "booking_id"))
		if e != nil {
			return nil, e
		}
		if b["child_id"] != r.Body["child_id"] || str(b, "status") != "active" {
			return nil, conflict("BOOKING_UNAVAILABLE")
		}
		var reserved bool
		if e = r.DB.QueryRow(r.ctx(), "SELECT EXISTS(SELECT 1 FROM subscription_reservations WHERE booking_id=$1 AND status IN ('reserved','consumed'))", b["id"]).Scan(&reserved); e != nil {
			return nil, e
		}
		if reserved {
			return nil, conflict("ALREADY_COVERED")
		}
		a, e := one(r.ctx(), r.DB, "SELECT * FROM attendance WHERE booking_id=$1", b["id"])
		if e != nil {
			return nil, e
		}
		if yes(a, "charged") && str(a, "subscription_id") != "" {
			return nil, conflict("ALREADY_COVERED")
		}
		var exists bool
		if e = r.DB.QueryRow(r.ctx(), "SELECT EXISTS(SELECT 1 FROM invoices WHERE booking_id=$1)", b["id"]).Scan(&exists); e != nil {
			return nil, e
		}
		if exists {
			return nil, conflict("INVOICE_EXISTS")
		}
		l, e := r.get("lessons", str(b, "lesson_id"))
		if e != nil {
			return nil, e
		}
		return r.singleInvoice(b, l)
	default:
		return nil, invalid("INVOICE_KIND")
	}
}

func (r *R) compensateInvoice(id string, cancel bool) error {
	inv, e := r.get("invoices", id)
	if e != nil {
		return e
	}
	if yes(inv, "compensated") == cancel {
		return nil
	}
	var pending bool
	if e = r.DB.QueryRow(r.ctx(), "SELECT EXISTS(SELECT 1 FROM payments WHERE invoice_id=$1 AND status='pending')", id).Scan(&pending); e != nil {
		return e
	}
	if pending {
		return conflict("PAYMENT_IN_PROGRESS")
	}
	delta := -num(inv, "amount_minor")
	reason := "invoice_reinstated"
	status := "open"
	if cancel {
		delta = -delta
		reason = "invoice_reversal"
		status = "cancelled"
	} else {
		var paid int64
		if e = r.DB.QueryRow(r.ctx(), "SELECT coalesce(sum(amount_minor),0)::bigint FROM payments WHERE invoice_id=$1 AND status='succeeded'", id).Scan(&paid); e != nil {
			return e
		}
		if paid >= num(inv, "amount_minor") {
			status = "paid"
		}
	}
	last, e := one(r.ctx(), r.DB, "SELECT * FROM money_ledger WHERE invoice_id=$1 AND reason IN ('invoice','invoice_reversal','invoice_reinstated') ORDER BY created_at DESC,id DESC LIMIT 1", id)
	if e != nil {
		return e
	}
	if _, e = r.insert("money_ledger", M{"family_id": inv["family_id"], "invoice_id": id, "amount_minor": delta, "reason": reason, "source_id": id, "reverses_entry_id": last["id"]}); e != nil {
		return e
	}
	v, e := r.update("invoices", id, M{"compensated": cancel, "status": status})
	if e != nil {
		return e
	}
	return r.audit(reason, "invoices", id, inv, v, "")
}

func invoiceCancel(r *R) (any, error) {
	if e := r.fields("reason expected_version", ""); e != nil {
		return nil, e
	}
	if e := validText(r.Body, "reason", 1000); e != nil {
		return nil, e
	}
	inv, e := r.get("invoices", r.C.Param("id"))
	if e != nil {
		return nil, e
	}
	if e = r.version(inv); e != nil {
		return nil, e
	}
	if str(inv, "status") != "open" {
		return nil, conflict("INVOICE_NOT_OPEN")
	}
	if e = r.compensateInvoice(str(inv, "id"), true); e != nil {
		return nil, e
	}
	r.Status = 200
	return r.get("invoices", str(inv, "id"))
}

const invoiceSQL = `SELECT i.*,greatest(0,-coalesce((SELECT sum(amount_minor) FROM money_ledger WHERE invoice_id=i.id),0))::bigint due_minor FROM invoices i`

func invoiceGet(r *R) (any, error) {
	if _, e := r.owned("invoices", r.C.Param("id")); e != nil {
		return nil, e
	}
	return one(r.ctx(), r.DB, invoiceSQL+" WHERE i.id=$1", r.C.Param("id"))
}

func invoices(r *R) (any, error) {
	family := r.C.Query("family_id")
	if str(r.User, "role") == "parent" {
		family = str(r.User, "family_id")
	}
	return r.list(invoiceSQL+` WHERE ($1='' OR i.family_id::text=$1) AND ($2='' OR i.child_id::text=$2) AND ($3='' OR i.status=$3) ORDER BY i.created_at,i.id`, family, r.C.Query("child_id"), r.C.Query("status"))
}

const subSQL = `SELECT s.*,(SELECT i.plan_snapshot->>'title' FROM invoices i WHERE i.id=s.invoice_id) plan_title,(SELECT count(*) FROM subscription_reservations x WHERE x.subscription_id=s.id AND x.status='reserved')::integer lessons_reserved,CASE WHEN s.suspended_at IS NOT NULL OR s.valid_until<(now() AT TIME ZONE 'Asia/Yekaterinburg')::date THEN 0 ELSE greatest(0,s.lessons_remaining-(SELECT count(*) FROM subscription_reservations x WHERE x.subscription_id=s.id AND x.status='reserved')) END::integer lessons_available,coalesce((SELECT -sum(delta) FROM subscription_ledger WHERE subscription_id=s.id AND reason IN ('attendance_charge','attendance_reversal')),0)::integer lessons_used,coalesce((SELECT sum(delta) FROM subscription_ledger WHERE subscription_id=s.id AND reason='manual_adjustment'),0)::integer adjustments_total,CASE WHEN suspended_at IS NOT NULL THEN 'suspended' WHEN lessons_remaining=0 THEN 'exhausted' WHEN valid_until<(now() AT TIME ZONE 'Asia/Yekaterinburg')::date THEN 'expired' WHEN valid_from>(now() AT TIME ZONE 'Asia/Yekaterinburg')::date THEN 'scheduled' ELSE 'active' END status FROM subscriptions s`

func subscriptions(r *R) (any, error) {
	family, child := "", r.C.Query("child_id")
	if str(r.User, "role") == "parent" {
		family = str(r.User, "family_id")
	}
	if r.C.Param("id") != "" {
		child = r.C.Param("id")
	}
	return r.list("SELECT * FROM ("+subSQL+`) x WHERE ($1='' OR family_id::text=$1) AND ($2='' OR child_id::text=$2) AND ($3='' OR $3='all' OR status=$3) ORDER BY created_at,id`, family, child, r.C.Query("status"))
}

func subscriptionLedger(r *R) (any, error) {
	if _, e := r.owned("subscriptions", r.C.Param("id")); e != nil {
		return nil, e
	}
	from, to, e := r.rangeDates(false)
	if e != nil {
		return nil, e
	}
	return r.list(`SELECT * FROM subscription_ledger WHERE subscription_id=$1 AND ($2='' OR created_at>=nullif($2,'')::timestamptz) AND ($3='' OR created_at<nullif($3,'')::timestamptz) AND ($4='' OR reason=$4) ORDER BY created_at,id`, r.C.Param("id"), from, to, r.C.Query("reason"))
}

func adjustment(r *R) (any, error) {
	if e := r.fields("delta comment", ""); e != nil {
		return nil, e
	}
	if e := integer(r.Body, "delta", -100000, 100000); e != nil {
		return nil, e
	}
	if num(r.Body, "delta") == 0 {
		return nil, invalid("DELTA")
	}
	if e := validText(r.Body, "comment", 1000); e != nil {
		return nil, e
	}
	s, e := r.get("subscriptions", r.C.Param("id"))
	if e != nil {
		return nil, e
	}
	var reserved int64
	if e = r.DB.QueryRow(r.ctx(), "SELECT count(*) FROM subscription_reservations WHERE subscription_id=$1 AND status='reserved'", s["id"]).Scan(&reserved); e != nil {
		return nil, e
	}
	if num(s, "lessons_remaining")+num(r.Body, "delta") < reserved {
		return nil, conflict("NO_LESSONS_REMAINING")
	}
	entry, e := r.insert("subscription_ledger", M{"subscription_id": s["id"], "delta": r.Body["delta"], "reason": "manual_adjustment", "created_by": r.User["user_id"], "comment": r.Body["comment"]})
	if e != nil {
		return nil, e
	}
	if e = r.audit("subscription.adjusted", "subscriptions", str(s, "id"), s, entry, str(r.Body, "comment")); e != nil {
		return nil, e
	}
	return M{"ledger_entry_id": entry["id"], "lessons_remaining": num(s, "lessons_remaining") + num(r.Body, "delta")}, nil
}

func (r *R) preparePayment(method string) (M, error) {
	inv, e := r.owned("invoices", str(r.Body, "invoice_id"))
	if e != nil {
		return nil, e
	}
	if str(inv, "status") != "open" {
		return nil, conflict("INVOICE_NOT_OPEN")
	}
	pending, e := one(r.ctx(), r.DB, "SELECT * FROM payments WHERE invoice_id=$1 AND status='pending'", inv["id"])
	if e == nil {
		if str(pending, "method") == method && (method == "online" || method == "sbp") {
			return pending, nil
		}
		return nil, conflict("PAYMENT_IN_PROGRESS")
	}
	if ae, ok := e.(*APIError); !ok || ae.Status != 404 {
		return nil, e
	}
	m := M{"invoice_id": inv["id"], "family_id": inv["family_id"], "child_id": inv["child_id"], "amount_minor": inv["amount_minor"], "method": method}
	if method == "online" || method == "sbp" {
		m["provider_key"] = newID()
	}
	if r.Body["comment"] != nil {
		m["comment"] = r.Body["comment"]
	}
	p, e := r.insert("payments", m)
	if e != nil {
		return nil, e
	}
	return p, r.audit("payment.created", "payments", str(p, "id"), nil, paymentPublic(p), "")
}

func paymentIntent(r *R) (any, error) {
	if e := r.fields("invoice_id method", ""); e != nil {
		return nil, e
	}
	method := str(r.Body, "method")
	if !contains([]string{"online", "sbp"}, method) {
		return nil, invalid("PAYMENT_METHOD")
	}
	if r.A.C.ShopID == "" || r.A.C.Secret == "" || r.A.C.ReturnURL == "" {
		return nil, fail(503, "PAYMENT_NOT_CONFIGURED")
	}
	p, e := r.preparePayment(method)
	if e != nil {
		return nil, e
	}
	return paymentPublic(p), nil
}

func paymentPublic(p M) M { m := clean(p, "provider_key"); m["payment_id"] = m["id"]; return m }

func manualPayment(r *R) (any, error) {
	if e := r.fields("invoice_id method", "paid_at comment"); e != nil {
		return nil, e
	}
	method := str(r.Body, "method")
	if !contains([]string{"cash", "transfer"}, method) {
		return nil, invalid("PAYMENT_METHOD")
	}
	paid := r.now()
	if r.Body["paid_at"] != nil {
		t, e := timestamp(str(r.Body, "paid_at"))
		if e != nil || t.After(r.now()) {
			return nil, invalid("PAID_AT")
		}
		paid = t
	}
	p, e := r.preparePayment(method)
	if e != nil {
		return nil, e
	}
	p, e = r.credit(p, paid.Format("2006-01-02T15:04:05.999999999Z07:00"))
	if e != nil {
		return nil, e
	}
	return paymentPublic(p), nil
}

func (r *R) credit(p M, paidAt string) (M, error) {
	if str(p, "status") == "succeeded" {
		return p, nil
	}
	if str(p, "status") == "voided" {
		return nil, conflict("PAYMENT_VOIDED")
	}
	inv, e := r.get("invoices", str(p, "invoice_id"))
	if e != nil {
		return nil, e
	}
	v, e := r.update("payments", str(p, "id"), M{"status": "succeeded", "paid_at": paidAt})
	if e != nil {
		return nil, e
	}
	if _, e = r.insert("money_ledger", M{"family_id": p["family_id"], "invoice_id": p["invoice_id"], "payment_id": p["id"], "amount_minor": p["amount_minor"], "reason": "payment", "source_id": p["id"]}); e != nil {
		return nil, e
	}
	if str(inv, "status") == "open" {
		if _, e = r.update("invoices", str(inv, "id"), M{"status": "paid"}); e != nil {
			return nil, e
		}
	}
	if str(inv, "kind") == "subscription" && !yes(inv, "compensated") {
		var exists bool
		if e = r.DB.QueryRow(r.ctx(), "SELECT EXISTS(SELECT 1 FROM subscriptions WHERE invoice_id=$1)", inv["id"]).Scan(&exists); e != nil {
			return nil, e
		}
		if exists {
			if e = r.exec("UPDATE subscriptions SET suspended_at=NULL,version=version+1 WHERE invoice_id=$1", inv["id"]); e != nil {
				return nil, e
			}
		}
		if !exists {
			plan := object(inv["plan_snapshot"])
			s, e := r.insert("subscriptions", M{"invoice_id": inv["id"], "child_id": inv["child_id"], "family_id": inv["family_id"], "group_id": inv["group_id"], "lessons_total": plan["lessons_total"], "valid_from": inv["valid_from"], "valid_until": inv["valid_until"]})
			if e != nil {
				return nil, e
			}
			if _, e = r.insert("subscription_ledger", M{"subscription_id": s["id"], "delta": plan["lessons_total"], "reason": "subscription_purchase", "payment_id": p["id"], "created_by": nullable(str(r.User, "user_id"))}); e != nil {
				return nil, e
			}
		}
	}
	if str(inv, "kind") == "subscription" {
		if e = r.reserveAwaiting(str(inv, "child_id")); e != nil {
			return nil, e
		}
	}
	if e = r.exec("UPDATE notifications SET read_at=coalesce(read_at,now()) WHERE type='payment.required' AND payload->>'invoice_id'=$1", str(inv, "id")); e != nil {
		return nil, e
	}
	if e = r.notify(str(p, "family_id"), "payment.succeeded", str(p, "id")+":paid", M{"payment_id": p["id"]}); e != nil {
		return nil, e
	}
	if str(inv, "status") != "open" {
		if e = r.exec(`INSERT INTO outbox(user_id,type,payload,event_key) SELECT id,'payment.overpaid',$1,$2||id::text FROM users WHERE role='master' ON CONFLICT(event_key) DO NOTHING`, string(jsonBytes(M{"payment_id": p["id"]})), str(p, "id")+":overpaid: "); e != nil {
			return nil, e
		}
	}
	return v, r.audit("payment.succeeded", "payments", str(p, "id"), paymentPublic(p), paymentPublic(v), "")
}

const paymentsSQL = `SELECT p.id,p.created_at,p.version,p.invoice_id,p.family_id,p.child_id,p.amount_minor,p.currency,p.method,p.status,p.provider_payment_id,p.payment_url,p.paid_at,p.comment,p.voided_at,p.void_reason,(SELECT jsonb_build_object('id',r.id,'kind',r.kind,'external_link',r.external_link,'download_url','/api/v1/receipts/'||r.id||'/download') FROM receipts r WHERE r.payment_id=p.id AND r.deleted_at IS NULL) receipt FROM payments p`

func payments(r *R) (any, error) {
	family := r.C.Query("family_id")
	if str(r.User, "role") == "parent" {
		family = str(r.User, "family_id")
	}
	from, to, e := r.rangeDates(false)
	if e != nil {
		return nil, e
	}
	return r.list(paymentsSQL+` WHERE ($1='' OR p.family_id::text=$1) AND ($2='' OR p.child_id::text=$2) AND ($3='' OR p.status=$3) AND ($4='' OR p.method=$4) AND ($5='' OR p.created_at>=nullif($5,'')::timestamptz) AND ($6='' OR p.created_at<nullif($6,'')::timestamptz) ORDER BY p.created_at,p.id`, family, r.C.Query("child_id"), r.C.Query("status"), r.C.Query("method"), from, to)
}

func paymentGet(r *R) (any, error) {
	if _, e := r.owned("payments", r.C.Param("id")); e != nil {
		return nil, e
	}
	return one(r.ctx(), r.DB, paymentsSQL+" WHERE p.id=$1", r.C.Param("id"))
}

func attendanceHistory(r *R) (any, error) {
	if _, e := r.get("bookings", r.C.Param("bookingId")); e != nil {
		return nil, e
	}
	return r.list(`SELECT id,created_at,before_state AS before,after_state AS after,reason AS comment,actor_user_id AS actor,after_state->'version' AS version FROM audit_log WHERE resource_type='attendance' AND resource_id=$1 ORDER BY created_at,id`, r.C.Param("bookingId"))
}

func auditList(r *R) (any, error) {
	from, to, e := r.rangeDates(false)
	if e != nil {
		return nil, e
	}
	args := []any{r.C.Query("resource_type"), r.C.Query("resource_id"), r.C.Query("actor_user_id"), r.C.Query("event_type"), from, to}
	return r.list(`SELECT id,created_at,event_type,actor_user_id,actor_role,resource_type,resource_id,before_state AS before,after_state AS after,reason,request_id FROM audit_log WHERE ($1='' OR resource_type=$1) AND ($2='' OR resource_id::text=$2) AND ($3='' OR actor_user_id::text=$3) AND ($4='' OR event_type=$4) AND ($5='' OR created_at>=nullif($5,'')::timestamptz) AND ($6='' OR created_at<nullif($6,'')::timestamptz) ORDER BY created_at,id`, args...)
}
