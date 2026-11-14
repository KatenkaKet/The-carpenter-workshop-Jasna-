package app

import (
	"fmt"
	"strings"
	"time"
)

const lessonSQL = `SELECT l.*,g.title group_title,g.is_individual,(SELECT count(*) FROM bookings b WHERE b.lesson_id=l.id AND b.status='active') seats_taken,l.capacity-(SELECT count(*) FROM bookings b WHERE b.lesson_id=l.id AND b.status='active') seats_available FROM lessons l JOIN groups g ON g.id=l.group_id`

func (r *R) rangeDates(required bool) (string, string, error) {
	from, to := r.C.Query("from"), r.C.Query("to")
	if from == "" && to == "" && !required {
		return "", "", nil
	}
	if from == "" || to == "" {
		return "", "", invalid("DATE_RANGE")
	}
	f, e := date(from)
	if e != nil {
		return "", "", e
	}
	t, e := date(to)
	if e != nil || t.Before(f) {
		return "", "", invalid("DATE_RANGE")
	}
	return f.Format(time.RFC3339), t.AddDate(0, 0, 1).Format(time.RFC3339), nil
}

func lessons(r *R) (any, error) {
	public := strings.Contains(r.C.FullPath(), "public")
	from, to, e := r.rangeDates(public)
	if e != nil {
		return nil, e
	}
	if public {
		f, _ := timestamp(from)
		t, _ := timestamp(to)
		if t.Sub(f) > 93*24*time.Hour {
			return nil, invalid("DATE_RANGE_TOO_LONG")
		}
	}
	query := lessonSQL + ` WHERE (NOT $1 OR (g.is_active AND l.status='scheduled')) AND ($2='' OR l.starts_at>=nullif($2,'')::timestamptz) AND ($3='' OR l.starts_at<nullif($3,'')::timestamptz) AND ($4='' OR l.group_id::text=$4) AND ($5='' OR l.status=$5) ORDER BY l.starts_at,l.id`
	v, e := r.list(query, public, from, to, r.C.Query("group_id"), r.C.Query("status"))
	if e != nil {
		return nil, e
	}
	if public {
		for _, m := range object(v)["items"].([]M) {
			m["lesson_id"] = m["id"]
			delete(m, "master_id")
			delete(m, "notes")
			delete(m, "cancel_reason")
		}
	}
	return v, nil
}

func lessonGet(r *R) (any, error) {
	return one(r.ctx(), r.DB, lessonSQL+" WHERE l.id=$1", r.C.Param("id"))
}

func availability(r *R) (any, error) {
	m, e := one(r.ctx(), r.DB, lessonSQL+" WHERE l.id=$1", r.C.Param("id"))
	if e != nil {
		return nil, e
	}
	return M{"lesson_id": m["id"], "status": m["status"], "capacity": m["capacity"], "seats_taken": m["seats_taken"], "seats_available": m["seats_available"], "version": m["version"]}, nil
}

