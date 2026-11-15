package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type M map[string]any

type Config struct {
	DatabaseURL, Addr, AppURL, MasterLogin, MasterPassword, ShopID, Secret, ReturnURL string
	Origins, ReceiptHosts                                                             []string
	SecureCookies, PaymentTest, PaymentStub                                           bool
}

func env(k, d string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return d
}

func LoadConfig() (Config, error) {
	c := Config{DatabaseURL: os.Getenv("DATABASE_URL"), Addr: ":" + env("PORT", "8080"), AppURL: env("APP_URL", "https://localhost:3000"), MasterLogin: os.Getenv("MASTER_LOGIN"), MasterPassword: os.Getenv("MASTER_PASSWORD"), ShopID: os.Getenv("YOOKASSA_SHOP_ID"), Secret: os.Getenv("YOOKASSA_SECRET_KEY"), ReturnURL: os.Getenv("PAYMENT_RETURN_URL"), SecureCookies: env("COOKIE_SECURE", "true") == "true", PaymentTest: true}
	c.PaymentStub = env("APP_ENV", "production") == "development" && env("PAYMENT_STUB", "false") == "true"
	c.Origins = strings.Split(env("ALLOWED_ORIGINS", c.AppURL), ",")
	c.ReceiptHosts = strings.Split(env("RECEIPT_ALLOWED_HOSTS", "lknpd.nalog.ru"), ",")
	if c.DatabaseURL == "" {
		u := url.URL{Scheme: "postgres", Host: env("DB_HOST", "localhost") + ":" + env("DB_PORT", "5432"), Path: env("DB_NAME", "jasna"), User: url.UserPassword(env("DB_USER", "jasna"), env("DB_PASSWORD", "jasna_local"))}
		q := u.Query()
		q.Set("sslmode", env("DB_SSLMODE", "disable"))
		u.RawQuery = q.Encode()
		c.DatabaseURL = u.String()
	}
	if !c.SecureCookies && env("APP_ENV", "production") != "development" {
		return c, errors.New("COOKIE_SECURE=false requires APP_ENV=development")
	}
	if env("PAYMENT_TEST", "true") != "true" {
		return c, errors.New("real payments disabled: prototype supports test shop only")
	}
	for _, o := range c.Origins {
		u, e := url.Parse(o)
		if e != nil || u.Host == "" || (u.Scheme != "https" && !(u.Scheme == "http" && !c.SecureCookies)) {
			return c, errors.New("invalid ALLOWED_ORIGINS")
		}
	}
	return c, nil
}

type App struct {
	DB    *pgxpool.Pool
	C     Config
	HTTP  *http.Client
	Now   func() time.Time
	mu    sync.Mutex
	rates map[string]*rateWindow
}

type rateWindow struct {
	Count int
	Until time.Time
}

type Q interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type R struct {
	A      *App
	C      *gin.Context
	DB     Q
	User   M
	Body   M
	Raw    []byte
	Status int
}

type Handler func(*R) (any, error)

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string { return e.Message }
func fail(s int, c string) error  { return &APIError{s, c, c} }
func conflict(c string) error     { return fail(409, c) }
func invalid(c string) error      { return fail(422, c) }

func newID() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func secret() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}

func hash(s string) string     { v := sha256.Sum256([]byte(s)); return hex.EncodeToString(v[:]) }
func str(m M, k string) string { v, _ := m[k].(string); return v }

func num(m M, k string) int64 {
	switch x := m[k].(type) {
	case json.Number:
		n, _ := x.Int64()
		return n
	case int:
		return int64(x)
	case int64:
		return x
	case float64:
		return int64(x)
	}
	return 0
}

func yes(m M, k string) bool { v, _ := m[k].(bool); return v }

func object(v any) M {
	switch x := v.(type) {
	case M:
		return x
	case map[string]any:
		return M(x)
	}
	return M{}
}

func jsonBytes(v any) []byte { b, _ := json.Marshal(v); return b }

func decode(b []byte) (M, error) {
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var m M
	e := d.Decode(&m)
	return m, e
}

func one(ctx context.Context, q Q, sql string, args ...any) (M, error) {
	var b []byte
	query := "SELECT to_jsonb(t) FROM (" + sql + ") t"
	if strings.HasPrefix(sql, "INSERT ") || strings.HasPrefix(sql, "UPDATE ") {
		query = "WITH t AS (" + sql + ") SELECT to_jsonb(t) FROM t"
	}
	e := q.QueryRow(ctx, query, args...).Scan(&b)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, fail(404, "NOT_FOUND")
	}
	if e != nil {
		return nil, e
	}
	return decode(b)
}

