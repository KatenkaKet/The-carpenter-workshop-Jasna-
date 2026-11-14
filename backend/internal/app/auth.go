package app

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/argon2"
)

func passwordHash(p string) string {
	salt := secret()[:32]
	h := argon2.IDKey([]byte(p), []byte(salt), 2, 64*1024, 2, 32)
	return salt + ":" + base64.RawStdEncoding.EncodeToString(h)
}

func passwordOK(encoded, p string) bool {
	parts := strings.Split(encoded, ":")
	if len(parts) != 2 {
		return false
	}
	got := argon2.IDKey([]byte(p), []byte(parts[0]), 2, 64*1024, 2, 32)
	want, e := base64.RawStdEncoding.DecodeString(parts[1])
	return e == nil && subtle.ConstantTimeCompare(want, got) == 1
}

func (a *App) Bootstrap(ctx context.Context) error {
	if a.C.MasterLogin == "" && a.C.MasterPassword == "" {
		return nil
	}
	if len(a.C.MasterPassword) < 12 || len(a.C.MasterLogin) < 3 {
		return errors.New("MASTER_LOGIN >=3 and MASTER_PASSWORD >=12 required")
	}
	_, e := a.DB.Exec(ctx, `INSERT INTO users(role,full_name,login,password_hash) VALUES('master','Мастер',$1,$2) ON CONFLICT(login) DO NOTHING`, a.C.MasterLogin, passwordHash(a.C.MasterPassword))
	return e
}

func (a *App) authenticate(c *gin.Context) (M, error) {
	parent, _ := c.Cookie("yasna_parent_session")
	master, _ := c.Cookie("yasna_master_session")
	if parent != "" && master != "" {
		return nil, conflict("SESSION_CONTEXT_CONFLICT")
	}
	token := valueOr(parent, master)
	if token == "" {
		return nil, fail(401, "UNAUTHORIZED")
	}
	m, e := one(c.Request.Context(), a.DB, `SELECT s.* FROM sessions s LEFT JOIN access_links l ON l.id=s.access_link_id WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now() AND (s.role='master' OR (l.revoked_at IS NULL AND l.expires_at>now()))`, hash(token))
	if e != nil {
		var ae *APIError
		if errors.As(e, &ae) {
			return nil, fail(401, "UNAUTHORIZED")
		}
		return nil, e
	}
	return m, nil
}

func (r *R) revalidate() error {
	var ok bool
	e := r.DB.QueryRow(r.ctx(), `SELECT EXISTS(SELECT 1 FROM sessions s LEFT JOIN access_links l ON l.id=s.access_link_id WHERE s.id=$1 AND s.revoked_at IS NULL AND s.expires_at>now() AND (s.role='master' OR (l.revoked_at IS NULL AND l.expires_at>now())))`, str(r.User, "id")).Scan(&ok)
	if e != nil {
		return e
	}
	if !ok {
		return fail(401, "UNAUTHORIZED")
	}
	return nil
}

func (a *App) origin(c *gin.Context) bool { return contains(a.C.Origins, c.GetHeader("Origin")) }

func (a *App) csrf(c *gin.Context, u M) bool {
	return a.origin(c) && subtle.ConstantTimeCompare([]byte(c.GetHeader("X-CSRF-Token")), []byte(str(u, "csrf_token"))) == 1
}

func (r *R) session(user, family, role, link string, expires time.Time) (any, error) {
	token, csrf := secret(), secret()
	_, e := r.insert("sessions", M{"user_id": user, "family_id": nullable(family), "role": role, "access_link_id": nullable(link), "token_hash": hash(token), "csrf_token": csrf, "expires_at": expires})
	if e != nil {
		return nil, e
	}
	http.SetCookie(r.C.Writer, &http.Cookie{Name: "yasna_" + role + "_session", Value: token, Path: "/", Secure: r.A.C.SecureCookies, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: int(expires.Sub(r.now()).Seconds())})
	r.Status = 200
	return M{"role": role, "family_id": nullable(family), "csrf_token": csrf}, nil
}

func login(r *R) (any, error) {
	if !r.A.origin(r.C) {
		return nil, fail(403, "ORIGIN")
	}
	if e := r.fields("login password", ""); e != nil {
		return nil, e
	}
	if len(str(r.Body, "password")) > 1024 {
		return nil, invalid("PASSWORD_LENGTH")
	}
	if !r.A.allow("login:"+str(r.Body, "login"), 10, time.Hour) {
		return nil, fail(429, "RATE_LIMIT")
	}
	u, e := one(r.ctx(), r.DB, "SELECT * FROM users WHERE role='master' AND login=$1", str(r.Body, "login"))
	if e != nil {
		passwordOK(strings.Repeat("0", 32)+":"+base64.RawStdEncoding.EncodeToString(make([]byte, 32)), str(r.Body, "password"))
		return nil, fail(401, "INVALID_CREDENTIALS")
	}
	if !passwordOK(str(u, "password_hash"), str(r.Body, "password")) {
		return nil, fail(401, "INVALID_CREDENTIALS")
	}
	return r.session(str(u, "id"), "", "master", "", r.now().Add(12*time.Hour))
}