func lessonSave(r *R) (any, error) {
	patch := r.C.Request.Method == "PATCH"
	req, opt := "group_id starts_at ends_at", "capacity notes"
	if patch {
		req, opt = "expected_version", "starts_at ends_at capacity status cancel_reason notes"
	}
	if e := r.fields(req, opt); e != nil {
		return nil, e
	}
	m := M{"status": "scheduled"}
	var old M
	var e error
	if patch {
		old, e = r.get("lessons", r.C.Param("id"))
		if e != nil {
			return nil, e
		}
		if e = r.version(old); e != nil {
			return nil, e
		}
		if str(old, "status") != "scheduled" {
			return nil, conflict("LESSON_FINAL")
		}
		m = clone(old)
	}
	for k, v := range r.Body {
		m[k] = v
	}
	g, e := r.get("groups", str(m, "group_id"))
	if e != nil {
		return nil, e
	}
	if !yes(g, "is_active") {
		return nil, conflict("GROUP_INACTIVE")
	}
	if m["capacity"] == nil {
		m["capacity"] = g["capacity"]
	}
	if e = integer(m, "capacity", 1, 1000); e != nil {
		return nil, e
	}
	if yes(g, "is_individual") && num(m, "capacity") != 1 {
		return nil, invalid("INDIVIDUAL_CAPACITY")
	}
	start, e := timestamp(str(m, "starts_at"))
	if e != nil {
		return nil, e
	}
	end, e := timestamp(str(m, "ends_at"))
	if e != nil || !end.After(start) {
		return nil, invalid("LESSON_TIME")
	}
	if !patch && !start.After(r.now()) {
		return nil, invalid("LESSON_IN_PAST")
	}
	if patch && (r.Body["starts_at"] != nil || r.Body["ends_at"] != nil) && !start.After(r.now()) {
		return nil, invalid("LESSON_IN_PAST")
	}
	status := str(m, "status")
	if !contains([]string{"scheduled", "completed", "cancelled"}, status) {
		return nil, invalid("STATUS")
	}
	var n int
	if status == "scheduled" {
		if e = r.DB.QueryRow(r.ctx(), `SELECT count(*) FROM lessons WHERE master_id=$1 AND status='scheduled' AND id::text<>$2 AND starts_at<$4 AND ends_at>$3`, g["master_id"], str(old, "id"), start, end).Scan(&n); e != nil {
			return nil, e
		}
		if n > 0 {
			return nil, conflict("MASTER_TIME_CONFLICT")
		}
	}
	if patch {
		if e = r.DB.QueryRow(r.ctx(), "SELECT count(*) FROM bookings WHERE lesson_id=$1 AND status='active'", old["id"]).Scan(&n); e != nil {
			return nil, e
		}
		if int64(n) > num(m, "capacity") {
			return nil, conflict("CAPACITY_BELOW_BOOKINGS")
		}
		if status == "completed" {
			if end.After(r.now()) {
				return nil, conflict("LESSON_NOT_ENDED")
			}
			if e = r.DB.QueryRow(r.ctx(), `SELECT count(*) FROM attendance a JOIN bookings b ON b.id=a.booking_id WHERE b.lesson_id=$1 AND b.status='active' AND a.status='pending'`, old["id"]).Scan(&n); e != nil {
				return nil, e
			}
			if n > 0 {
				return nil, conflict("PENDING_ATTENDANCE")
			}
		}
		if status == "scheduled" {
			if e = r.DB.QueryRow(r.ctx(), `SELECT count(*) FROM bookings b JOIN bookings other ON other.child_id=b.child_id AND other.lesson_id<>b.lesson_id JOIN lessons l ON l.id=other.lesson_id WHERE b.lesson_id=$1 AND b.status='active' AND other.status='active' AND l.status='scheduled' AND l.starts_at<$3 AND l.ends_at>$2`, old["id"], start, end).Scan(&n); e != nil {
				return nil, e
			}
			if n > 0 {
				return nil, conflict("CHILD_TIME_CONFLICT")
			}
		}
	}
	if status == "cancelled" {
		if e = validText(m, "cancel_reason", 1000); e != nil {
			return nil, e
		}
	}
	var v M
	if patch {
		v, e = r.update("lessons", str(old, "id"), clean(r.Body, "expected_version"))
	} else {
		m["master_id"] = g["master_id"]
		v, e = r.insert("lessons", m)
	}
	if e != nil {
		return nil, e
	}
	if patch {
		bs, e := many(r.ctx(), r.DB, "SELECT * FROM bookings WHERE lesson_id=$1 ORDER BY id", v["id"])
		if e != nil {
			return nil, e
		}
		for _, b := range bs {
			if status == "cancelled" {
				if _, e = r.update("bookings", str(b, "id"), M{"status": "cancelled_by_master", "cancelled_at": r.now(), "cancelled_by": r.User["user_id"]}); e != nil {
					return nil, e
				}
				if _, e = r.mark(b, v, "cancelled", "master", "Отмена занятия мастером"); e != nil {
					return nil, e
				}
			}
			typ := "lesson.changed"
			if status == "cancelled" {
				typ = "lesson.cancelled"
			}
			if e = r.notify(str(b, "family_id"), typ, fmt.Sprint(v["id"], ":", v["version"], ":", b["family_id"]), M{"lesson_id": v["id"]}); e != nil {
				return nil, e
			}
		}
	}
	return v, r.audit("lesson.saved", "lessons", str(v, "id"), old, v, "")
}

