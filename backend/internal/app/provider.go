package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func rubles(n int64) string { return fmt.Sprintf("%d.%02d", n/100, n%100) }

func parseRubles(s string) (int64, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 2 || len(parts[1]) != 2 {
		return 0, invalid("PROVIDER_AMOUNT")
	}
	a, e := strconv.ParseInt(parts[0], 10, 64)
	if e != nil || a < 0 || a > 10000000000 {
		return 0, invalid("PROVIDER_AMOUNT")
	}
	b, e := strconv.ParseInt(parts[1], 10, 64)
	if e != nil || b < 0 || b > 99 {
		return 0, invalid("PROVIDER_AMOUNT")
	}
	return a*100 + b, nil
}

func (a *App) provider(ctx context.Context, method, path, key string, body any) (M, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(jsonBytes(body))
	}
	req, e := http.NewRequestWithContext(ctx, method, "https://api.yookassa.ru/v3"+path, reader)
	if e != nil {
		return nil, e
	}
	req.SetBasicAuth(a.C.ShopID, a.C.Secret)
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotence-Key", key)
	}
	resp, e := a.HTTP.Do(req)
	if e != nil {
		return nil, fail(503, "PROVIDER_UNAVAILABLE")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Warn("provider request failed", "status", resp.StatusCode)
		return nil, fail(503, "PROVIDER_UNAVAILABLE")
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if e != nil {
		return nil, e
	}
	m, e := decode(b)
	if e != nil {
		return nil, fail(503, "INVALID_PROVIDER_RESPONSE")
	}
	return m, nil
}

func (a *App) system(ctx context.Context, h Handler) (any, error) {
	tx, e := a.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	c := &gin.Context{Request: (&http.Request{}).WithContext(ctx)}
	c.Set("request_id", newID())
	r := &R{A: a, C: c, DB: tx, User: M{}, Body: M{}}
	if e = r.exec("SELECT pg_advisory_xact_lock(7341022)"); e != nil {
		return nil, e
	}
	v, e := h(r)
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	return v, nil
}

func (a *App) ensurePayment(ctx context.Context, id string) (M, error) {
	p, e := one(ctx, a.DB, "SELECT * FROM payments WHERE id=$1", id)
	if e != nil {
		return nil, e
	}
	if str(p, "payment_url") != "" || str(p, "status") != "pending" {
		return paymentPublic(p), nil
	}
	if str(p, "provider_payment_id") == "" && a.Now().Sub(at(p, "created_at")) >= 23*time.Hour {
		_, err := a.system(ctx, func(r *R) (any, error) {
			return nil, r.exec(`INSERT INTO outbox(user_id,type,payload,event_key) SELECT id,'payment.reconciliation_required',$1,$2||id::text FROM users WHERE role='master' ON CONFLICT(event_key) DO NOTHING`, string(jsonBytes(M{"payment_id": p["id"]})), str(p, "id")+":manual-review:")
		})
		if err != nil {
			return nil, err
		}
		return nil, conflict("PAYMENT_RECONCILIATION_REQUIRED")
	}
	body := M{"amount": M{"value": rubles(num(p, "amount_minor")), "currency": "RUB"}, "capture": true, "confirmation": M{"type": "redirect", "return_url": a.C.ReturnURL}, "metadata": M{"payment_id": p["id"], "invoice_id": p["invoice_id"]}, "description": "Ясна: оплата счёта " + str(p, "invoice_id")}
	if str(p, "method") == "sbp" {
		body["payment_method_data"] = M{"type": "sbp"}
	}
	obj, e := a.provider(ctx, "POST", "/payments", str(p, "provider_key"), body)
	if e != nil {
		return nil, e
	}
	if e = a.applyProvider(ctx, obj); e != nil {
		return nil, e
	}
	fresh, e := one(ctx, a.DB, "SELECT * FROM payments WHERE id=$1", id)
	if e != nil {
		return nil, e
	}
	return paymentPublic(fresh), nil
}

