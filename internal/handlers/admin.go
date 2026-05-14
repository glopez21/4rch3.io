package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

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
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	users, err := h.db.ListUsers()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	services, err := h.db.ListServices()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	logs, err := h.db.GetAuditLogs(50)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
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
	}
	adminTmpl.Execute(w, data)
}

func (h *AdminHandler) Users(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil || user.Role != models.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	switch r.Method {
	case "GET":
		users, err := h.db.ListUsers()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, users)

	case "POST":
		var req struct {
			Username string      `json:"username"`
			Email    string      `json:"email"`
			Password string      `json:"password"`
			Role     models.Role `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		_, err := h.authSvc.CreateUser(req.Username, req.Email, req.Password, req.Role)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})

	case "DELETE":
		idStr := r.URL.Query().Get("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}
		if err := h.db.DeleteUser(id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}