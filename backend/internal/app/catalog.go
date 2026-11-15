package app

import (
	"fmt"
	"strings"
	"time"
)

func clone(m M) M {
	n := M{}
	for k, v := range m {
		n[k] = v
	}
	return n
}

func clean(m M, keys ...string) M {
	n := clone(m)
	for _, k := range keys {
		delete(n, k)
	}
	return n
}

func validText(m M, k string, max int) error {
	s, ok := m[k].(string)
	if !ok || strings.TrimSpace(s) == "" || len([]rune(s)) > max {
		return invalid(k)
	}
	return nil
}

func integer(m M, k string, min, max int64) error {
	n := num(m, k)
	if n < min || n > max {
		return invalid(k)
	}
	if fmt.Sprint(m[k]) != fmt.Sprint(n) {
		return invalid(k)
	}
	return nil
}

func boolean(m M, k string) error {
	if _, ok := m[k].(bool); !ok {
		return invalid(k)
	}
	return nil
}

func familyGet(r *R) (any, error) {
	id := r.C.Param("id")
	if id == "" {
		id = str(r.User, "family_id")
	}
	f, e := one(r.ctx(), r.DB, `SELECT f.*,jsonb_build_object('id',u.id,'full_name',u.full_name,'email',u.email,'phone',u.phone) parent FROM families f JOIN users u ON u.id=f.parent_user_id WHERE f.id=$1`, id)
	if e != nil {
		return nil, e
	}
	kids, e := many(r.ctx(), r.DB, "SELECT * FROM children WHERE family_id=$1 ORDER BY created_at,id", id)
	if e != nil {
		return nil, e
	}
	f["children"] = kids
	f["financial_summary"], e = r.balance(id)
	return f, e
}

func families(r *R) (any, error) {
	q := "%" + r.C.Query("search") + "%"
	return r.list(`SELECT f.*,u.full_name parent_full_name,u.email,u.phone,(SELECT count(*) FROM children c WHERE c.family_id=f.id) children_count FROM families f JOIN users u ON u.id=f.parent_user_id WHERE u.full_name ILIKE $1 OR coalesce(u.email,'') ILIKE $1 ORDER BY f.created_at,f.id`, q)
}

func familyCreate(r *R) (any, error) {
	if e := r.fields("parent_full_name children", "phone email"); e != nil {
		return nil, e
	}
	if e := validText(r.Body, "parent_full_name", 200); e != nil {
		return nil, e
	}
	kids, ok := r.Body["children"].([]any)
	if !ok || len(kids) < 1 || len(kids) > 30 {
		return nil, invalid("CHILDREN")
	}
	u, e := r.insert("users", M{"role": "parent", "full_name": r.Body["parent_full_name"], "phone": r.Body["phone"], "email": r.Body["email"]})
	if e != nil {
		return nil, e
	}
	f, e := r.insert("families", M{"parent_user_id": u["id"]})
	if e != nil {
		return nil, e
	}
	out := []M{}
	for _, v := range kids {
		m := object(v)
		for k := range m {
			if k != "full_name" && k != "birth_date" {
				return nil, invalid("CHILD_FIELD")
			}
		}
		if e = validateChild(m); e != nil {
			return nil, e
		}
		m["family_id"] = f["id"]
		child, e := r.insert("children", m)
		if e != nil {
			return nil, e
		}
		out = append(out, child)
	}
	f["children"] = out
	f["parent"] = clean(u, "password_hash", "login")
	return f, r.audit("family.created", "families", str(f, "id"), nil, f, "")
}

func familyPatch(r *R) (any, error) {
	if e := r.fields("expected_version", "parent_full_name phone email"); e != nil {
		return nil, e
	}
	f, e := r.get("families", r.C.Param("id"))
	if e != nil {
		return nil, e
	}
	if e = r.version(f); e != nil {
		return nil, e
	}
	u := M{}
	for _, k := range []string{"phone", "email"} {
		if v, ok := r.Body[k]; ok {
			u[k] = v
		}
	}
	if v, ok := r.Body["parent_full_name"]; ok {
		if e = validText(r.Body, "parent_full_name", 200); e != nil {
			return nil, e
		}
		u["full_name"] = v
	}
	if _, e = r.update("users", str(f, "parent_user_id"), u); e != nil {
		return nil, e
	}
	f2, e := r.update("families", str(f, "id"), M{})
	if e != nil {
		return nil, e
	}
	if e = r.audit("family.updated", "families", str(f, "id"), f, f2, ""); e != nil {
		return nil, e
	}
	return familyGet(r)
}

