package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/4rch3/portal/internal/auth"
	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/middleware"
	"github.com/4rch3/portal/internal/models"
)

type AdminHandler struct {
	db      *database.DB
	authSvc *auth.Service
}

func NewAdminHandler(db *database.DB, authSvc *auth.Service) *AdminHandler {
	return &AdminHandler{db: db, authSvc: authSvc}
}

func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil || user.Role != models.RoleAdmin {
		writeErrorJSON(w, http.StatusForbidden, "forbidden")
		return
	}

	csrfToken, err := h.authSvc.GenerateCSRFToken("")
	if err != nil {
		slog.Error("failed to generate admin csrf token", "error", err)
		writeErrorJSON(w, http.StatusInternalServerError, "internal error")
		return
	}

	users, err := h.db.ListUsers()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "internal error")
		return
	}

	services, err := h.db.ListServices()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "internal error")
		return
	}

	logs, err := h.db.GetAuditLogs(50)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "internal error")
		return
	}

	userData := make([]UserData, 0, len(users))
	for _, u := range users {
		userData = append(userData, UserData{
			Username:    u.Username,
			Role:        string(u.Role),
			RoleColor:   roleColor(string(u.Role)),
			StatusText:  userStatusText(u.Active),
			StatusColor: userStatusColor(u.Active),
		})
	}

	serviceData := make([]ServiceData, 0, len(services))
	for _, s := range services {
		serviceData = append(serviceData, ServiceData{
			DisplayName: s.DisplayName,
			Name:        s.Name,
			Status:      s.Status,
			StatusColor: statusColor(s.Status),
		})
	}

	logData := make([]AuditLogData, 0, len(logs))
	for _, l := range logs {
		logData = append(logData, AuditLogData{
			CreatedAt: l.CreatedAt.Format("2006-01-02 15:04:05"),
			UserID:    strconv.FormatInt(l.UserID, 10),
			Action:    l.Action,
			Resource:  l.Resource,
			Detail:    l.Detail,
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := AdminData{
		AdminUsername: user.Username,
		UserCount:     strconv.Itoa(len(users)),
		ServiceCount:  strconv.Itoa(len(services)),
		Users:         userData,
		Services:      serviceData,
		Logs:          logData,
		CSRFToken:     csrfToken,
	}
	adminTmpl.Execute(w, data)
}

func (h *AdminHandler) Users(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil || user.Role != models.RoleAdmin {
		writeErrorJSON(w, http.StatusForbidden, "forbidden")
		return
	}

	switch r.Method {
	case "GET":
		users, err := h.db.ListUsers()
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, users)

	case "POST":
		var req struct {
			Username  string      `json:"username"`
			Email     string      `json:"email"`
			Password  string      `json:"password"`
			Role      models.Role `json:"role"`
			CSRFToken string      `json:"csrf_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if req.CSRFToken == "" || !h.authSvc.ValidateCSRFToken(req.CSRFToken, "", 15*time.Minute) {
			slog.Warn("admin csrf validation failed", "action", "create_user", "username", req.Username)
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid request"})
			return
		}
		_, err := h.authSvc.CreateUser(req.Username, req.Email, req.Password, req.Role)
		if err != nil {
			slog.Error("failed to create user", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})

	case "DELETE":
		csrfToken := r.Header.Get("X-CSRF-Token")
		if csrfToken == "" || !h.authSvc.ValidateCSRFToken(csrfToken, "", 15*time.Minute) {
			slog.Warn("admin csrf validation failed", "action", "delete_user")
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid request"})
			return
		}
		idStr := r.URL.Query().Get("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}
		if err := h.db.DeleteUser(id); err != nil {
			slog.Error("failed to delete user", "id", id, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete user"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