func exchange(r *R) (any, error) {
	if !r.A.origin(r.C) {
		return nil, fail(403, "ORIGIN")
	}
	if e := r.fields("token", ""); e != nil {
		return nil, e
	}
	if len(str(r.Body, "token")) != 64 {
		return nil, fail(401, "INVALID_ACCESS_LINK")
	}
	l, e := one(r.ctx(), r.DB, `SELECT l.*,f.parent_user_id FROM access_links l JOIN families f ON f.id=l.family_id WHERE l.token_hash=$1 AND l.revoked_at IS NULL AND l.expires_at>now()`, hash(str(r.Body, "token")))
	if e != nil {
		return nil, fail(401, "INVALID_ACCESS_LINK")
	}
	until := r.now().Add(30 * 24 * time.Hour)
	if at(l, "expires_at").Before(until) {
		until = at(l, "expires_at")
	}
	return r.session(str(l, "parent_user_id"), str(l, "family_id"), "parent", str(l, "id"), until)
}

func me(r *R) (any, error) {
	u, e := one(r.ctx(), r.DB, "SELECT id,role,full_name FROM users WHERE id=$1", str(r.User, "user_id"))
	if e != nil {
		return nil, e
	}
	u["family_id"] = r.User["family_id"]
	u["csrf_token"] = r.User["csrf_token"]
	return u, nil
}

func logout(r *R) (any, error) {
	if !r.A.origin(r.C) {
		return nil, fail(403, "ORIGIN")
	}
	for _, role := range []string{"parent", "master"} {
		name := "yasna_" + role + "_session"
		v, _ := r.C.Cookie(name)
		if v != "" {
			if e := r.exec("UPDATE sessions SET revoked_at=now() WHERE token_hash=$1", hash(v)); e != nil {
				return nil, e
			}
		}
		http.SetCookie(r.C.Writer, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: r.A.C.SecureCookies, SameSite: http.SameSiteLaxMode})
	}
	r.Status = 204
	return nil, nil
}

func links(r *R) (any, error) {
	id := r.C.Param("id")
	if _, e := r.get("families", id); e != nil {
		return nil, e
	}
	return r.list("SELECT id,created_at,expires_at,revoked_at FROM access_links WHERE family_id=$1 ORDER BY created_at,id", id)
}

func issueLink(r *R) (any, error) {
	if e := r.fields("expires_at", ""); e != nil {
		return nil, e
	}
	id := r.C.Param("id")
	if _, e := r.get("families", id); e != nil {
		return nil, e
	}
	t, e := timestamp(str(r.Body, "expires_at"))
	if e != nil || !t.After(r.now()) {
		return nil, invalid("EXPIRY")
	}
	if e = r.exec("UPDATE access_links SET revoked_at=now() WHERE family_id=$1 AND revoked_at IS NULL", id); e != nil {
		return nil, e
	}
	if e = r.exec("UPDATE sessions SET revoked_at=now() WHERE family_id=$1", id); e != nil {
		return nil, e
	}
	token := secret()
	l, e := r.insert("access_links", M{"family_id": id, "token_hash": hash(token), "expires_at": t})
	if e != nil {
		return nil, e
	}
	if e = r.audit("access_link.issued", "access_links", str(l, "id"), nil, M{"family_id": id}, ""); e != nil {
		return nil, e
	}
	return M{"id": l["id"], "url": r.A.C.AppURL + "/access#token=" + token, "expires_at": t}, nil
}

func revokeLink(r *R) (any, error) {
	l, e := r.get("access_links", r.C.Param("linkId"))
	if e != nil {
		return nil, e
	}
	if str(l, "family_id") != r.C.Param("id") {
		return nil, fail(404, "NOT_FOUND")
	}
	if e = r.exec("UPDATE access_links SET revoked_at=coalesce(revoked_at,now()) WHERE id=$1", l["id"]); e != nil {
		return nil, e
	}
	if e = r.exec("UPDATE sessions SET revoked_at=now() WHERE access_link_id=$1", l["id"]); e != nil {
		return nil, e
	}
	return nil, r.audit("access_link.revoked", "access_links", str(l, "id"), nil, nil, "")
}

func (a *App) allow(key string, max int, d time.Duration) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.Now()
	if a.rates == nil {
		a.rates = map[string]*rateWindow{}
	}
	if len(a.rates) > 10000 {
		for k, v := range a.rates {
			if now.After(v.Until) {
				delete(a.rates, k)
			}
		}
		if len(a.rates) > 10000 {
			return false
		}
	}
	v := a.rates[key]
	if v == nil || now.After(v.Until) {
		v = &rateWindow{Until: now.Add(d)}
		a.rates[key] = v
	}
	v.Count++
	return v.Count <= max
}

func (a *App) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := newID()
		c.Set("request_id", id)
		c.Header("X-Request-Id", id)
		c.Header("Cache-Control", "no-store")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "no-referrer")
		if a.origin(c) {
			c.Header("Access-Control-Allow-Origin", c.GetHeader("Origin"))
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token, Idempotency-Key")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if c.Request.Method == "OPTIONS" {
			if !a.origin(c) {
				c.AbortWithStatus(403)
			} else {
				c.AbortWithStatus(204)
			}
			return
		}
		limit := 300
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/auth/") {
			limit = 20
		}
		if !a.allow(c.ClientIP()+":"+fmt.Sprint(limit), limit, time.Minute) {
			c.Header("Retry-After", "60")
			(&R{C: c}).replyError(fail(429, "RATE_LIMIT"))
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 11<<20)
		c.Next()
	}
}