func (r *R) book(lessonID, childID string) (M, error) {
	child, e := r.owned("children", childID)
	if e != nil {
		return nil, e
	}
	if !yes(child, "is_active") {
		return nil, conflict("CHILD_INACTIVE")
	}
	l, e := r.get("lessons", lessonID)
	if e != nil {
		return nil, e
	}
	if str(l, "status") != "scheduled" || !at(l, "starts_at").After(r.now()) {
		return nil, conflict("LESSON_UNAVAILABLE")
	}
	var n int
	if e = r.DB.QueryRow(r.ctx(), "SELECT count(*) FROM bookings WHERE child_id=$1 AND lesson_id=$2 AND status='active'", childID, lessonID).Scan(&n); e != nil {
		return nil, e
	}
	if n > 0 {
		return nil, conflict("DUPLICATE_BOOKING")
	}
	if e = r.DB.QueryRow(r.ctx(), "SELECT count(*) FROM bookings WHERE lesson_id=$1 AND status='active'", lessonID).Scan(&n); e != nil {
		return nil, e
	}
	if int64(n) >= num(l, "capacity") {
		return nil, conflict("SEAT_LIMIT_EXCEEDED")
	}
	if e = r.DB.QueryRow(r.ctx(), `SELECT count(*) FROM bookings b JOIN lessons l ON l.id=b.lesson_id WHERE b.child_id=$1 AND b.status='active' AND l.status='scheduled' AND l.starts_at<$3 AND l.ends_at>$2`, childID, at(l, "starts_at"), at(l, "ends_at")).Scan(&n); e != nil {
		return nil, e
	}
	if n > 0 {
		return nil, conflict("CHILD_TIME_CONFLICT")
	}
	rule, e := r.rule(str(l, "group_id"), at(l, "starts_at").In(zone).Format("2006-01-02"))
	if e != nil {
		return nil, e
	}
	b, e := r.insert("bookings", M{"lesson_id": lessonID, "child_id": childID, "family_id": child["family_id"], "rule_snapshot": rule, "rule_id": rule["id"]})
	if e != nil {
		return nil, e
	}
	if _, e = r.insert("attendance", M{"booking_id": b["id"]}); e != nil {
		return nil, e
	}
	if e = r.notify(str(child, "family_id"), "booking.confirmed", str(b, "id")+":confirmed", M{"booking_id": b["id"]}); e != nil {
		return nil, e
	}
	return b, r.audit("booking.created", "bookings", str(b, "id"), nil, b, "")
}

func bookingCreate(r *R) (any, error) {
	if e := r.fields("lesson_id child_id", ""); e != nil {
		return nil, e
	}
	return r.book(str(r.Body, "lesson_id"), str(r.Body, "child_id"))
}

