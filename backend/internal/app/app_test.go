package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"jasna/backend/internal/database"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestChargeRules(t *testing.T) {
	rule := M{"attended_charge_lesson": true, "missed_charge_lesson": true, "parent_cancel_charge_lesson": true, "free_cancel_before_hours": int64(24)}
	start := time.Now()
	for _, v := range []struct {
		status, reason string
		hours          int
		want           bool
	}{{"attended", "", 0, true}, {"missed", "", 0, true}, {"cancelled", "parent", 24, false}, {"cancelled", "parent", 23, true}, {"cancelled", "master", 0, false}, {"cancelled", "transfer", 0, false}, {"pending", "", 0, false}} {
		if got := chargeFor(rule, v.status, v.reason, start, start.Add(-time.Duration(v.hours)*time.Hour)); got != v.want {
			t.Fatalf("%+v got %v", v, got)
		}
	}
}

func TestMoneyParsing(t *testing.T) {
	for _, v := range []int64{1, 99, 100, 400000, 987654321} {
		n, e := parseRubles(rubles(v))
		if e != nil || n != v {
			t.Fatal(v, n, e)
		}
	}
	for _, s := range []string{"NaN", "1.001", "-1.00", "1.-1", "1.0", "9223372036854775807.00"} {
		if _, e := parseRubles(s); e == nil {
			t.Fatal("accepted", s)
		}
	}
}

func TestPassword(t *testing.T) {
	h := passwordHash("correct-long-password")
	if !passwordOK(h, "correct-long-password") || passwordOK(h, "wrong") || passwordOK("broken", "anything") {
		t.Fatal("password verification")
	}
}

func TestCSVSafe(t *testing.T) {
	for _, s := range []string{"=SUM(A1)", "  +cmd", "\ttext", "\rtext", "-1+1", "@x"} {
		if !strings.HasPrefix(csvSafe(s), "'") {
			t.Fatal(s)
		}
	}
	if csvSafe("Анна") != "Анна" {
		t.Fatal("plain name changed")
	}
}

func TestRouteCoverage(t *testing.T) {
	routes := New(Config{}).Router().Routes()
	set := map[string]bool{}
	for _, r := range routes {
		set[r.Method+" "+strings.TrimPrefix(r.Path, "/api/v1")] = true
	}
	b, e := os.ReadFile("../../../endpoint.md")
	if e != nil {
		t.Fatal(e)
	}
	re := regexp.MustCompile("(?m)^### `(GET|POST|PATCH|PUT|DELETE) ([^`]+)`")
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		if strings.HasPrefix(m[2], "/master/export/") {
			if !set["GET /master/export/:file"] {
				t.Fatal("export missing")
			}
			continue
		}
		if !set[m[1]+" "+strings.ReplaceAll(m[2], ":lessonId", ":id")] {
			t.Errorf("missing %s %s", m[1], m[2])
		}
	}
}

type testClient struct {
	router http.Handler
	cookie *http.Cookie
	csrf   string
}