func (a *App) applyProvider(ctx context.Context, obj M) error {
	_, e := a.system(ctx, func(r *R) (any, error) {
		id := str(obj, "id")
		meta := object(obj["metadata"])
		p, e := one(ctx, r.DB, "SELECT * FROM payments WHERE provider_payment_id=$1 OR (id::text=$2 AND provider_key IS NOT NULL)", id, str(meta, "payment_id"))
		if e != nil {
			return nil, fail(503, "PAYMENT_NOT_FOUND")
		}
		amount := object(obj["amount"])
		n, e := parseRubles(str(amount, "value"))
		if e != nil {
			return nil, e
		}
		recipient := object(obj["recipient"])
		if id == "" || n != num(p, "amount_minor") || str(amount, "currency") != "RUB" || str(recipient, "account_id") != a.C.ShopID || !yes(obj, "test") || (str(p, "provider_payment_id") != "" && str(p, "provider_payment_id") != id) || str(meta, "invoice_id") != str(p, "invoice_id") || str(meta, "payment_id") != str(p, "id") {
			return nil, fail(400, "PROVIDER_MISMATCH")
		}
		status := str(obj, "status")
		if !contains([]string{"pending", "waiting_for_capture", "succeeded", "canceled"}, status) {
			return nil, fail(503, "PROVIDER_STATUS")
		}
		m := M{"provider_payment_id": id}
		if link := str(object(obj["confirmation"]), "confirmation_url"); link != "" {
			u, e := url.Parse(link)
			if e != nil || u.Scheme != "https" || u.Host == "" {
				return nil, fail(503, "PROVIDER_URL")
			}
			m["payment_url"] = link
		}
		p, e = r.update("payments", str(p, "id"), m)
		if e != nil {
			return nil, e
		}
		if status == "succeeded" {
			if !yes(obj, "paid") {
				return nil, fail(400, "PROVIDER_NOT_PAID")
			}
			return r.credit(p, r.now().Format(time.RFC3339Nano))
		}
		if status == "canceled" && str(p, "status") != "succeeded" {
			return r.update("payments", str(p, "id"), M{"status": "canceled"})
		}
		return p, nil
	})
	return e
}

func (a *App) webhook(c *gin.Context) {
	r := &R{A: a, C: c}
	if a.C.ShopID == "" || a.C.Secret == "" {
		r.replyError(fail(503, "PAYMENT_NOT_CONFIGURED"))
		return
	}
	var v struct {
		Event  string `json:"event"`
		Object struct {
			ID string `json:"id"`
		} `json:"object"`
	}
	d := json.NewDecoder(c.Request.Body)
	if e := d.Decode(&v); e != nil || v.Object.ID == "" {
		r.replyError(fail(400, "INVALID_JSON"))
		return
	}
	if !contains([]string{"payment.succeeded", "payment.canceled", "payment.waiting_for_capture"}, v.Event) {
		c.JSON(200, M{"ok": true})
		return
	}
	if len(v.Object.ID) > 100 || strings.ContainsAny(v.Object.ID, "/?#") {
		r.replyError(fail(400, "INVALID_PROVIDER_ID"))
		return
	}
	obj, e := a.provider(c.Request.Context(), "GET", "/payments/"+url.PathEscape(v.Object.ID), "", nil)
	if e == nil {
		e = a.applyProvider(c.Request.Context(), obj)
	}
	if e != nil {
		r.replyError(e)
		return
	}
	c.JSON(200, M{"ok": true})
}

func (a *App) RunJobs(ctx context.Context) {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	a.jobs(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			a.jobs(ctx)
		}
	}
}