func (r *R) cancelBooking(b, l M, reason string) (M, error) {
	rule := object(b["rule_snapshot"])
	hours := at(l, "starts_at").Sub(r.now()).Hours()
	if reason == "parent" && (!yes(rule, "parent_cancel_allowed") || hours < float64(num(rule, "cancel_deadline_hours"))) {
		return nil, conflict("CANCELLATION_DEADLINE")
	}
	status := "cancelled_by_parent"
	if str(r.User, "role") == "master" {
		status = "cancelled_by_master"
	}
	v, e := r.update("bookings", str(b, "id"), M{"status": status, "cancelled_at": r.now(), "cancelled_by": r.User["user_id"]})
	if e != nil {
		return nil, e
	}
	if _, e = r.mark(v, l, "cancelled", reason, "Отмена записи"); e != nil {
		return nil, e
	}
	if e = r.notify(str(b, "family_id"), "booking.cancelled", str(b, "id")+":cancelled", M{"booking_id": b["id"]}); e != nil {
		return nil, e
	}
	return v, r.audit("booking.cancelled", "bookings", str(b, "id"), b, v, "")
}

func bookingCancel(r *R) (any, error) {
	b, e := r.owned("bookings", r.C.Param("id"))
	if e != nil {
		return nil, e
	}
	if str(b, "status") != "active" {
		return nil, nil
	}
	l, e := r.get("lessons", str(b, "lesson_id"))
	if e != nil {
		return nil, e
	}
	if !at(l, "starts_at").After(r.now()) {
		return nil, conflict("LESSON_IN_PAST")
	}
	reason := str(r.User, "role")
	_, e = r.cancelBooking(b, l, reason)
	return nil, e
}

func transfer(r *R) (any, error) {
	if e := r.fields("target_lesson_id", ""); e != nil {
		return nil, e
	}
	b, e := r.owned("bookings", r.C.Param("id"))
	if e != nil {
		return nil, e
	}
	if str(b, "status") != "active" {
		return nil, conflict("BOOKING_CANCELLED")
	}
	l, e := r.get("lessons", str(b, "lesson_id"))
	if e != nil {
		return nil, e
	}
	target, e := r.get("lessons", str(r.Body, "target_lesson_id"))
	if e != nil {
		return nil, e
	}
	rule := object(b["rule_snapshot"])
	if l["group_id"] != target["group_id"] || l["id"] == target["id"] {
		return nil, invalid("TRANSFER_TARGET")
	}
	if !yes(rule, "transfer_allowed") || at(l, "starts_at").Sub(r.now()).Hours() < float64(num(rule, "transfer_deadline_hours")) || !at(l, "starts_at").After(r.now()) {
		return nil, conflict("TRANSFER_DEADLINE")
	}
	if _, e = r.cancelBooking(b, l, "transfer"); e != nil {
		return nil, e
	}
	next, e := r.book(str(target, "id"), str(b, "child_id"))
	if e != nil {
		return nil, e
	}
	if e = r.notify(str(b, "family_id"), "booking.transferred", str(next, "id")+":transfer", M{"old_booking_id": b["id"], "new_booking_id": next["id"]}); e != nil {
		return nil, e
	}
	return M{"old_booking_id": b["id"], "new_booking_id": next["id"], "charge_applied": yes(rule, "transfer_charge_lesson")}, nil
}

const bookingsSQL = `SELECT b.*,l.starts_at,l.ends_at,g.title group_title,c.full_name child_full_name,to_jsonb(a) attendance FROM bookings b JOIN lessons l ON l.id=b.lesson_id JOIN groups g ON g.id=l.group_id JOIN children c ON c.id=b.child_id JOIN attendance a ON a.booking_id=b.id`

func bookings(r *R) (any, error) {
	from, to, e := r.rangeDates(false)
	if e != nil {
		return nil, e
	}
	family := ""
	if str(r.User, "role") == "parent" {
		family = str(r.User, "family_id")
	}
	return r.list(bookingsSQL+` WHERE ($1='' OR b.family_id::text=$1) AND ($2='' OR b.lesson_id::text=$2) AND ($3='' OR b.child_id::text=$3) AND ($4='' OR $4='all' OR b.status=$4) AND ($5='' OR l.starts_at>=nullif($5,'')::timestamptz) AND ($6='' OR l.starts_at<nullif($6,'')::timestamptz) AND ($7<>'true' OR l.starts_at>now()) ORDER BY l.starts_at,b.id`, family, r.C.Param("id"), r.C.Query("child_id"), r.C.Query("status"), from, to, r.C.Query("upcoming"))
}