func (c *testClient) request(method, path string, body any, key string) (int, M, *httptest.ResponseRecorder) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(jsonBytes(body))
	}
	req := httptest.NewRequest(method, "/api/v1"+path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	if c.csrf != "" {
		req.Header.Set("X-CSRF-Token", c.csrf)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	c.router.ServeHTTP(w, req)
	m, _ := decode(w.Body.Bytes())
	return w.Code, m, w
}

func (c *testClient) ok(t *testing.T, method, path string, body any, want int) M {
	t.Helper()
	code, m, w := c.request(method, path, body, newID())
	if code != want {
		t.Fatalf("%s %s = %d want %d: %s", method, path, code, want, w.Body.String())
	}
	return m
}

func (c *testClient) auth(t *testing.T, path string, body M) {
	t.Helper()
	code, m, w := c.request("POST", path, body, "")
	if code != 200 {
		t.Fatalf("auth %d %s", code, w.Body.String())
	}
	c.csrf = str(m, "csrf_token")
	c.cookie = w.Result().Cookies()[0]
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPostgresFullCycle(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL pointing to disposable jasna_test database")
	}
	u, e := url.Parse(dsn)
	if e != nil || !strings.HasPrefix(strings.TrimPrefix(u.Path, "/"), "jasna_test") {
		t.Fatal("refusing non-test database")
	}
	ctx := context.Background()
	db, e := database.Open(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec(ctx, "DROP SCHEMA IF EXISTS jasna CASCADE"); e != nil {
		t.Fatal(e)
	}
	if e = database.Migrate(ctx, db); e != nil {
		t.Fatal(e)
	}
	if e = database.Migrate(ctx, db); e != nil {
		t.Fatal("idempotent migration", e)
	}
	a := New(Config{AppURL: "http://localhost:3000", Origins: []string{"http://localhost:3000"}, MasterLogin: "master", MasterPassword: "test-long-password", ShopID: "test-shop", Secret: "test-secret", ReturnURL: "http://localhost:3000/return", ReceiptHosts: []string{"lknpd.nalog.ru"}, PaymentTest: true})
	a.DB = db
	if e = a.Bootstrap(ctx); e != nil {
		t.Fatal(e)
	}
	router := a.Router()
	master := &testClient{router: router}
	master.auth(t, "/auth/master/login", M{"login": "master", "password": "test-long-password"})
	group := master.ok(t, "POST", "/master/groups", M{"title": "Тестовая группа", "capacity": 2}, 201)
	rule := M{"group_id": group["id"], "valid_from": time.Now().In(zone).Format("2006-01-02"), "attended_charge_lesson": true, "missed_charge_lesson": true, "parent_cancel_charge_lesson": true, "free_cancel_before_hours": 24, "parent_cancel_allowed": true, "cancel_deadline_hours": 0, "transfer_allowed": true, "transfer_deadline_hours": 0, "transfer_charge_lesson": false, "single_lesson_price_minor": 100000, "ending_notice_days": 3, "ending_notice_lessons": 1}
	master.ok(t, "POST", "/master/attendance-rules", rule, 201)
	start := time.Now().Add(48 * time.Hour).Truncate(time.Second)
	lesson := master.ok(t, "POST", "/master/lessons", M{"group_id": group["id"], "starts_at": start.Format(time.RFC3339), "ends_at": start.Add(time.Hour).Format(time.RFC3339)}, 201)
	family := master.ok(t, "POST", "/master/families", M{"parent_full_name": "Тестовый родитель", "children": []M{{"full_name": "Ребёнок А"}, {"full_name": "Ребёнок Б"}}}, 201)
	kids := family["children"].([]any)
	child := str(object(kids[0]), "id")
	child2 := str(object(kids[1]), "id")
	link := master.ok(t, "POST", "/master/families/"+str(family, "id")+"/access-links", M{"expires_at": time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339)}, 201)
	token := strings.Split(str(link, "url"), "token=")[1]
	parent := &testClient{router: router}
	parent.auth(t, "/auth/access-link/exchange", M{"token": token})
	parent.ok(t, "GET", "/families/me", nil, 200)
	b := parent.ok(t, "POST", "/bookings", M{"lesson_id": lesson["id"], "child_id": child}, 201)
	family2 := master.ok(t, "POST", "/master/families", M{"parent_full_name": "Другая семья", "children": []M{{"full_name": "Ребёнок В"}}}, 201)
	child3 := str(object(family2["children"].([]any)[0]), "id")
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for _, kid := range []string{child2, child3} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			code, _, _ := master.request("POST", "/bookings", M{"lesson_id": lesson["id"], "child_id": id}, newID())
			codes <- code
		}(kid)
	}
	wg.Wait()
	close(codes)
	counts := map[int]int{}
	for code := range codes {
		counts[code]++
	}
	if counts[201] != 1 || counts[409] != 1 {
		t.Fatal("capacity race", counts)
	}
	parent.ok(t, "POST", "/bookings", M{"lesson_id": lesson["id"], "child_id": child}, 409)
	parent.ok(t, "POST", "/bookings", M{"lesson_id": lesson["id"], "child_id": child3}, 404)
	plan := master.ok(t, "POST", "/master/plans", M{"title": "4 занятия", "group_id": group["id"], "lessons_total": 4, "validity_days": 30, "price_minor": 400000}, 201)
	inv := parent.ok(t, "POST", "/invoices", M{"child_id": child, "kind": "subscription", "plan_id": plan["id"], "valid_from": time.Now().In(zone).Format("2006-01-02")}, 201)
	var provider M
	var providerMu sync.Mutex
	a.HTTP = &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		providerMu.Lock()
		defer providerMu.Unlock()
		if req.Method == "POST" {
			var body M
			json.NewDecoder(req.Body).Decode(&body)
			if req.Header.Get("Idempotence-Key") == "" {
				return nil, fmt.Errorf("missing provider key")
			}
			provider = M{"id": "provider-1", "status": "pending", "test": true, "paid": false, "amount": body["amount"], "recipient": M{"account_id": "test-shop"}, "metadata": body["metadata"], "confirmation": M{"confirmation_url": "https://yoomoney.ru/test"}}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(jsonBytes(provider))), Header: make(http.Header)}, nil
	})}
	intent := parent.ok(t, "POST", "/payments/intent", M{"invoice_id": inv["id"], "method": "sbp"}, 201)
	if str(intent, "payment_url") == "" {
		t.Fatal("no payment URL")
	}
	providerMu.Lock()
	provider["status"] = "succeeded"
	provider["paid"] = true
	providerMu.Unlock()
	guest := &testClient{router: router}
	codes = make(chan int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, _, _ := guest.request("POST", "/webhooks/payments/yookassa", M{"event": "payment.succeeded", "object": M{"id": "provider-1"}}, "")
			codes <- code
		}()
	}
	wg.Wait()
	close(codes)
	for code := range codes {
		if code != 200 {
			t.Fatal("webhook", code)
		}
	}
	var credits, subs int
	db.QueryRow(ctx, "SELECT count(*) FROM money_ledger WHERE reason='payment'").Scan(&credits)
	db.QueryRow(ctx, "SELECT count(*) FROM subscriptions").Scan(&subs)
	if credits != 1 || subs != 1 {
		t.Fatal("duplicate credit", credits, subs)
	}
	a.Now = func() time.Time { return start.Add(30 * time.Minute) }
	attendancePath := "/master/lessons/" + str(lesson, "id") + "/attendance/" + str(b, "id")
	marked := master.ok(t, "PUT", attendancePath, M{"status": "attended", "expected_version": 1}, 200)
	if num(marked, "lessons_remaining") != 3 {
		t.Fatal("remaining", marked)
	}
	repeated := master.ok(t, "PUT", attendancePath, M{"status": "attended", "expected_version": marked["version"]}, 200)
	if num(repeated, "version") != num(marked, "version") {
		t.Fatal("non-idempotent attendance")
	}
	corrected := master.ok(t, "PUT", attendancePath, M{"status": "missed", "expected_version": marked["version"], "comment": "Исправление"}, 200)
	if num(corrected, "lessons_remaining") != 3 {
		t.Fatal("paid to paid correction", corrected)
	}
	reversed := master.ok(t, "PUT", attendancePath, M{"status": "cancelled", "cancel_reason": "master", "expected_version": corrected["version"], "comment": "Отмена"}, 200)
	if num(reversed, "lessons_remaining") != 4 {
		t.Fatal("reversal", reversed)
	}
	marked = master.ok(t, "PUT", attendancePath, M{"status": "attended", "expected_version": reversed["version"], "comment": "Восстановление отметки"}, 200)
	if num(marked, "lessons_remaining") != 3 {
		t.Fatal("recharge", marked)
	}
	bal := parent.ok(t, "GET", "/families/me/balance", nil, 200)
	if num(bal, "balance_minor") != 0 || num(bal, "debt_minor") != 0 {
		t.Fatal("money", bal)
	}
	var upload bytes.Buffer
	mw := multipart.NewWriter(&upload)
	f, _ := mw.CreateFormFile("file", "receipt.pdf")
	f.Write([]byte("%PDF-1.4\nTest receipt\n%%EOF"))
	mw.Close()
	req := httptest.NewRequest("POST", "/api/v1/master/payments/"+str(intent, "id")+"/receipts", &upload)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Origin", a.C.AppURL)
	req.Header.Set("X-CSRF-Token", master.csrf)
	req.AddCookie(master.cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("receipt %d %s", w.Code, w.Body.String())
	}
	receipt, _ := decode(w.Body.Bytes())
	code, _, w := parent.request("GET", "/receipts/"+str(receipt, "id")+"/download", nil, "")
	if code != 200 || !strings.HasPrefix(w.Body.String(), "%PDF-") {
		t.Fatal("download", code, w.Body.String())
	}
	guest.ok(t, "GET", "/receipts/"+str(receipt, "id")+"/download", nil, 401)
	a.jobs(ctx)
	notices := parent.ok(t, "GET", "/notifications", nil, 200)
	if num(notices, "unread_count") == 0 {
		t.Fatal("missing notifications")
	}
	code, _, w = master.request("GET", "/master/export/payments.csv?from="+time.Now().In(zone).Format("2006-01-02")+"&to="+start.In(zone).Format("2006-01-02"), nil, "")
	if code != 200 || !strings.Contains(w.Body.String(), "amount_minor") {
		t.Fatal("csv", code, w.Body.String())
	}
	parent.ok(t, "GET", "/master/families", nil, 403)
	noCSRF := *master
	noCSRF.csrf = ""
	noCSRF.ok(t, "POST", "/master/groups", M{"title": "X", "capacity": 1}, 403)
	master.ok(t, "PATCH", "/master/groups/"+str(group, "id"), M{"expected_version": 999, "title": "X"}, 409)
	master.ok(t, "POST", "/master/groups", M{"title": "X", "capacity": 1, "is_admin": true}, 422)
	parent.ok(t, "GET", "/payments/00000000-0000-0000-0000-000000000001", nil, 404)

	// Transfer rolls back cancellation when destination is full; then succeeds atomically.
	nextStart := start.Add(24 * time.Hour)
	next := master.ok(t, "POST", "/master/lessons", M{"group_id": group["id"], "capacity": 1, "starts_at": nextStart.Format(time.RFC3339), "ends_at": nextStart.Add(time.Hour).Format(time.RFC3339)}, 201)
	targetStart := nextStart.Add(2 * time.Hour)
	target := master.ok(t, "POST", "/master/lessons", M{"group_id": group["id"], "capacity": 1, "starts_at": targetStart.Format(time.RFC3339), "ends_at": targetStart.Add(time.Hour).Format(time.RFC3339)}, 201)
	original := parent.ok(t, "POST", "/bookings", M{"child_id": child2, "lesson_id": next["id"]}, 201)
	occupied := master.ok(t, "POST", "/bookings", M{"child_id": child3, "lesson_id": target["id"]}, 201)
	parent.ok(t, "POST", "/bookings/"+str(original, "id")+"/transfer", M{"target_lesson_id": target["id"]}, 409)
	var active bool
	if e = db.QueryRow(ctx, "SELECT status='active' FROM bookings WHERE id=$1", original["id"]).Scan(&active); e != nil || !active {
		t.Fatal("failed transfer canceled source", e)
	}
	master.ok(t, "DELETE", "/bookings/"+str(occupied, "id"), nil, 204)
	moved := parent.ok(t, "POST", "/bookings/"+str(original, "id")+"/transfer", M{"target_lesson_id": target["id"]}, 201)
	// Absent subscription: record attendance, create debt, settle manually exactly once.
	a.Now = func() time.Time { return targetStart.Add(30 * time.Minute) }
	debtPath := "/master/lessons/" + str(target, "id") + "/attendance/" + str(moved, "new_booking_id")
	debtMark := master.ok(t, "PUT", debtPath, M{"status": "attended", "expected_version": 1}, 200)
	if str(debtMark, "invoice_id") == "" {
		t.Fatal("missing single-lesson invoice")
	}
	bal = parent.ok(t, "GET", "/families/me/balance", nil, 200)
	if num(bal, "debt_minor") != 100000 || num(bal, "balance_minor") != -100000 {
		t.Fatal("debt", bal)
	}
	key := newID()
	paymentBody := M{"invoice_id": debtMark["invoice_id"], "method": "cash"}
	code, manual, _ := master.request("POST", "/master/payments", paymentBody, key)
	if code != 201 {
		t.Fatal("manual", code, manual)
	}
	code, replay, _ := master.request("POST", "/master/payments", paymentBody, key)
	if code != 201 || replay["id"] != manual["id"] {
		t.Fatal("manual replay", code, replay)
	}
	code, _, _ = master.request("POST", "/master/payments", M{"invoice_id": debtMark["invoice_id"], "method": "transfer"}, key)
	if code != 409 {
		t.Fatal("key conflict", code)
	}
	debtReverse := master.ok(t, "PUT", debtPath, M{"status": "cancelled", "cancel_reason": "master", "comment": "Исправление", "expected_version": debtMark["version"]}, 200)
	bal = parent.ok(t, "GET", "/families/me/balance", nil, 200)
	if num(bal, "balance_minor") != 100000 {
		t.Fatal("credit after correction", bal)
	}
	master.ok(t, "PUT", debtPath, M{"status": "attended", "comment": "Возврат отметки", "expected_version": debtReverse["version"]}, 200)
	bal = parent.ok(t, "GET", "/families/me/balance", nil, 200)
	if num(bal, "balance_minor") != 0 || num(bal, "debt_minor") != 0 {
		t.Fatal("restored paid attendance", bal)
	}
	// Cross-family isolation includes receipts and ledger, even with a valid parent session.
	secondLink := master.ok(t, "POST", "/master/families/"+str(family2, "id")+"/access-links", M{"expires_at": time.Now().Add(90 * 24 * time.Hour).Format(time.RFC3339)}, 201)
	parent2 := &testClient{router: router}
	parent2.auth(t, "/auth/access-link/exchange", M{"token": strings.Split(str(secondLink, "url"), "token=")[1]})
	parent2.ok(t, "GET", "/receipts/"+str(receipt, "id")+"/download", nil, 404)
	parent2.ok(t, "GET", "/invoices/"+str(inv, "id"), nil, 404)
	parent2.ok(t, "GET", "/payments/"+str(intent, "id"), nil, 404)
	// Cancellation followed by rebooking is allowed, with a new historical record.
	a.Now = time.Now
	fresh := master.ok(t, "POST", "/master/lessons", M{"group_id": group["id"], "starts_at": start.Add(7 * 24 * time.Hour).Format(time.RFC3339), "ends_at": start.Add(7*24*time.Hour + time.Hour).Format(time.RFC3339)}, 201)
	first := parent.ok(t, "POST", "/bookings", M{"child_id": child, "lesson_id": fresh["id"]}, 201)
	parent.ok(t, "DELETE", "/bookings/"+str(first, "id"), nil, 204)
	again := parent.ok(t, "POST", "/bookings", M{"child_id": child, "lesson_id": fresh["id"]}, 201)
	if again["id"] == first["id"] {
		t.Fatal("history overwritten")
	}
	// Cancellation of the lesson also reverses charges of previously canceled bookings.
	master.ok(t, "PATCH", "/master/lessons/"+str(fresh, "id"), M{"expected_version": fresh["version"], "status": "cancelled", "cancel_reason": "Тест"}, 200)
	// Provider mismatch must not credit money.
	providerMu.Lock()
	provider["amount"] = M{"value": "0.01", "currency": "RUB"}
	providerMu.Unlock()
	guest.ok(t, "POST", "/webhooks/payments/yookassa", M{"event": "payment.succeeded", "object": M{"id": "provider-1"}}, 400)

	// Prepaid single lesson canceled before attendance: credit must be returned too.
	prepaidLesson := master.ok(t, "POST", "/master/lessons", M{"group_id": group["id"], "starts_at": start.Add(8 * 24 * time.Hour).Format(time.RFC3339), "ends_at": start.Add(8*24*time.Hour + time.Hour).Format(time.RFC3339)}, 201)
	prepaidBooking := parent.ok(t, "POST", "/bookings", M{"child_id": child2, "lesson_id": prepaidLesson["id"]}, 201)
	prepaidInvoice := parent.ok(t, "POST", "/invoices", M{"child_id": child2, "kind": "single_lesson", "booking_id": prepaidBooking["id"]}, 201)
	master.ok(t, "POST", "/master/payments", M{"invoice_id": prepaidInvoice["id"], "method": "cash"}, 201)
	parent.ok(t, "DELETE", "/bookings/"+str(prepaidBooking, "id"), nil, 204)
	parent.ok(t, "DELETE", "/bookings/"+str(prepaidBooking, "id"), nil, 204)
	bal = parent.ok(t, "GET", "/families/me/balance", nil, 200)
	if num(bal, "balance_minor") != 100000 {
		t.Fatal("prepaid cancellation", bal)
	}
	// An unknown provider outcome beyond its idempotency window must not create a new payment.
	uncertainInvoice := parent.ok(t, "POST", "/invoices", M{"child_id": child, "kind": "subscription", "plan_id": plan["id"], "valid_from": time.Now().In(zone).Format("2006-01-02")}, 201)
	uncertainID := newID()
	_, e = db.Exec(ctx, `INSERT INTO payments(id,invoice_id,family_id,child_id,amount_minor,method,provider_key,created_at) VALUES($1,$2,$3,$4,400000,'sbp',$5,now()-interval '25 hours')`, uncertainID, uncertainInvoice["id"], family["id"], child, newID())
	if e != nil {
		t.Fatal(e)
	}
	parent.ok(t, "POST", "/payments/intent", M{"invoice_id": uncertainInvoice["id"], "method": "sbp"}, 409)
	var linked bool
	db.QueryRow(ctx, "SELECT provider_payment_id IS NOT NULL FROM payments WHERE id=$1", uncertainID).Scan(&linked)
	if linked {
		t.Fatal("old uncertain payment recreated")
	}
	master.ok(t, "DELETE", "/master/families/"+str(family, "id")+"/access-links/"+str(link, "id"), nil, 204)
	parent.ok(t, "GET", "/families/me", nil, 401)
	guest.ok(t, "POST", "/auth/access-link/exchange", M{"token": token}, 401)
}