func validateChild(m M) error {
	if e := validText(m, "full_name", 200); e != nil {
		return e
	}
	if m["birth_date"] != nil {
		t, e := date(str(m, "birth_date"))
		if e != nil || t.After(time.Now()) {
			return invalid("BIRTH_DATE")
		}
	}
	return nil
}

func childCreate(r *R) (any, error) {
	if e := r.fields("full_name", "birth_date"); e != nil {
		return nil, e
	}
	if e := validateChild(r.Body); e != nil {
		return nil, e
	}
	m := clone(r.Body)
	m["family_id"] = r.C.Param("id")
	v, e := r.insert("children", m)
	if e != nil {
		return nil, e
	}
	return v, r.audit("child.created", "children", str(v, "id"), nil, v, "")
}

func childPatch(r *R) (any, error) {
	if e := r.fields("expected_version", "full_name birth_date is_active"); e != nil {
		return nil, e
	}
	old, e := r.get("children", r.C.Param("id"))
	if e != nil {
		return nil, e
	}
	if e = r.version(old); e != nil {
		return nil, e
	}
	m := clone(old)
	for k, v := range r.Body {
		m[k] = v
	}
	if e = validateChild(m); e != nil {
		return nil, e
	}
	if e = boolean(m, "is_active"); e != nil {
		return nil, e
	}
	if !yes(m, "is_active") {
		var n int
		e = r.DB.QueryRow(r.ctx(), `SELECT count(*) FROM bookings b JOIN lessons l ON l.id=b.lesson_id WHERE b.child_id=$1 AND b.status='active' AND l.starts_at>now()`, old["id"]).Scan(&n)
		if e != nil {
			return nil, e
		}
		if n > 0 {
			return nil, conflict("ACTIVE_BOOKINGS")
		}
	}
	v, e := r.update("children", str(old, "id"), clean(r.Body, "expected_version"))
	if e != nil {
		return nil, e
	}
	return v, r.audit("child.updated", "children", str(v, "id"), old, v, "")
}

func children(r *R) (any, error) {
	return r.list(`SELECT c.* FROM children c WHERE ($1='' OR c.family_id::text=$1) AND c.full_name ILIKE $2 ORDER BY c.created_at,c.id`, r.C.Query("family_id"), "%"+r.C.Query("search")+"%")
}

func groups(r *R) (any, error) {
	if strings.Contains(r.C.FullPath(), "/public/") {
		return r.list(`SELECT id,title,description,capacity,is_individual FROM groups WHERE is_active AND ($1='' OR is_individual::text=$1) ORDER BY created_at,id`, r.C.Query("is_individual"))
	}
	return r.list(`SELECT * FROM groups WHERE ($1='' OR is_active::text=$1) AND title ILIKE $2 ORDER BY created_at,id`, r.C.Query("is_active"), "%"+r.C.Query("search")+"%")
}

func groupGet(r *R) (any, error) {
	return one(r.ctx(), r.DB, "SELECT id,title,description,capacity,is_individual FROM groups WHERE id=$1 AND is_active", r.C.Param("id"))
}

