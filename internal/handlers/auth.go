package handlers

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/4rch3/portal/internal/auth"
	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/metrics"
	"github.com/4rch3/portal/internal/middleware"
	"github.com/4rch3/portal/internal/models"
)

type AuthHandler struct {
	db            *database.DB
	authSvc       *auth.Service
	sessionCookie string
	sessionExpiry time.Duration
}

func NewAuthHandler(db *database.DB, authSvc *auth.Service, sessionCookie string, sessionExpiry time.Duration) *AuthHandler {
	return &AuthHandler{db: db, authSvc: authSvc, sessionCookie: sessionCookie, sessionExpiry: sessionExpiry}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		h.renderLogin(w, r)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, loginResponse{Error: "invalid request"})
			return
		}
		csrfToken := r.Header.Get("X-CSRF-Token")
		if csrfToken == "" || !h.authSvc.ValidateCSRFToken(csrfToken, "", 15*time.Minute) {
			slog.Warn("csrf validation failed on json login", "username", req.Username)
			writeJSON(w, http.StatusUnauthorized, loginResponse{Error: "invalid request"})
			return
		}
	} else {
		req.Username = r.FormValue("username")
		req.Password = r.FormValue("password")

		csrfToken := r.FormValue("csrf_token")
		if csrfToken == "" || !h.authSvc.ValidateCSRFToken(csrfToken, "", 15*time.Minute) {
			slog.Warn("csrf validation failed", "username", req.Username)
			if r.Header.Get("HX-Request") != "" {
				w.Header().Set("HX-Retarget", "#login-error")
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid request"})
				return
			}
			writeJSON(w, http.StatusUnauthorized, loginResponse{Error: "invalid request"})
			return
		}
	}

	user, err := h.authSvc.Authenticate(req.Username, req.Password)
	if err != nil {
		metrics.AuthAttempts.WithLabelValues("failure").Inc()
		slog.Warn("login failed", "username", req.Username, "error", err)
		if r.Header.Get("HX-Request") != "" {
			w.Header().Set("HX-Retarget", "#login-error")
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusUnauthorized, loginResponse{Error: err.Error()})
		return
	}

	metrics.AuthAttempts.WithLabelValues("success").Inc()

	ip := h.getIP(r)
	h.db.UpdateUserLogin(user.ID, ip)

	sessionToken, err := auth.GenerateSessionToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, loginResponse{Error: "internal error"})
		return
	}

	sessionID, err := auth.GenerateSessionToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, loginResponse{Error: "internal error"})
		return
	}

	session := &models.Session{
		ID:        sessionID,
		UserID:    user.ID,
		Token:     sessionToken,
		ExpiresAt: time.Now().Add(h.sessionExpiry),
		IP:        ip,
		UserAgent: r.UserAgent(),
	}

	if err := h.db.CreateSession(session); err != nil {
		writeJSON(w, http.StatusInternalServerError, loginResponse{Error: "internal error"})
		return
	}

	jwt, err := h.authSvc.GenerateJWT(user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, loginResponse{Error: "internal error"})
		return
	}

	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	if !secure {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(host)
		local := ip != nil && (ip.IsLoopback() || ip.IsPrivate())
		if local {
			secure = false
		}
	}
	slog.Info("login success", "username", user.Username, "secure_cookie", secure, "remote_addr", r.RemoteAddr, "hx_request", r.Header.Get("HX-Request"))
	http.SetCookie(w, &http.Cookie{
		Name:     h.sessionCookie,
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.sessionExpiry.Seconds()),
	})

	if r.Header.Get("HX-Request") != "" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Header.Get("Content-Type") == "application/json" {
		writeJSON(w, http.StatusOK, loginResponse{Token: jwt, SessionID: sessionToken})
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.sessionCookie)
	if err == nil {
		h.db.DeleteSession(cookie.Value)
	}

	logoutSecure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	if !logoutSecure {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsPrivate()) {
			logoutSecure = false
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   logoutSecure,
		MaxAge:   -1,
	})

	if r.Header.Get("HX-Request") != "" {
		w.Header().Set("HX-Redirect", "/login")
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) renderLogin(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	csrfToken, err := h.authSvc.GenerateCSRFToken("")
	if err != nil {
		slog.Error("failed to generate csrf token", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	loginTmpl.Execute(w, map[string]string{"CSRFToken": csrfToken})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})
}

func (h *AuthHandler) CSRFToken(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	token, err := h.authSvc.GenerateCSRFToken("")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"csrf_token": token})
}

func (h *AuthHandler) getIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	return r.RemoteAddr
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
