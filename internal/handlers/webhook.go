package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/models"
)

type WebhookHandler struct {
	db *database.DB
}

type DeployEvent struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	SHA     string `json:"sha"`
}

func NewWebhookHandler(db *database.DB) *WebhookHandler {
	return &WebhookHandler{db: db}
}

func (h *WebhookHandler) Deploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var ev DeployEvent
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if ev.Service == "" || ev.Status == "" {
		writeErrorJSON(w, http.StatusBadRequest, "service and status required")
		return
	}

	a := &models.AuditLog{
		UserID:    0,
		Action:    "deploy",
		Resource:  ev.Service,
		Detail:    "status=" + ev.Status + " sha=" + ev.SHA,
		IP:        r.RemoteAddr,
		UserAgent: r.UserAgent(),
	}

	if err := h.db.LogAudit(a); err != nil {
		slog.Warn("failed to log deploy event", "error", err)
	}

	slog.Info("deploy event", "service", ev.Service, "status", ev.Status, "sha", ev.SHA)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

type DeployRecord struct {
	Service   string `json:"service"`
	Status    string `json:"status"`
	SHA       string `json:"sha"`
	Timestamp string `json:"timestamp"`
}

func (h *WebhookHandler) RecentDeploys(w http.ResponseWriter, r *http.Request) {
	logs, err := h.db.GetAuditLogs(50)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "internal error")
		return
	}

	var deploys []DeployRecord
	for _, l := range logs {
		if l.Action == "deploy" {
			deploys = append(deploys, DeployRecord{
				Service:   l.Resource,
				Status:    "",
				SHA:       "",
				Timestamp: l.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	if deploys == nil {
		deploys = []DeployRecord{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deploys)
}