func groupSave(r *R) (any, error) {
	patch := r.C.Request.Method == "PATCH"
	required, optional := "title capacity", "is_individual description"
	if patch {
		required, optional = "expected_version", "title capacity description is_active"
	}
	if e := r.fields(required, optional); e != nil {
		return nil, e
	}
	m := M{"is_individual": false, "is_active": true}
	var old M
	var e error
	if patch {
		old, e = r.get("groups", r.C.Param("id"))
		if e != nil {
			return nil, e
		}
		if e = r.version(old); e != nil {
			return nil, e
		}
		m = clone(old)
	}
	for k, v := range r.Body {
		m[k] = v
	}
	if e = validText(m, "title", 200); e != nil {
		return nil, e
	}
	if e = integer(m, "capacity", 1, 1000); e != nil {
		return nil, e
	}
	for _, k := range []string{"is_individual", "is_active"} {
		if e = boolean(m, k); e != nil {
			return nil, e
		}
	}
	if yes(m, "is_individual") && num(m, "capacity") != 1 {
		return nil, invalid("INDIVIDUAL_CAPACITY")
	}
	if patch && !yes(m, "is_active") {
		var n int
		e = r.DB.QueryRow(r.ctx(), "SELECT count(*) FROM lessons WHERE group_id=$1 AND status='scheduled' AND starts_at>now()", old["id"]).Scan(&n)
		if e != nil {
			return nil, e
		}
		if n > 0 {
			return nil, conflict("FUTURE_LESSONS")
		}
	}
	var v M
	if patch {
		v, e = r.update("groups", str(old, "id"), clean(r.Body, "expected_version"))
	} else {
		m["master_id"] = r.User["user_id"]
		v, e = r.insert("groups", m)
	}
	if e != nil {
		return nil, e
	}
	return v, r.audit("group.saved", "groups", str(v, "id"), old, v, "")
}

func planSave(r *R) (any, error) {
	patch := r.C.Request.Method == "PATCH"
	req, opt := "title lessons_total validity_days price_minor", "group_id"
	if patch {
		req, opt = "expected_version", "title lessons_total validity_days price_minor is_active"
	}
	if e := r.fields(req, opt); e != nil {
		return nil, e
	}
	var old M
	var e error
	m := M{"is_active": true}
	if patch {
		old, e = r.get("plans", r.C.Param("id"))
		if e != nil {
			return nil, e
		}
		if e = r.version(old); e != nil {
			return nil, e
		}
		m = clone(old)
	}
	for k, v := range r.Body {
		m[k] = v
	}
	if e = validText(m, "title", 200); e != nil {
		return nil, e
	}
	for _, k := range []string{"lessons_total", "validity_days", "price_minor"} {
		if e = integer(m, k, 1, 100000000); e != nil {
			return nil, e
		}
	}
	if e = boolean(m, "is_active"); e != nil {
		return nil, e
	}
	var v M
	if patch {
		v, e = r.update("plans", str(old, "id"), clean(r.Body, "expected_version"))
	} else {
		v, e = r.insert("plans", m)
	}
	if e != nil {
		return nil, e
	}
	return v, r.audit("plan.saved", "plans", str(v, "id"), old, v, "")
}

func plans(r *R) (any, error) {
	public := strings.Contains(r.C.FullPath(), "/public/")
	return r.list("SELECT * FROM plans WHERE (NOT $1 OR is_active) AND ($2='' OR group_id::text=$2) ORDER BY created_at,id", public, r.C.Query("group_id"))
}

const ruleFields = "group_id valid_from attended_charge_lesson missed_charge_lesson parent_cancel_charge_lesson free_cancel_before_hours parent_cancel_allowed cancel_deadline_hours transfer_allowed transfer_deadline_hours transfer_charge_lesson single_lesson_price_minor ending_notice_days ending_notice_lessons"

