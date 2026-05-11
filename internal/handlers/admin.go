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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderAdminPanel(user, users, services, logs)))
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

func (h *AdminHandler) renderAdminPanel(admin *models.User, users []*models.User, services []*models.Service, logs []*models.AuditLog) string {
	// Placeholder for admin panel template
	return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Admin - 4rch3.io</title>
    <script src="https://unpkg.com/htmx.org@2.0.4"></script>
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        tailwind.config = {
            darkMode: 'class',
            theme: {
                extend: {
                    colors: { neon: { cyan: '#00f0ff', purple: '#a855f7', pink: '#ec4899' } }
                }
            }
        }
    </script>
    <style>
        body { background: #0a0a0f; font-family: ui-monospace, monospace; }
        .grid-bg {
            background-image:
                linear-gradient(rgba(0, 240, 255, 0.03) 1px, transparent 1px),
                linear-gradient(90deg, rgba(0, 240, 255, 0.03) 1px, transparent 1px);
            background-size: 40px 40px;
        }
    </style>
</head>
<body class="grid-bg min-h-screen">
    <nav class="border-b border-gray-800 bg-gray-900/50 backdrop-blur-xl sticky top-0 z-50">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div class="flex items-center justify-between h-16">
                <div class="flex items-center gap-3">
                    <span class="text-2xl font-bold bg-gradient-to-r from-neon-cyan via-neon-purple to-neon-pink bg-clip-text text-transparent">4rch3</span>
                    <span class="text-gray-600 text-sm">/</span>
                    <span class="text-neon-cyan text-sm font-semibold">admin</span>
                </div>
                <div class="flex items-center gap-4">
                    <a href="/" class="text-gray-500 hover:text-gray-300 text-sm transition-colors">dashboard</a>
                    <span class="text-gray-500 text-sm">` + admin.Username + `</span>
                    <a href="/logout" class="text-gray-500 hover:text-red-400 text-sm transition-colors">logout</a>
                </div>
            </div>
        </div>
    </nav>

    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <h2 class="text-2xl font-bold text-white mb-6">Admin Panel</h2>

        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
            <div class="bg-gray-900/50 border border-gray-800 rounded-2xl p-6">
                <h3 class="text-lg font-semibold text-white mb-4">Users (` + strconv.Itoa(len(users)) + `)</h3>
                <div class="space-y-2">
                    ` + h.renderUserList(users) + `
                </div>
            </div>

            <div class="bg-gray-900/50 border border-gray-800 rounded-2xl p-6">
                <h3 class="text-lg font-semibold text-white mb-4">Services (` + strconv.Itoa(len(services)) + `)</h3>
                <div class="space-y-2">
                    ` + h.renderServiceList(services) + `
                </div>
            </div>
        </div>

        <div class="bg-gray-900/50 border border-gray-800 rounded-2xl p-6">
            <h3 class="text-lg font-semibold text-white mb-4">Recent Audit Log</h3>
            <div class="overflow-x-auto">
                <table class="w-full text-sm">
                    <thead>
                        <tr class="text-gray-500 text-left">
                            <th class="pb-2 pr-4">Time</th>
                            <th class="pb-2 pr-4">User</th>
                            <th class="pb-2 pr-4">Action</th>
                            <th class="pb-2 pr-4">Resource</th>
                            <th class="pb-2">Detail</th>
                        </tr>
                    </thead>
                    <tbody>
                        ` + h.renderAuditLogs(logs) + `
                    </tbody>
                </table>
            </div>
        </div>
    </main>
</body>
</html>`
}

func (h *AdminHandler) renderUserList(users []*models.User) string {
	var html string
	for _, u := range users {
		roleColor := map[string]string{
			"admin": "text-neon-cyan", "user": "text-green-400",
			"readonly": "text-gray-400", "service": "text-yellow-400",
		}
		status := "text-green-400"
		if !u.Active {
			status = "text-red-400"
		}
		html += `<div class="flex items-center justify-between py-2 border-b border-gray-800 last:border-0">
			<div>
				<span class="text-gray-200">` + u.Username + `</span>
				<span class="` + roleColor[string(u.Role)] + ` text-xs ml-2">` + string(u.Role) + `</span>
			</div>
			<span class="` + status + ` text-xs">` + map[bool]string{true: "active", false: "disabled"}[u.Active] + `</span>
		</div>`
	}
	return html
}

func (h *AdminHandler) renderServiceList(services []*models.Service) string {
	var html string
	for _, s := range services {
		statusColor := map[string]string{
			"online": "text-green-400", "degraded": "text-yellow-400",
			"offline": "text-red-400", "unknown": "text-gray-500",
		}
		html += `<div class="flex items-center justify-between py-2 border-b border-gray-800 last:border-0">
			<div>
				<span class="text-gray-200">` + s.DisplayName + `</span>
				<span class="text-gray-500 text-xs ml-2">` + s.Name + `</span>
			</div>
			<span class="` + statusColor[s.Status] + ` text-xs">` + s.Status + `</span>
		</div>`
	}
	return html
}

func (h *AdminHandler) renderAuditLogs(logs []*models.AuditLog) string {
	var html string
	for _, l := range logs {
		html += `<tr class="border-b border-gray-800 text-gray-400">
			<td class="py-2 pr-4 text-xs">` + l.CreatedAt.Format("2006-01-02 15:04:05") + `</td>
			<td class="py-2 pr-4">` + strconv.FormatInt(l.UserID, 10) + `</td>
			<td class="py-2 pr-4">` + l.Action + `</td>
			<td class="py-2 pr-4">` + l.Resource + `</td>
			<td class="py-2 text-xs text-gray-600">` + l.Detail + `</td>
		</tr>`
	}
	return html
}
