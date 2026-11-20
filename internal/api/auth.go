package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

type authUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
type userKey struct{}

func (s *Server) initAuth() error {
	if s.db == nil {
		return nil
	}
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS investor_users (id TEXT PRIMARY KEY, name TEXT NOT NULL, email TEXT NOT NULL UNIQUE, password_salt TEXT NOT NULL, password_hash TEXT NOT NULL, signup_ip TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS investor_sessions (token_hash TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES investor_users(id), expires_at TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE INDEX IF NOT EXISTS idx_investor_sessions_user ON investor_sessions(user_id)`,
	} {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
func clientIP(r *http.Request) string { // Never trust client-supplied X-Forwarded-For by default.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return net.ParseIP(host).String()
}
func passwordHash(password string, salt []byte) string {
	return hex.EncodeToString(pbkdf2.Key([]byte(password), salt, 310000, 32, sha256.New))
}
func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	if !s.requireDB(w) {
		return
	}
	var v struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if readJSON(r, &v) != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request"})
		return
	}
	v.Name = strings.TrimSpace(v.Name)
	v.Email = strings.ToLower(strings.TrimSpace(v.Email))
	if !validCustomer(v.Name, v.Email) || len(v.Password) < 12 || len(v.Password) > 128 {
		writeJSON(w, 400, map[string]string{"error": "Provide a valid name, email and password of 12–128 characters"})
		return
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		writeJSON(w, 500, map[string]string{"error": "registration unavailable"})
		return
	}
	id, err := newOpaqueID(16)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "registration unavailable"})
		return
	}
	ip := clientIP(r)
	if ip == "" {
		ip = "unknown"
	}
	_, err = s.db.ExecContext(r.Context(), `INSERT INTO investor_users(id,name,email,password_salt,password_hash,signup_ip) VALUES(?,?,?,?,?,?)`, id, v.Name, v.Email, hex.EncodeToString(salt), passwordHash(v.Password, salt), ip)
	if err != nil {
		writeJSON(w, 409, map[string]string{"error": "Account could not be created; email may already be registered"})
		return
	}
	s.issueSession(w, r, authUser{ID: id, Name: v.Name, Email: v.Email})
}
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.requireDB(w) {
		return
	}
	var v struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if readJSON(r, &v) != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request"})
		return
	}
	var u authUser
	var saltHex, stored string
	err := s.db.QueryRowContext(r.Context(), `SELECT id,name,email,password_salt,password_hash FROM investor_users WHERE email=?`, strings.ToLower(strings.TrimSpace(v.Email))).Scan(&u.ID, &u.Name, &u.Email, &saltHex, &stored)
	salt, _ := hex.DecodeString(saltHex)
	if err != nil || len(salt) != 16 || subtle.ConstantTimeCompare([]byte(passwordHash(v.Password, salt)), []byte(stored)) != 1 {
		writeJSON(w, 401, map[string]string{"error": "Invalid email or password"})
		return
	}
	s.issueSession(w, r, u)
}
func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, u authUser) {
	token, err := newOpaqueID(32)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "session unavailable"})
		return
	}
	expires := time.Now().UTC().Add(24 * time.Hour)
	if _, err = s.db.ExecContext(r.Context(), `INSERT INTO investor_sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, tokenHash(token), u.ID, expires.Format(time.RFC3339Nano)); err != nil {
		writeJSON(w, 500, map[string]string{"error": "session unavailable"})
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "vantiqor_session", Value: token, Path: "/", HttpOnly: true, Secure: r.TLS != nil || os.Getenv("VANTIQOR_HTTPS_ONLY") == "true", SameSite: http.SameSiteStrictMode, Expires: expires, MaxAge: 86400})
	writeJSON(w, 200, map[string]any{"user": u})
}
func (s *Server) currentUser(r *http.Request) (authUser, bool) {
	if s.db == nil {
		return authUser{}, false
	}
	c, err := r.Cookie("vantiqor_session")
	if err != nil {
		return authUser{}, false
	}
	var u authUser
	err = s.db.QueryRowContext(r.Context(), `SELECT u.id,u.name,u.email FROM investor_users u JOIN investor_sessions ss ON ss.user_id=u.id WHERE ss.token_hash=? AND ss.expires_at>?`, tokenHash(c.Value), time.Now().UTC().Format(time.RFC3339Nano)).Scan(&u.ID, &u.Name, &u.Email)
	return u, err == nil
}
func (s *Server) requireInvestor(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.currentUser(r)
		if !ok {
			writeJSON(w, 401, map[string]string{"error": "Sign in required"})
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey{}, u)))
	}
}
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, _ := s.currentUser(r)
	writeJSON(w, 200, map[string]any{"user": u})
}
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("vantiqor_session"); err == nil && s.db != nil {
		_, _ = s.db.ExecContext(r.Context(), `DELETE FROM investor_sessions WHERE token_hash=?`, tokenHash(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: "vantiqor_session", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	writeJSON(w, 200, map[string]bool{"ok": true})
}
func (s *Server) handleMyDeposits(w http.ResponseWriter, r *http.Request) {
	u := r.Context().Value(userKey{}).(authUser)
	rows, err := s.db.QueryContext(r.Context(), `SELECT d.id,d.amount,d.currency,d.deposit_method,d.status,d.created_at FROM deposit_applications d JOIN support_threads t ON d.thread_id=t.id WHERE t.customer_email=? ORDER BY d.created_at DESC LIMIT 100`, u.Email)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Unable to load requests"})
		return
	}
	defer rows.Close()
	type item struct {
		ID        string  `json:"id"`
		Amount    float64 `json:"amount"`
		Currency  string  `json:"currency"`
		Method    string  `json:"method"`
		Status    string  `json:"status"`
		CreatedAt string  `json:"created_at"`
	}
	out := []item{}
	for rows.Next() {
		var x item
		if rows.Scan(&x.ID, &x.Amount, &x.Currency, &x.Method, &x.Status, &x.CreatedAt) == nil {
			out = append(out, x)
		}
	}
	writeJSON(w, 200, out)
}

var _ = sql.ErrNoRows
var _ = json.Valid

// Reject cross-site state-changing requests. SameSite cookies provide a second defense.
func (s *Server) sameOriginWrites(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete {
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || !strings.EqualFold(u.Host, r.Host) {
					writeJSON(w, 403, map[string]string{"error": "cross-origin request denied"})
					return
				}
			}
			if site := r.Header.Get("Sec-Fetch-Site"); site == "cross-site" {
				writeJSON(w, 403, map[string]string{"error": "cross-site request denied"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