func ruleSave(r *R) (any, error) {
	patch := r.C.Request.Method == "PATCH"
	req, opt := strings.ReplaceAll(ruleFields, "group_id ", ""), "group_id valid_to"
	if patch {
		req, opt = "expected_version", ruleFields+" valid_to"
	}
	if e := r.fields(req, opt); e != nil {
		return nil, e
	}
	var old M
	var e error
	m := clone(r.Body)
	if patch {
		old, e = r.get("attendance_rules", r.C.Param("id"))
		if e != nil {
			return nil, e
		}
		if e = r.version(old); e != nil {
			return nil, e
		}
		var used bool
		if e = r.DB.QueryRow(r.ctx(), "SELECT EXISTS(SELECT 1 FROM bookings WHERE rule_id=$1)", old["id"]).Scan(&used); e != nil {
			return nil, e
		}
		if used {
			return nil, conflict("RULE_IN_USE")
		}
		m = clone(old)
		for k, v := range r.Body {
			m[k] = v
		}
	}
	from, e := date(str(m, "valid_from"))
	if e != nil {
		return nil, e
	}
	if m["valid_to"] != nil {
		to, e := date(str(m, "valid_to"))
		if e != nil || to.Before(from) {
			return nil, invalid("RULE_DATES")
		}
	}
	for _, k := range []string{"attended_charge_lesson", "missed_charge_lesson", "parent_cancel_charge_lesson", "parent_cancel_allowed", "transfer_allowed", "transfer_charge_lesson"} {
		if e = boolean(m, k); e != nil {
			return nil, e
		}
	}
	for _, k := range []string{"free_cancel_before_hours", "cancel_deadline_hours", "transfer_deadline_hours", "ending_notice_days", "ending_notice_lessons"} {
		if e = integer(m, k, 0, 100000); e != nil {
			return nil, e
		}
	}
	if e = integer(m, "single_lesson_price_minor", 1, 100000000); e != nil {
		return nil, e
	}
	// A successor can close the preceding open period, without altering booking snapshots.
	if !patch && from.After(r.now().In(zone)) {
		if e = r.exec(`UPDATE attendance_rules SET valid_to=$2::date-1,version=version+1 WHERE group_id IS NOT DISTINCT FROM $1::uuid AND valid_to IS NULL AND valid_from<$2::date`, m["group_id"], str(m, "valid_from")); e != nil {
			return nil, e
		}
	}
	var overlap bool
	if e = r.DB.QueryRow(r.ctx(), `SELECT EXISTS(SELECT 1 FROM attendance_rules WHERE group_id IS NOT DISTINCT FROM $1::uuid AND id::text<>$4 AND daterange(valid_from,valid_to,'[]') && daterange($2::date,$3::date,'[]'))`, m["group_id"], str(m, "valid_from"), m["valid_to"], str(old, "id")).Scan(&overlap); e != nil {
		return nil, e
	}
	if overlap {
		return nil, conflict("RULE_PERIOD_OVERLAP")
	}
	var v M
	if patch {
		v, e = r.update("attendance_rules", str(old, "id"), clean(r.Body, "expected_version"))
	} else {
		v, e = r.insert("attendance_rules", m)
	}
	if e != nil {
		return nil, e
	}
	return v, r.audit("rule.saved", "attendance_rules", str(v, "id"), old, v, "")
}

func rules(r *R) (any, error) {
	return r.list(`SELECT * FROM attendance_rules WHERE ($1='' OR coalesce(group_id::text,'null')=$1) AND ($2='' OR (valid_from<=nullif($2,'')::date AND (valid_to IS NULL OR valid_to>=nullif($2,'')::date))) ORDER BY created_at,id`, r.C.Query("group_id"), r.C.Query("date"))
}

func (r *R) rule(group, day string) (M, error) {
	m, e := one(r.ctx(), r.DB, `SELECT * FROM attendance_rules WHERE (group_id=$1 OR group_id IS NULL) AND valid_from<=$2::date AND (valid_to IS NULL OR valid_to>=$2::date) ORDER BY group_id NULLS LAST LIMIT 1`, nullable(group), day)
	if e != nil {
		if ae, ok := e.(*APIError); ok && ae.Status == 404 {
			return nil, conflict("RULES_NOT_CONFIGURED")
		}
	}
	return m, e
}

func publicRules(r *R) (any, error) {
	if _, e := date(r.C.Query("date")); e != nil {
		return nil, e
	}
	return r.rule(r.C.Query("group_id"), r.C.Query("date"))
}

func ruleDelete(r *R) (any, error) {
	m, e := r.get("attendance_rules", r.C.Param("id"))
	if e != nil {
		return nil, e
	}
	var used bool
	if e = r.DB.QueryRow(r.ctx(), "SELECT EXISTS(SELECT 1 FROM bookings WHERE rule_id=$1)", m["id"]).Scan(&used); e != nil {
		return nil, e
	}
	if used {
		return nil, conflict("RULE_IN_USE")
	}
	if e = r.exec("DELETE FROM attendance_rules WHERE id=$1", m["id"]); e != nil {
		return nil, e
	}
	return nil, r.audit("rule.deleted", "attendance_rules", str(m, "id"), m, nil, "")
}