func many(ctx context.Context, q Q, sql string, args ...any) ([]M, error) {
	rows, e := q.Query(ctx, "SELECT to_jsonb(t) FROM ("+sql+") t", args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []M{}
	for rows.Next() {
		var b []byte
		if e = rows.Scan(&b); e != nil {
			return nil, e
		}
		m, e := decode(b)
		if e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *R) ctx() context.Context { return r.C.Request.Context() }

func (r *R) get(table, id string) (M, error) {
	return one(r.ctx(), r.DB, "SELECT * FROM "+table+" WHERE id=$1", id)
}

func (r *R) exec(sql string, args ...any) error { _, e := r.DB.Exec(r.ctx(), sql, args...); return e }

func (r *R) insert(table string, m M) (M, error) {
	if m["id"] == nil {
		m["id"] = newID()
	}
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	values := []string{}
	args := []any{}
	for i, k := range keys {
		values = append(values, fmt.Sprintf("$%d", i+1))
		args = append(args, dbValue(m[k]))
	}
	return one(r.ctx(), r.DB, "INSERT INTO "+table+" ("+strings.Join(keys, ",")+") VALUES ("+strings.Join(values, ",")+") RETURNING *", args...)
}

func dbValue(v any) any {
	switch x := v.(type) {
	case M, map[string]any, []any:
		return string(jsonBytes(v))
	case json.Number:
		if n, e := x.Int64(); e == nil {
			return n
		}
	}
	return v
}

func (r *R) update(table, id string, m M) (M, error) {
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sets := []string{"version=version+1"}
	args := []any{id}
	for _, k := range keys {
		args = append(args, dbValue(m[k]))
		sets = append(sets, fmt.Sprintf("%s=$%d", k, len(args)))
	}
	return one(r.ctx(), r.DB, "UPDATE "+table+" SET "+strings.Join(sets, ",")+" WHERE id=$1 RETURNING *", args...)
}

func (r *R) fields(required string, optional string) error {
	allowed := strings.Fields(required + " " + optional)
	for k := range r.Body {
		if !contains(allowed, k) {
			return invalid("UNKNOWN_FIELD_" + k)
		}
	}
	for _, k := range strings.Fields(required) {
		if r.Body[k] == nil {
			return invalid("REQUIRED_" + k)
		}
	}
	return nil
}

func contains(a []string, s string) bool {
	for _, x := range a {
		if x == s {
			return true
		}
	}
	return false
}

func (r *R) family(id string) error {
	if str(r.User, "role") == "parent" && str(r.User, "family_id") != id {
		return fail(404, "NOT_FOUND")
	}
	return nil
}

func (r *R) owned(table, id string) (M, error) {
	m, e := r.get(table, id)
	if e != nil {
		return nil, e
	}
	if e = r.family(str(m, "family_id")); e != nil {
		return nil, e
	}
	return m, nil
}

func (r *R) version(m M) error {
	if num(r.Body, "expected_version") != num(m, "version") {
		return conflict("VERSION_CONFLICT")
	}
	return nil
}

func (r *R) audit(event, table, id string, before, after any, reason string) error {
	_, e := r.insert("audit_log", M{"event_type": event, "actor_user_id": nullable(str(r.User, "user_id")), "actor_role": valueOr(str(r.User, "role"), "system"), "resource_type": table, "resource_id": nullable(id), "before_state": before, "after_state": after, "reason": reason, "request_id": r.C.GetString("request_id")})
	return e
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func valueOr(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func (r *R) notify(family, typ, key string, payload M) error {
	return r.exec(`INSERT INTO outbox(user_id,family_id,type,payload,event_key) SELECT parent_user_id,id,$2,$3,$4 FROM families WHERE id=$1 ON CONFLICT(event_key) DO NOTHING`, family, typ, string(jsonBytes(payload)), key)
}

func (r *R) list(sql string, args ...any) (any, error) {
	limit, offset := 50, 0
	var e error
	if v := r.C.Query("limit"); v != "" {
		limit, e = strconv.Atoi(v)
		if e != nil || limit < 1 || limit > 200 {
			return nil, invalid("LIMIT")
		}
	}
	if v := r.C.Query("offset"); v != "" {
		offset, e = strconv.Atoi(v)
		if e != nil || offset < 0 {
			return nil, invalid("OFFSET")
		}
	}
	var total int64
	if e = r.DB.QueryRow(r.ctx(), "SELECT count(*) FROM ("+sql+") t", args...).Scan(&total); e != nil {
		return nil, e
	}
	args = append(args, limit, offset)
	items, e := many(r.ctx(), r.DB, sql+fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	return M{"items": items, "total": total, "limit": limit, "offset": offset}, e
}

func date(s string) (time.Time, error) {
	t, e := time.ParseInLocation("2006-01-02", s, zone)
	if e != nil {
		return t, invalid("DATE")
	}
	return t, nil
}

var zone = func() *time.Location {
	l, e := time.LoadLocation("Asia/Yekaterinburg")
	if e != nil {
		panic(e)
	}
	return l
}()

func timestamp(s string) (time.Time, error) {
	t, e := time.Parse(time.RFC3339Nano, s)
	if e != nil {
		return t, invalid("DATETIME")
	}
	return t, nil
}

func at(m M, k string) time.Time { t, _ := timestamp(str(m, k)); return t }
func (r *R) now() time.Time      { return r.A.Now() }

func (r *R) replyError(e error) {
	if r.C.Writer.Written() {
		slog.Error("stream interrupted", "request_id", r.C.GetString("request_id"), "error", e)
		r.C.Abort()
		return
	}
	var a *APIError
	if !errors.As(e, &a) {
		var pg *pgconn.PgError
		if errors.As(e, &pg) {
			switch pg.Code {
			case "23505":
				a = &APIError{409, "CONFLICT", "Запись уже существует"}
			case "23503":
				a = &APIError{404, "NOT_FOUND", "Объект не найден"}
			case "23514", "23502", "22P02", "22007", "22008", "22003":
				a = &APIError{422, "VALIDATION_ERROR", "Некорректные данные"}
			}
		}
		if a == nil {
			slog.Error("request failed", "request_id", r.C.GetString("request_id"), "error", e)
			a = &APIError{500, "INTERNAL_ERROR", "Внутренняя ошибка"}
		}
	}
	r.C.AbortWithStatusJSON(a.Status, M{"error": M{"code": a.Code, "message": a.Message}, "request_id": r.C.GetString("request_id")})
}

func (a *App) handle(role string, write, idem bool, h Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		r := &R{A: a, C: c, DB: a.DB, Status: 200, Body: M{}}
		if c.Request.Method == "POST" {
			r.Status = 201
		}
		if c.Request.Method == "DELETE" {
			r.Status = 204
		}
		if role != "guest" {
			u, e := a.authenticate(c)
			if e != nil {
				r.replyError(e)
				return
			}
			r.User = u
			if role != "any" && str(u, "role") != role {
				r.replyError(fail(403, "FORBIDDEN"))
				return
			}
			if write && !a.csrf(c, u) {
				r.replyError(fail(403, "CSRF"))
				return
			}
		}
		if strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
			b, e := io.ReadAll(c.Request.Body)
			if e != nil {
				r.replyError(fail(413, "BODY_TOO_LARGE"))
				return
			}
			r.Raw = b
			if len(b) > 0 {
				d := json.NewDecoder(bytes.NewReader(b))
				d.UseNumber()
				if e = d.Decode(&r.Body); e != nil || r.Body == nil {
					r.replyError(fail(400, "INVALID_JSON"))
					return
				}
				var extra any
				if d.Decode(&extra) != io.EOF {
					r.replyError(fail(400, "INVALID_JSON"))
					return
				}
			}
		} else if write && c.Request.ContentLength > 0 && !strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
			r.replyError(fail(415, "JSON_REQUIRED"))
			return
		}
		var result any
		var e error
		if write {
			tx, err := a.DB.Begin(c.Request.Context())
			if err != nil {
				r.replyError(err)
				return
			}
			defer tx.Rollback(c.Request.Context())
			r.DB = tx
			// A small single-workshop deployment serializes mutations across all replicas.
			// All transaction entry points, including workers, use this same advisory lock.
			if e = r.exec("SELECT pg_advisory_xact_lock(7341022)"); e != nil {
				r.replyError(e)
				return
			}
			if role != "guest" {
				if e = r.revalidate(); e != nil {
					r.replyError(e)
					return
				}
			}
			key := c.GetHeader("Idempotency-Key")
			if idem && (len(key) < 8 || len(key) > 200) {
				r.replyError(invalid("IDEMPOTENCY_KEY_REQUIRED"))
				return
			}
			if idem {
				saved, err := one(r.ctx(), tx, "SELECT * FROM idempotency WHERE user_id=$1 AND route=$2 AND key=$3", str(r.User, "user_id"), c.Request.URL.Path, key)
				if err == nil {
					if str(saved, "body_hash") != hash(string(r.Raw)) {
						r.replyError(conflict("IDEMPOTENCY_CONFLICT"))
						return
					}
					tx.Rollback(r.ctx())
					result = saved["response"]
					if strings.HasSuffix(c.FullPath(), "/payments/intent") {
						result, err = a.ensurePayment(r.ctx(), str(object(result), "id"))
						if err != nil {
							r.replyError(err)
							return
						}
					}
					c.JSON(int(num(saved, "status_code")), result)
					return
				}
				var ae *APIError
				if !errors.As(err, &ae) || ae.Status != 404 {
					r.replyError(err)
					return
				}
			}
			result, e = h(r)
			if e == nil && idem {
				_, e = r.insert("idempotency", M{"user_id": str(r.User, "user_id"), "route": c.Request.URL.Path, "key": key, "body_hash": hash(string(r.Raw)), "status_code": r.Status, "response": result})
			}
			if e == nil {
				e = tx.Commit(r.ctx())
			}
		} else {
			result, e = h(r)
		}
		if e == nil && strings.HasSuffix(c.FullPath(), "/payments/intent") {
			result, e = a.ensurePayment(r.ctx(), str(object(result), "id"))
		}
		if e != nil {
			r.replyError(e)
			return
		}
		if !c.Writer.Written() {
			if r.Status == 204 {
				c.Status(204)
			} else {
				c.JSON(r.Status, result)
			}
		}
	}
}
