package app

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const receiptSQL = `SELECT id,payment_id,kind,external_link,mime_type,file_name,uploaded_by,uploaded_at,version,deleted_at,CASE WHEN kind='file' THEN '/api/v1/receipts/'||id||'/download' END download_url FROM receipts`

func (r *R) receipt(id string) (M, error) {
	m, e := one(r.ctx(), r.DB, receiptSQL+" WHERE id=$1 AND deleted_at IS NULL", id)
	if e != nil {
		return nil, e
	}
	if _, e = r.owned("payments", str(m, "payment_id")); e != nil {
		return nil, e
	}
	return m, nil
}

func receiptGet(r *R) (any, error) { return r.receipt(r.C.Param("id")) }

func receiptDownload(r *R) (any, error) {
	m, e := r.receipt(r.C.Param("id"))
	if e != nil {
		return nil, e
	}
	if str(m, "kind") != "file" {
		return nil, conflict("RECEIPT_IS_EXTERNAL")
	}
	var data []byte
	if e = r.DB.QueryRow(r.ctx(), "SELECT content FROM receipts WHERE id=$1", m["id"]).Scan(&data); e != nil {
		return nil, e
	}
	r.C.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": str(m, "file_name")}))
	r.C.Data(200, str(m, "mime_type"), data)
	return nil, nil
}

func receiptSave(r *R) (any, error) {
	p, e := r.get("payments", r.C.Param("id"))
	if e != nil {
		return nil, e
	}
	if str(p, "status") != "succeeded" {
		return nil, conflict("PAYMENT_NOT_SUCCEEDED")
	}
	if e = r.C.Request.ParseMultipartForm(1 << 20); e != nil {
		return nil, invalid("MULTIPART")
	}
	defer r.C.Request.MultipartForm.RemoveAll()
	form := r.C.Request.MultipartForm
	for k := range form.Value {
		if !contains([]string{"external_link", "comment", "expected_version"}, k) || len(form.Value[k]) != 1 {
			return nil, invalid("MULTIPART_FIELD")
		}
	}
	for k := range form.File {
		if k != "file" || len(form.File[k]) != 1 {
			return nil, invalid("MULTIPART_FILE")
		}
	}
	link := r.C.PostForm("external_link")
	files := form.File["file"]
	if (link == "") == (len(files) == 0) {
		return nil, invalid("ONE_RECEIPT_SOURCE_REQUIRED")
	}
	m := M{"payment_id": p["id"], "uploaded_by": r.User["user_id"], "uploaded_at": r.now(), "external_link": nil, "content": nil, "mime_type": nil, "file_name": nil}
	if link != "" {
		u, e := url.Parse(link)
		if e != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || !contains(r.A.C.ReceiptHosts, u.Hostname()) {
			return nil, invalid("RECEIPT_URL")
		}
		m["kind"] = "link"
		m["external_link"] = link
	} else {
		f, e := files[0].Open()
		if e != nil {
			return nil, e
		}
		defer f.Close()
		data, e := io.ReadAll(io.LimitReader(f, (10<<20)+1))
		if e != nil {
			return nil, e
		}
		if len(data) > 10<<20 {
			return nil, fail(413, "FILE_TOO_LARGE")
		}
		typ := http.DetectContentType(data)
		ext := map[string]string{"application/pdf": "pdf", "image/jpeg": "jpg", "image/png": "png"}[typ]
		if ext == "" {
			return nil, invalid("FILE_TYPE")
		}
		m["kind"] = "file"
		m["content"] = data
		m["mime_type"] = typ
		m["file_name"] = "receipt." + ext
	}
	old, e := one(r.ctx(), r.DB, receiptSQL+" WHERE payment_id=$1", p["id"])
	exists := e == nil
	if !exists {
		if ae, ok := e.(*APIError); !ok || ae.Status != 404 {
			return nil, e
		}
	}
	if r.C.Request.Method == "POST" && exists && old["deleted_at"] == nil {
		return nil, conflict("RECEIPT_ALREADY_ATTACHED")
	}
	if r.C.Request.Method == "PUT" && !exists {
		return nil, fail(404, "NOT_FOUND")
	}
	var v M
	if exists {
		if old["deleted_at"] != nil {
			m["deleted_at"] = nil
		}
		expected, e := strconv.ParseInt(r.C.PostForm("expected_version"), 10, 64)
		if (e != nil || expected != num(old, "version")) && old["deleted_at"] == nil {
			return nil, conflict("VERSION_CONFLICT")
		}
		comment := strings.TrimSpace(r.C.PostForm("comment"))
		if comment == "" && old["deleted_at"] != nil {
			comment = "Новый чек после удаления"
		}
		if comment == "" {
			return nil, invalid("COMMENT_REQUIRED")
		}
		if e = r.exec(`INSERT INTO receipt_versions(receipt_id,snapshot,content,comment,actor_id) SELECT id,to_jsonb(receipts)-'content',content,$2,$3 FROM receipts WHERE id=$1`, old["id"], comment, r.User["user_id"]); e != nil {
			return nil, e
		}
		v, e = r.update("receipts", str(old, "id"), m)
	} else {
		v, e = r.insert("receipts", m)
	}
	if e != nil {
		return nil, e
	}
	result, e := one(r.ctx(), r.DB, receiptSQL+" WHERE id=$1", v["id"])
	if e != nil {
		return nil, e
	}
	if e = r.notify(str(p, "family_id"), "receipt.attached", fmt.Sprint(v["id"], ":", v["version"]), M{"receipt_id": v["id"], "payment_id": p["id"]}); e != nil {
		return nil, e
	}
	return result, r.audit("receipt.saved", "receipts", str(v, "id"), old, result, r.C.PostForm("comment"))
}