func attendanceList(r *R) (any, error) {
	from, to, e := r.rangeDates(false)
	if e != nil {
		return nil, e
	}
	return r.list(`SELECT a.*,b.child_id,b.lesson_id,l.starts_at FROM attendance a JOIN bookings b ON b.id=a.booking_id JOIN lessons l ON l.id=b.lesson_id WHERE b.family_id=$1 AND ($2='' OR b.child_id::text=$2) AND ($3='' OR l.starts_at>=nullif($3,'')::timestamptz) AND ($4='' OR l.starts_at<nullif($4,'')::timestamptz) ORDER BY l.starts_at,a.id`, r.User["family_id"], r.C.Query("child_id"), from, to)
}

func attendanceSave(r *R) (any, error) {
	if e := r.fields("status expected_version", "cancel_reason comment"); e != nil {
		return nil, e
	}
	b, e := r.get("bookings", r.C.Param("bookingId"))
	if e != nil {
		return nil, e
	}
	if str(b, "lesson_id") != r.C.Param("id") {
		return nil, fail(404, "NOT_FOUND")
	}
	a, e := one(r.ctx(), r.DB, "SELECT * FROM attendance WHERE booking_id=$1", b["id"])
	if e != nil {
		return nil, e
	}
	if e = r.version(a); e != nil {
		return nil, e
	}
	l, e := r.get("lessons", str(b, "lesson_id"))
	if e != nil {
		return nil, e
	}
	status := str(r.Body, "status")
	reason := str(r.Body, "cancel_reason")
	if str(l, "status") == "cancelled" && (status != "cancelled" || reason != "master") {
		return nil, conflict("LESSON_CANCELLED")
	}
	if str(a, "status") != "pending" && (str(a, "status") != status || str(a, "cancel_reason") != reason) && strings.TrimSpace(str(r.Body, "comment")) == "" {
		return nil, invalid("COMMENT_REQUIRED")
	}
	return r.mark(b, l, status, reason, str(r.Body, "comment"))
}

func chargeFor(rule M, status, reason string, start, cancel time.Time) bool {
	switch status {
	case "attended":
		return yes(rule, "attended_charge_lesson")
	case "missed":
		return yes(rule, "missed_charge_lesson")
	case "cancelled":
		switch reason {
		case "master":
			return false
		case "transfer":
			return yes(rule, "transfer_charge_lesson")
		case "parent":
			return start.Sub(cancel).Hours() < float64(num(rule, "free_cancel_before_hours")) && yes(rule, "parent_cancel_charge_lesson")
		}
	}
	return false
}

