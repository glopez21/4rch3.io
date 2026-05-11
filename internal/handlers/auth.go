package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/4rch3/portal/internal/auth"
	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/middleware"
	"github.com/4rch3/portal/internal/models"
)

type AuthHandler struct {
	db        *database.DB
	authSvc   *auth.Service
	cfg       auth.AuthConfig
}

func NewAuthHandler(db *database.DB, authSvc *auth.Service, cfg auth.AuthConfig) *AuthHandler {
	return &AuthHandler{db: db, authSvc: authSvc, cfg: cfg}
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
	} else {
		req.Username = r.FormValue("username")
		req.Password = r.FormValue("password")
	}

	user, err := h.authSvc.Authenticate(req.Username, req.Password)
	if err != nil {
		if r.Header.Get("HX-Request") != "" {
			w.Header().Set("HX-Retarget", "#login-error")
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusUnauthorized, loginResponse{Error: err.Error()})
		return
	}

	ip := h.getIP(r)
	h.db.UpdateUserLogin(user.ID, ip)

	sessionToken, err := auth.GenerateSessionToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, loginResponse{Error: "internal error"})
		return
	}

	session := &models.Session{
		ID:        sessionToken,
		UserID:    user.ID,
		Token:     sessionToken,
		ExpiresAt: time.Now().Add(h.cfg.SessionExpiry),
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

	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.cfg.SessionExpiry.Seconds()),
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
	cookie, err := r.Cookie(h.cfg.SessionCookieName)
	if err == nil {
		h.db.DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(loginPage))
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