const visibleNotice = `NOT EXISTS(SELECT 1 FROM notifications newer WHERE newer.user_id=notifications.user_id AND newer.type=notifications.type AND newer.payload=notifications.payload AND (newer.created_at,newer.id)>(notifications.created_at,notifications.id)) AND type<>'booking.confirmed' AND (type NOT IN ('payment.required','payment.voided') OR EXISTS(SELECT 1 FROM invoices i WHERE i.id::text=notifications.payload->>'invoice_id' AND i.status='open' AND NOT i.compensated)) AND (type<>'receipt.attached' OR EXISTS(SELECT 1 FROM receipts r WHERE r.id::text=notifications.payload->>'receipt_id' AND r.deleted_at IS NULL))`

func notifications(r *R) (any, error) {
	v, e := r.list("SELECT * FROM notifications WHERE "+visibleNotice+" AND user_id=$1 AND ($2<>'true' OR read_at IS NULL) ORDER BY created_at,id", r.User["user_id"], r.C.Query("unread"))
	if e != nil {
		return nil, e
	}
	var n int
	if e = r.DB.QueryRow(r.ctx(), "SELECT count(*) FROM notifications WHERE "+visibleNotice+" AND user_id=$1 AND read_at IS NULL", r.User["user_id"]).Scan(&n); e != nil {
		return nil, e
	}
	object(v)["unread_count"] = n
	return v, nil
}

func notificationRead(r *R) (any, error) {
	if r.C.Param("id") == "" {
		tag, e := r.DB.Exec(r.ctx(), "UPDATE notifications SET read_at=now() WHERE user_id=$1 AND read_at IS NULL", r.User["user_id"])
		r.Status = 200
		return M{"updated": tag.RowsAffected()}, e
	}
	m, e := r.get("notifications", r.C.Param("id"))
	if e != nil {
		return nil, e
	}
	if m["user_id"] != r.User["user_id"] {
		return nil, fail(404, "NOT_FOUND")
	}
	r.Status = 204
	return nil, r.exec("UPDATE notifications SET read_at=coalesce(read_at,now()) WHERE id=$1", m["id"])
}

func csvSafe(s string) string {
	trim := strings.TrimLeft(s, " \t\r\n")
	if len(trim) > 0 && strings.ContainsRune("=+-@", rune(trim[0])) {
		return "'" + s
	}
	if strings.HasPrefix(s, "\t") || strings.HasPrefix(s, "\r") {
		return "'" + s
	}
	return s
}

