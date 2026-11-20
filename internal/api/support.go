package api

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type supportThread struct {
	ID            string `json:"id"`
	CustomerName  string `json:"customer_name"`
	CustomerEmail string `json:"customer_email"`
	Subject       string `json:"subject"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type supportMessage struct {
	ID        int64  `json:"id"`
	ThreadID  string `json:"thread_id"`
	Sender    string `json:"sender"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func (s *Server) requireDB(w http.ResponseWriter) bool {
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "support storage is not configured"})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func readJSON(r *http.Request, target any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 16*1024))
	dec.DisallowUnknownFields()
	return dec.Decode(target)
}

func newOpaqueID(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func tokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func validCustomer(name, email string) bool {
	return len(strings.TrimSpace(name)) >= 2 && len(name) <= 120 && emailPattern.MatchString(email) && len(email) <= 254
}

func (s *Server) createThreadTx(tx *sql.Tx, name, email, subject, message string) (string, string, error) {
	id, err := newOpaqueID(16)
	if err != nil {
		return "", "", err
	}
	token, err := newOpaqueID(32)
	if err != nil {
		return "", "", err
	}
	_, err = tx.Exec(`INSERT INTO support_threads(id,access_token_hash,customer_name,customer_email,subject) VALUES(?,?,?,?,?)`, id, tokenHash(token), strings.TrimSpace(name), strings.TrimSpace(email), strings.TrimSpace(subject))
	if err != nil {
		return "", "", err
	}
	_, err = tx.Exec(`INSERT INTO support_messages(thread_id,sender,content) VALUES(?,?,?)`, id, "customer", strings.TrimSpace(message))
	if err != nil {
		return "", "", err
	}
	return id, token, nil
}

func (s *Server) handleCreateDepositApplication(w http.ResponseWriter, r *http.Request) {
	if !s.requireDB(w) {
		return
	}
	var req struct {
		Name     string  `json:"name"`
		Email    string  `json:"email"`
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
		Method   string  `json:"method"`
		Message  string  `json:"message"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request"})
		return
	}
	u := r.Context().Value(userKey{}).(authUser)
	req.Name, req.Email = u.Name, u.Email
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.Method = strings.TrimSpace(req.Method)
	if !validCustomer(req.Name, req.Email) || req.Amount < 1 || req.Amount > 100000000 || req.Currency != "USD" || (req.Method != "Bank transfer" && req.Method != "Crypto" && req.Method != "Other") || len(req.Message) > 2000 {
		writeJSON(w, 400, map[string]string{"error": "enter valid contact details, a USD amount, deposit method, and optional message"})
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not create application"})
		return
	}
	defer tx.Rollback()
	threadID, token, err := s.createThreadTx(tx, req.Name, req.Email, "Deposit application", fmt.Sprintf("I would like to apply for a deposit of $%.2f USD. Preferred method: %s. %s", req.Amount, req.Method, strings.TrimSpace(req.Message)))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not create application"})
		return
	}
	appID, err := newOpaqueID(12)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not create application"})
		return
	}
	_, err = tx.Exec(`INSERT INTO deposit_applications(id,thread_id,amount,currency,deposit_method) VALUES(?,?,?,?,?)`, appID, threadID, req.Amount, req.Currency, req.Method)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not create application"})
		return
	}
	if err = tx.Commit(); err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not save application"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"application_id": appID, "thread_id": threadID, "access_token": token, "status": "pending", "message": "Your application has been sent to the VANTIQOR admin team. Save your support access code to continue the conversation."})
}

func (s *Server) handleCreateSupportThread(w http.ResponseWriter, r *http.Request) {
	if !s.requireDB(w) {
		return
	}
	var req struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Subject string `json:"subject"`
		Message string `json:"message"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request"})
		return
	}
	req.Subject = strings.TrimSpace(req.Subject)
	req.Message = strings.TrimSpace(req.Message)
	if !validCustomer(req.Name, req.Email) || len(req.Subject) < 3 || len(req.Subject) > 120 || len(req.Message) < 2 || len(req.Message) > 4000 {
		writeJSON(w, 400, map[string]string{"error": "please provide valid contact details, subject and message"})
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not open support chat"})
		return
	}
	defer tx.Rollback()
	id, token, err := s.createThreadTx(tx, req.Name, req.Email, req.Subject, req.Message)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not open support chat"})
		return
	}
	if err = tx.Commit(); err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not save support chat"})
		return
	}
	writeJSON(w, 201, map[string]any{"thread_id": id, "access_token": token, "status": "open"})
}