func (r *R) mark(b, l M, status, reason, comment string) (M, error) {
	if !contains([]string{"pending", "attended", "missed", "cancelled"}, status) {
		return nil, invalid("ATTENDANCE_STATUS")
	}
	if status != "cancelled" && reason != "" {
		return nil, invalid("CANCEL_REASON_NOT_ALLOWED")
	}
	if status == "cancelled" && !contains([]string{"parent", "master", "transfer"}, reason) {
		return nil, invalid("CANCEL_REASON")
	}
	if (status == "attended" || status == "missed") && at(l, "starts_at").After(r.now()) {
		return nil, conflict("LESSON_NOT_STARTED")
	}
	old, e := one(r.ctx(), r.DB, "SELECT * FROM attendance WHERE booking_id=$1", b["id"])
	if e != nil {
		return nil, e
	}
	if str(old, "status") == status && str(old, "cancel_reason") == reason {
		return r.attendanceResult(old)
	}
	rule := object(b["rule_snapshot"])
	cancel := at(b, "cancelled_at")
	if cancel.IsZero() {
		cancel = r.now()
	}
	want := chargeFor(rule, status, reason, at(l, "starts_at"), cancel)
	changes := M{"status": status, "cancel_reason": nullable(reason), "comment": comment, "marked_at": r.now(), "marked_by": r.User["user_id"], "charged": want}

	// A prepaid single lesson can be canceled before it ever receives an attendance mark.
	// Its invoice is a financial effect even while attendance.charged is still false.
	if !want && !yes(old, "charged") && status == "cancelled" {
		inv, err := one(r.ctx(), r.DB, "SELECT * FROM invoices WHERE booking_id=$1", b["id"])
		if err == nil {
			if e = r.compensateInvoice(str(inv, "id"), true); e != nil {
				return nil, e
			}
			changes["invoice_id"] = inv["id"]
		} else if ae, ok := err.(*APIError); !ok || ae.Status != 404 {
			return nil, err
		}
	}
	if want != yes(old, "charged") {
		if !want {
			if id := str(old, "subscription_id"); id != "" {
				_, e := r.insert("subscription_ledger", M{"subscription_id": id, "delta": 1, "reason": "attendance_reversal", "attendance_id": old["id"], "reverses_entry_id": old["charge_entry_id"], "created_by": r.User["user_id"], "comment": comment})
				if e != nil {
					return nil, e
				}
				changes["charge_entry_id"] = nil
			} else if id := str(old, "invoice_id"); id != "" {
				if e = r.compensateInvoice(id, true); e != nil {
					return nil, e
				}
			}
		} else {
			// Existing single-lesson invoice always wins, including a paid compensated invoice.
			inv, err := one(r.ctx(), r.DB, "SELECT * FROM invoices WHERE booking_id=$1", b["id"])
			if err == nil {
				if yes(inv, "compensated") {
					if e = r.compensateInvoice(str(inv, "id"), false); e != nil {
						return nil, e
					}
				}
				changes["invoice_id"] = inv["id"]
				changes["subscription_id"] = nil
				changes["charge_entry_id"] = nil
			} else {
				if ae, ok := err.(*APIError); !ok || ae.Status != 404 {
					return nil, err
				}
				day := at(l, "starts_at").In(zone).Format("2006-01-02")
				sub, err := one(r.ctx(), r.DB, `SELECT * FROM subscriptions WHERE child_id=$1 AND group_id=$2 AND valid_from<=$3::date AND valid_until>=$3::date AND lessons_remaining>0 ORDER BY valid_until,id LIMIT 1`, b["child_id"], l["group_id"], day)
				if err == nil {
					entry, e := r.insert("subscription_ledger", M{"subscription_id": sub["id"], "delta": -1, "reason": "attendance_charge", "attendance_id": old["id"], "created_by": r.User["user_id"], "comment": comment})
					if e != nil {
						return nil, e
					}
					changes["subscription_id"] = sub["id"]
					changes["charge_entry_id"] = entry["id"]
					changes["invoice_id"] = nil
				} else {
					if ae, ok := err.(*APIError); !ok || ae.Status != 404 {
						return nil, err
					}
					inv, e := r.singleInvoice(b, l)
					if e != nil {
						return nil, e
					}
					changes["invoice_id"] = inv["id"]
					changes["subscription_id"] = nil
					changes["charge_entry_id"] = nil
				}
			}
		}
	}
	v, e := r.update("attendance", str(old, "id"), changes)
	if e != nil {
		return nil, e
	}
	if e = r.audit("attendance.corrected", "attendance", str(b, "id"), old, v, comment); e != nil {
		return nil, e
	}
	return r.attendanceResult(v)
}

func (r *R) attendanceResult(v M) (M, error) {
	v["attendance_id"] = v["id"]
	if str(v, "subscription_id") != "" {
		sub, e := r.get("subscriptions", str(v, "subscription_id"))
		if e != nil {
			return nil, e
		}
		v["lessons_remaining"] = sub["lessons_remaining"]
	}
	return v, nil
}