func (a *App) jobs(ctx context.Context) {
	jobCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	_, e := a.system(jobCtx, func(r *R) (any, error) {
		if e := r.exec(`INSERT INTO notifications(user_id,family_id,type,payload,event_key) SELECT user_id,family_id,type,payload,event_key FROM outbox WHERE delivered_at IS NULL ON CONFLICT(event_key) DO NOTHING`); e != nil {
			return nil, e
		}
		if e := r.exec("UPDATE outbox SET delivered_at=now() WHERE delivered_at IS NULL"); e != nil {
			return nil, e
		}
		if e := r.exec("UPDATE sessions SET revoked_at=now() WHERE revoked_at IS NULL AND expires_at<=now()"); e != nil {
			return nil, e
		}
		subs, e := many(ctx, r.DB, subSQL)
		if e != nil {
			return nil, e
		}
		for _, s := range subs {
			rule, e := r.rule(str(s, "group_id"), r.now().In(zone).Format("2006-01-02"))
			if e != nil {
				continue
			}
			until, _ := date(str(s, "valid_until"))
			days := int(until.Sub(r.now().In(zone)).Hours() / 24)
			typ := ""
			switch str(s, "status") {
			case "expired":
				typ = "subscription.expired"
			case "exhausted":
				typ = "subscription.exhausted"
			case "active":
				if days <= int(num(rule, "ending_notice_days")) || num(s, "lessons_remaining") <= num(rule, "ending_notice_lessons") {
					typ = "subscription.ending"
				}
			}
			if typ != "" {
				key := str(s, "id") + ":" + typ + ":" + r.now().In(zone).Format("2006-01-02")
				if e = r.notify(str(s, "family_id"), typ, key, M{"subscription_id": s["id"], "lessons_remaining": s["lessons_remaining"]}); e != nil {
					return nil, e
				}
				if e = r.exec(`INSERT INTO outbox(user_id,type,payload,event_key) SELECT id,$1,$2,$3||id::text FROM users WHERE role='master' ON CONFLICT(event_key) DO NOTHING`, typ, string(jsonBytes(M{"subscription_id": s["id"], "family_id": s["family_id"]})), key+":master:"); e != nil {
					return nil, e
				}
			}
		}
		invs, e := many(ctx, r.DB, "SELECT * FROM invoices WHERE status='open'")
		if e != nil {
			return nil, e
		}
		for _, i := range invs {
			if e = r.notify(str(i, "family_id"), "payment.required", str(i, "id")+":required:"+r.now().In(zone).Format("2006-01-02"), M{"invoice_id": i["id"]}); e != nil {
				return nil, e
			}
		}
		var bad int
		if e = r.DB.QueryRow(ctx, `SELECT count(*) FROM subscriptions s WHERE lessons_remaining<>(SELECT coalesce(sum(delta),0) FROM subscription_ledger WHERE subscription_id=s.id)`).Scan(&bad); e != nil {
			return nil, e
		}
		if bad > 0 {
			slog.Error("subscription balance invariant failed", "count", bad)
		}
		return nil, nil
	})
	if e != nil {
		slog.Error("jobs failed", "error", e)
	}
	if a.C.ShopID == "" {
		return
	}
	pending, e := many(jobCtx, a.DB, "SELECT id,provider_payment_id FROM payments WHERE status='pending' AND provider_key IS NOT NULL ORDER BY created_at LIMIT 30")
	if e != nil {
		return
	}
	for _, p := range pending {
		if jobCtx.Err() != nil {
			return
		}
		if id := str(p, "provider_payment_id"); id != "" {
			obj, err := a.provider(jobCtx, "GET", "/payments/"+url.PathEscape(id), "", nil)
			if err == nil {
				err = a.applyProvider(jobCtx, obj)
			}
			if err != nil {
				slog.Warn("payment reconciliation failed", "payment_id", p["id"], "error", err)
			}
		} else {
			if _, err := a.ensurePayment(jobCtx, str(p, "id")); err != nil {
				slog.Warn("payment creation retry failed", "payment_id", p["id"], "error", err)
			}
		}
	}
}