func (s *Server) authorizeCustomerThread(r *http.Request) (string, error) {
	id := chi.URLParam(r, "id")
	token := r.Header.Get("X-Support-Token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if id == "" || token == "" {
		return "", errors.New("missing support credentials")
	}
	var stored string
	if err := s.db.QueryRowContext(r.Context(), `SELECT access_token_hash FROM support_threads WHERE id=?`, id).Scan(&stored); err != nil {
		return "", err
	}
	if stored != tokenHash(token) {
		return "", errors.New("invalid support credentials")
	}
	return id, nil
}

func (s *Server) handleGetCustomerMessages(w http.ResponseWriter, r *http.Request) {
	if !s.requireDB(w) {
		return
	}
	id, err := s.authorizeCustomerThread(r)
	if err != nil {
		writeJSON(w, 403, map[string]string{"error": "invalid support access code"})
		return
	}
	messages, err := s.readMessages(r, id)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not load messages"})
		return
	}
	writeJSON(w, 200, messages)
}

func (s *Server) handlePostCustomerMessage(w http.ResponseWriter, r *http.Request) {
	if !s.requireDB(w) {
		return
	}
	id, err := s.authorizeCustomerThread(r)
	if err != nil {
		writeJSON(w, 403, map[string]string{"error": "invalid support access code"})
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if readJSON(r, &req) != nil || len(strings.TrimSpace(req.Content)) < 1 || len(req.Content) > 4000 {
		writeJSON(w, 400, map[string]string{"error": "message must be between 1 and 4000 characters"})
		return
	}
	if _, err = s.db.ExecContext(r.Context(), `INSERT INTO support_messages(thread_id,sender,content) VALUES(?,?,?)`, id, "customer", strings.TrimSpace(req.Content)); err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not send message"})
		return
	}
	_, _ = s.db.ExecContext(r.Context(), `UPDATE support_threads SET updated_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	writeJSON(w, 201, map[string]string{"status": "sent"})
}

func (s *Server) readMessages(r *http.Request, id string) ([]supportMessage, error) {
	rows, err := s.db.QueryContext(r.Context(), `SELECT id,thread_id,sender,content,created_at FROM support_messages WHERE thread_id=? ORDER BY id ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []supportMessage{}
	for rows.Next() {
		var m supportMessage
		if err = rows.Scan(&m.ID, &m.ThreadID, &m.Sender, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secret := os.Getenv("ADMIN_API_KEY")
		if secret == "" {
			writeJSON(w, 503, map[string]string{"error": "admin access is not configured"})
			return
		}
		provided := r.Header.Get("X-Admin-Key")
		if provided == "" || provided != secret {
			writeJSON(w, 401, map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r)
	}
}

func (s *Server) handleAdminListThreads(w http.ResponseWriter, r *http.Request) {
	if !s.requireDB(w) {
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT id,customer_name,customer_email,subject,status,created_at,updated_at FROM support_threads ORDER BY updated_at DESC LIMIT 250`)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not load inbox"})
		return
	}
	defer rows.Close()
	out := []supportThread{}
	for rows.Next() {
		var t supportThread
		if rows.Scan(&t.ID, &t.CustomerName, &t.CustomerEmail, &t.Subject, &t.Status, &t.CreatedAt, &t.UpdatedAt) != nil {
			continue
		}
		out = append(out, t)
	}
	writeJSON(w, 200, out)
}
func (s *Server) handleAdminGetMessages(w http.ResponseWriter, r *http.Request) {
	if !s.requireDB(w) {
		return
	}
	out, err := s.readMessages(r, chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not load messages"})
		return
	}
	writeJSON(w, 200, out)
}
func (s *Server) handleAdminPostMessage(w http.ResponseWriter, r *http.Request) {
	if !s.requireDB(w) {
		return
	}
	id := chi.URLParam(r, "id")
	var req struct {
		Content string `json:"content"`
	}
	if readJSON(r, &req) != nil || len(strings.TrimSpace(req.Content)) < 1 || len(req.Content) > 4000 {
		writeJSON(w, 400, map[string]string{"error": "message must be between 1 and 4000 characters"})
		return
	}
	result, err := s.db.ExecContext(r.Context(), `INSERT INTO support_messages(thread_id,sender,content) SELECT ?, 'admin', ? WHERE EXISTS(SELECT 1 FROM support_threads WHERE id=?)`, id, strings.TrimSpace(req.Content), id)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not send reply"})
		return
	}
	if count, _ := result.RowsAffected(); count == 0 {
		writeJSON(w, 404, map[string]string{"error": "support thread not found"})
		return
	}
	_, _ = s.db.ExecContext(r.Context(), `UPDATE support_threads SET updated_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	writeJSON(w, 201, map[string]string{"status": "sent"})
}
func (s *Server) handleAdminUpdateThread(w http.ResponseWriter, r *http.Request) {
	if !s.requireDB(w) {
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if readJSON(r, &req) != nil || (req.Status != "open" && req.Status != "closed") {
		writeJSON(w, 400, map[string]string{"error": "status must be open or closed"})
		return
	}
	res, err := s.db.ExecContext(r.Context(), `UPDATE support_threads SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, req.Status, chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not update thread"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeJSON(w, 404, map[string]string{"error": "thread not found"})
		return
	}
	writeJSON(w, 200, map[string]string{"status": req.Status})
}
func (s *Server) handleAdminListDeposits(w http.ResponseWriter, r *http.Request) {
	if !s.requireDB(w) {
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT d.id,d.thread_id,t.customer_name,t.customer_email,d.amount,d.currency,d.deposit_method,d.status,COALESCE(d.admin_note,''),d.created_at FROM deposit_applications d JOIN support_threads t ON t.id=d.thread_id ORDER BY d.created_at DESC LIMIT 250`)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not load applications"})
		return
	}
	defer rows.Close()
	type item struct {
		ID        string  `json:"id"`
		ThreadID  string  `json:"thread_id"`
		Name      string  `json:"name"`
		Email     string  `json:"email"`
		Amount    float64 `json:"amount"`
		Currency  string  `json:"currency"`
		Method    string  `json:"method"`
		Status    string  `json:"status"`
		Note      string  `json:"admin_note"`
		CreatedAt string  `json:"created_at"`
	}
	out := []item{}
	for rows.Next() {
		var x item
		if rows.Scan(&x.ID, &x.ThreadID, &x.Name, &x.Email, &x.Amount, &x.Currency, &x.Method, &x.Status, &x.Note, &x.CreatedAt) == nil {
			out = append(out, x)
		}
	}
	writeJSON(w, 200, out)
}
func (s *Server) handleAdminUpdateDeposit(w http.ResponseWriter, r *http.Request) {
	if !s.requireDB(w) {
		return
	}
	var req struct {
		Status string `json:"status"`
		Note   string `json:"admin_note"`
	}
	if readJSON(r, &req) != nil || (req.Status != "pending" && req.Status != "approved" && req.Status != "rejected") || len(req.Note) > 2000 {
		writeJSON(w, 400, map[string]string{"error": "status must be pending, approved or rejected"})
		return
	}
	appID := chi.URLParam(r, "id")
	var threadID string
	if err := s.db.QueryRowContext(r.Context(), `SELECT thread_id FROM deposit_applications WHERE id=?`, appID).Scan(&threadID); err != nil {
		writeJSON(w, 404, map[string]string{"error": "application not found"})
		return
	}
	res, err := s.db.ExecContext(r.Context(), `UPDATE deposit_applications SET status=?,admin_note=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, req.Status, strings.TrimSpace(req.Note), appID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not update application"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeJSON(w, 404, map[string]string{"error": "application not found"})
		return
	}
	statusMessage := "Your deposit application is now " + req.Status + "."
	if strings.TrimSpace(req.Note) != "" {
		statusMessage += " Admin note: " + strings.TrimSpace(req.Note)
	}
	_, _ = s.db.ExecContext(r.Context(), `INSERT INTO support_messages(thread_id,sender,content) VALUES(?,?,?)`, threadID, "admin", statusMessage)
	_, _ = s.db.ExecContext(r.Context(), `UPDATE support_threads SET updated_at=CURRENT_TIMESTAMP WHERE id=?`, threadID)
	writeJSON(w, 200, map[string]string{"status": req.Status})
}