func (r *R) export(w io.Writer, kind, from, to string) error {
	var sql string
	var args []any
	switch kind {
	case "children":
		sql = `SELECT c.id,c.full_name,c.birth_date,c.family_id,u.full_name parent_full_name,u.phone parent_phone,u.email parent_email FROM children c JOIN families f ON f.id=c.family_id JOIN users u ON u.id=f.parent_user_id ORDER BY c.created_at,c.id`
	case "attendance":
		sql = `SELECT b.id booking_id,l.id lesson_id,l.starts_at,b.child_id,c.full_name child_full_name,a.status,a.cancel_reason,a.charged,a.subscription_id,a.invoice_id,a.marked_at FROM attendance a JOIN bookings b ON b.id=a.booking_id JOIN lessons l ON l.id=b.lesson_id JOIN children c ON c.id=b.child_id WHERE ($1='' OR l.starts_at>=nullif($1,'')::timestamptz) AND ($2='' OR l.starts_at<nullif($2,'')::timestamptz) AND ($3='' OR l.group_id::text=$3) ORDER BY l.starts_at,b.id`
		args = []any{from, to, r.C.Query("group_id")}
	case "payments":
		sql = `SELECT p.id,p.invoice_id,p.child_id,p.family_id,p.amount_minor,p.currency,p.method,p.status,p.paid_at,p.provider_payment_id,r.id receipt_id FROM payments p LEFT JOIN receipts r ON r.payment_id=p.id AND r.deleted_at IS NULL WHERE ($1='' OR p.created_at>=nullif($1,'')::timestamptz) AND ($2='' OR p.created_at<nullif($2,'')::timestamptz) AND ($3='' OR p.status=$3) AND ($4='' OR p.method=$4) ORDER BY p.created_at,p.id`
		args = []any{from, to, r.C.Query("status"), r.C.Query("method")}
	default:
		return fail(404, "NOT_FOUND")
	}
	rows, e := r.DB.Query(r.ctx(), sql, args...)
	if e != nil {
		return e
	}
	defer rows.Close()
	writer := csv.NewWriter(w)
	header := []string{}
	for _, f := range rows.FieldDescriptions() {
		header = append(header, f.Name)
	}
	if e = writer.Write(header); e != nil {
		return e
	}
	for rows.Next() {
		values, e := rows.Values()
		if e != nil {
			return e
		}
		record := []string{}
		for _, v := range values {
			var s string
			switch t := v.(type) {
			case nil:
				s = ""
			case string:
				s = csvSafe(t)
			case [16]byte:
				s = fmt.Sprintf("%x-%x-%x-%x-%x", t[:4], t[4:6], t[6:8], t[8:10], t[10:])
			default:
				s = fmt.Sprint(v)
			}
			record = append(record, s)
		}
		if e = writer.Write(record); e != nil {
			return e
		}
	}
	writer.Flush()
	if e = rows.Err(); e != nil {
		return e
	}
	return writer.Error()
}

func export(r *R) (any, error) {
	name := r.C.Param("file")
	if !contains([]string{"children.csv", "attendance.csv", "payments.csv", "all.zip"}, name) {
		return nil, fail(404, "NOT_FOUND")
	}
	required := name == "attendance.csv" || name == "payments.csv"
	from, to, e := r.rangeDates(required)
	if e != nil {
		return nil, e
	}
	r.C.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	if name == "all.zip" {
		r.C.Header("Content-Type", "application/zip")
		z := zip.NewWriter(r.C.Writer)
		for _, k := range []string{"children", "attendance", "payments"} {
			w, e := z.Create(k + ".csv")
			if e != nil {
				return nil, e
			}
			if e = r.export(w, k, from, to); e != nil {
				z.Close()
				return nil, e
			}
		}
		return nil, z.Close()
	}
	r.C.Header("Content-Type", "text/csv; charset=utf-8")
	return nil, r.export(r.C.Writer, strings.TrimSuffix(name, ".csv"), from, to)
}
