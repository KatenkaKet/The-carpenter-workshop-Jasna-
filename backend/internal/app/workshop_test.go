package app

import (
	"bytes"
	"context"
	"fmt"
	"jasna/backend/internal/database"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWorkshopReservationsAndCorrections(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("isolated TEST_DATABASE_URL required")
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
		t.Fatal(e)
	}
	now := time.Now().Truncate(time.Second)
	a := New(Config{AppURL: "http://localhost:3000", Origins: []string{"http://localhost:3000"}, MasterLogin: "master", MasterPassword: "test-long-password", ReceiptHosts: []string{"lknpd.nalog.ru"}})
	a.DB = db
	a.Now = func() time.Time { return now }
	if e = a.Bootstrap(ctx); e != nil {
		t.Fatal(e)
	}
	router := a.Router()
	master := &testClient{router: router}
	master.auth(t, "/auth/master/login", M{"login": "master", "password": "test-long-password"})
	g := master.ok(t, "POST", "/master/groups", M{"title": "Test fixture group", "capacity": 10}, 201)
	rule := M{"group_id": g["id"], "valid_from": now.In(zone).Format("2006-01-02"), "attended_charge_lesson": true, "missed_charge_lesson": true, "parent_cancel_charge_lesson": true, "free_cancel_before_hours": 3, "parent_cancel_allowed": true, "cancel_deadline_hours": 0, "transfer_allowed": true, "transfer_deadline_hours": 0, "transfer_charge_lesson": false, "single_lesson_price_minor": 100000, "ending_notice_days": 3, "ending_notice_lessons": 1}
	master.ok(t, "POST", "/master/attendance-rules", rule, 201)
	var plan M
	for i := 0; i < 5; i++ {
		plan = master.ok(t, "POST", "/master/plans", M{"title": fmt.Sprint("Plan ", i), "group_id": g["id"], "lessons_total": 8, "validity_days": 30, "price_minor": 800000}, 201)
	}
	if num(master.ok(t, "GET", "/master/plans", nil, 200), "total") != 5 {
		t.Fatal("plan count")
	}
	fam := master.ok(t, "POST", "/master/families", M{"parent_full_name": "Test fixture parent", "children": []M{{"full_name": "Test child"}}}, 201)
	child := str(object(fam["children"].([]any)[0]), "id")
	linkPath := "/master/families/" + str(fam, "id") + "/access-links"
	link := master.ok(t, "POST", linkPath, M{"expires_at": now.Add(30 * 24 * time.Hour).Format(time.RFC3339)}, 201)
	parent := &testClient{router: router}
	parent.auth(t, "/auth/access-link/exchange", M{"token": strings.Split(str(link, "url"), "token=")[1]})
	purchase := M{"child_id": child, "plan_id": plan["id"], "valid_from": now.In(zone).Format("2006-01-02"), "method": "cash"}
	k := newID()
	code, p, _ := master.request("POST", "/master/subscription-purchases", purchase, k)
	if code != 201 {
		t.Fatal(code, p)
	}
	code, repeat, _ := master.request("POST", "/master/subscription-purchases", purchase, k)
	if code != 201 || repeat["id"] != p["id"] {
		t.Fatal("non-idempotent purchase")
	}
	if num(master.ok(t, "GET", "/master/payments", nil, 200), "total") != 1 {
		t.Fatal("duplicate payment")
	}
	readSub := func() M {
		return object(parent.ok(t, "GET", "/families/me/subscriptions", nil, 200)["items"].([]any)[0])
	}
	assertCounts := func(used, reserved, available int64) {
		t.Helper()
		s := readSub()
		if num(s, "lessons_used") != used || num(s, "lessons_reserved") != reserved || num(s, "lessons_available") != available {
			t.Fatalf("expected used/reserved/available %d/%d/%d: %+v", used, reserved, available, s)
		}
	}
	assertCounts(0, 0, 8)
	a.jobs(ctx)
	a.jobs(ctx)
	for _, v := range parent.ok(t, "GET", "/notifications", nil, 200)["items"].([]any) {
		n := object(v)
		if str(n, "type") == "payment.required" || str(n, "type") == "booking.confirmed" {
			t.Fatal("obsolete notification", n)
		}
	}
	start := now.Add(48 * time.Hour).In(zone)
	start = time.Date(start.Year(), start.Month(), start.Day(), 9, 15, 0, 0, zone)
	lessons := []M{}
	for i := 0; i < 10; i++ {
		x := start.Add(time.Duration(i) * 2 * time.Hour)
		lessons = append(lessons, master.ok(t, "POST", "/master/lessons", M{"group_id": g["id"], "starts_at": x.Format(time.RFC3339), "ends_at": x.Add(75 * time.Minute).Format(time.RFC3339)}, 201))
	}
	ig := master.ok(t, "POST", "/master/groups", M{"title": "Individual fixture", "capacity": 1, "is_individual": true}, 201)
	indiv := master.ok(t, "POST", "/master/lessons", M{"group_id": ig["id"], "starts_at": start.Add(48 * time.Hour).Format(time.RFC3339), "ends_at": start.Add(48*time.Hour + 135*time.Minute).Format(time.RFC3339), "is_makeup": true}, 201)
	if !yes(indiv, "is_makeup") {
		t.Fatal("makeup lost")
	}
	bookings := []M{}
	for i := 0; i < 3; i++ {
		bookings = append(bookings, parent.ok(t, "POST", "/bookings", M{"lesson_id": lessons[i]["id"], "child_id": child, "funding_kind": "subscription"}, 201))
	}
	assertCounts(0, 3, 5)
	var wg sync.WaitGroup
	codes := make(chan int, 6)
	for i := 3; i < 9; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c, _, _ := parent.request("POST", "/bookings", M{"lesson_id": lessons[i]["id"], "child_id": child, "funding_kind": "subscription"}, newID())
			codes <- c
		}(i)
	}
	wg.Wait()
	close(codes)
	accepted := 0
	for c := range codes {
		if c == 201 {
			accepted++
		} else if c != 409 {
			t.Fatal("unexpected concurrent status", c)
		}
	}
	if accepted != 5 {
		t.Fatal("over-reservation", accepted)
	}
	assertCounts(0, 8, 0)
	// Immediate settings override the old booking snapshot, with optimistic locking.
	pol := master.ok(t, "PUT", "/master/groups/"+str(g, "id")+"/booking-policy", M{"expected_version": 0, "missed_charge_lesson": false, "free_cancel_before_hours": 3, "transfer_allowed": true}, 200)
	master.ok(t, "PUT", "/master/groups/"+str(g, "id")+"/booking-policy", M{"expected_version": 0, "missed_charge_lesson": true, "free_cancel_before_hours": 1, "transfer_allowed": true}, 409)
	q := parent.ok(t, "GET", "/bookings/"+str(bookings[0], "id")+"/cancellation", nil, 200)
	if yes(q, "will_charge") || !yes(q, "allowed") {
		t.Fatal(q)
	}
	parent.ok(t, "DELETE", "/bookings/"+str(bookings[0], "id"), nil, 204)
	parent.ok(t, "DELETE", "/bookings/"+str(bookings[0], "id"), nil, 204)
	assertCounts(0, 7, 1)
	now = start.Add(time.Hour) // second lesson is one hour away
	q = parent.ok(t, "GET", "/bookings/"+str(bookings[1], "id")+"/cancellation", nil, 200)
	if !yes(q, "will_charge") {
		t.Fatal("late cancellation not charged", q)
	}
	parent.ok(t, "DELETE", "/bookings/"+str(bookings[1], "id"), nil, 204)
	parent.ok(t, "DELETE", "/bookings/"+str(bookings[1], "id"), nil, 204)
	assertCounts(1, 6, 1)
	now = start.Add(5 * time.Hour)
	mark := master.ok(t, "PUT", "/master/lessons/"+str(lessons[2], "id")+"/attendance/"+str(bookings[2], "id"), M{"status": "attended", "expected_version": 1}, 200)
	assertCounts(2, 5, 1)
	master.ok(t, "PUT", "/master/lessons/"+str(lessons[2], "id")+"/attendance/"+str(bookings[2], "id"), M{"status": "attended", "expected_version": mark["version"]}, 200)
	assertCounts(2, 5, 1)
	// Pick a successful concurrent booking and record a non-chargeable miss.
	allBookings := parent.ok(t, "GET", "/families/me/bookings", nil, 200)["items"].([]any)
	var missed M
	for _, v := range allBookings {
		b := object(v)
		if str(b, "status") == "active" && str(object(b["attendance"]), "status") == "pending" {
			missed = b
			break
		}
	}
	now = at(missed, "starts_at").Add(time.Minute)
	master.ok(t, "PUT", "/master/lessons/"+str(missed, "lesson_id")+"/attendance/"+str(missed, "id"), M{"status": "missed", "expected_version": 1}, 200)
	assertCounts(2, 4, 2)
	// Zero-hour cancellation is free until start, including already-created bookings.
	master.ok(t, "PUT", "/master/groups/"+str(g, "id")+"/booking-policy", M{"expected_version": pol["version"], "missed_charge_lesson": false, "free_cancel_before_hours": 0, "transfer_allowed": false}, 200)
	// Upload is optional and does not alter successful payment.
	var upload bytes.Buffer
	mw := multipart.NewWriter(&upload)
	file, _ := mw.CreateFormFile("file", "receipt.pdf")
	file.Write([]byte("%PDF-1.4\nfixture\n%%EOF"))
	mw.Close()
	req := httptest.NewRequest("POST", "/api/v1/master/payments/"+str(p, "id")+"/receipts", &upload)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Origin", a.C.AppURL)
	req.Header.Set("X-CSRF-Token", master.csrf)
	req.AddCookie(master.cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	receipt, _ := decode(w.Body.Bytes())
	parent.ok(t, "GET", "/receipts/"+str(receipt, "id")+"/download", nil, 200)
	master.ok(t, "DELETE", "/master/receipts/"+str(receipt, "id"), M{"expected_version": receipt["version"], "reason": "Wrong attachment"}, 204)
	parent.ok(t, "GET", "/receipts/"+str(receipt, "id")+"/download", nil, 404)
	voidBody := M{"expected_version": p["version"], "reason": "Duplicate manual entry"}
	k = newID()
	code, _, _ = master.request("POST", "/master/payments/"+str(p, "id")+"/void", voidBody, k)
	if code != 201 {
		t.Fatal("void", code)
	}
	code, _, _ = master.request("POST", "/master/payments/"+str(p, "id")+"/void", voidBody, k)
	if code != 201 {
		t.Fatal(code)
	}
	assertCounts(2, 4, 0)
	if str(readSub(), "status") != "suspended" {
		t.Fatal("not suspended")
	}
	if num(parent.ok(t, "GET", "/families/me/balance", nil, 200), "debt_minor") != 800000 {
		t.Fatal("debt after void")
	}
	restored := master.ok(t, "POST", "/master/payments", M{"invoice_id": p["invoice_id"], "method": "transfer"}, 201)
	if str(restored, "status") != "succeeded" {
		t.Fatal(restored)
	}
	assertCounts(2, 4, 2)
	if num(parent.ok(t, "GET", "/families/me/balance", nil, 200), "debt_minor") != 0 {
		t.Fatal("phantom debt")
	}
	a.jobs(ctx)
	a.jobs(ctx)
	for _, v := range parent.ok(t, "GET", "/notifications", nil, 200)["items"].([]any) {
		if str(object(v), "type") == "payment.required" {
			t.Fatal("repeat payment demand")
		}
	}
	master.ok(t, "DELETE", linkPath+"/"+str(link, "id"), nil, 204)
	parent.ok(t, "GET", "/families/me", nil, 401)
	guest := &testClient{router: router}
	guest.ok(t, "POST", "/auth/access-link/exchange", M{"token": strings.Split(str(link, "url"), "token=")[1]}, 401)
	fresh := master.ok(t, "POST", linkPath, M{"expires_at": now.Add(24 * time.Hour).Format(time.RFC3339)}, 201)
	guest.auth(t, "/auth/access-link/exchange", M{"token": strings.Split(str(fresh, "url"), "token=")[1]})
	guest.ok(t, "GET", "/families/me", nil, 200)
	var unbalanced int
	db.QueryRow(ctx, `SELECT count(*) FROM subscriptions s WHERE lessons_remaining<>(SELECT coalesce(sum(delta),0) FROM subscription_ledger WHERE subscription_id=s.id)`).Scan(&unbalanced)
	if unbalanced != 0 {
		t.Fatal("ledger mismatch")
	}
}
