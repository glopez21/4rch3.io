package handlers

import (
	"crypto/hmac"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/deploy"
	"github.com/4rch3/portal/internal/middleware"
	"github.com/4rch3/portal/internal/models"
)

type WebhookHandler struct {
	db     *database.DB
	runner *deploy.Runner
	secret string
}

type DeployEvent struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	SHA     string `json:"sha"`
}

func NewWebhookHandler(db *database.DB, runner *deploy.Runner, secret string) *WebhookHandler {
	return &WebhookHandler{db: db, runner: runner, secret: secret}
}

// Deploy accepts deploy notifications and, when the target service is in
// the configured whitelist, triggers a redeploy (docker compose pull+up).
//
// Authorization: a valid browser session OR a matching X-Webhook-Secret
// header (used by CI pipelines when PORTAL_DEPLOY_SECRET is configured).
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

	if ev.Service == "" {
		writeErrorJSON(w, http.StatusBadRequest, "service required")
		return
	}

	authorized := middleware.GetUser(r) != nil
	if !authorized && h.secret != "" {
		provided := r.Header.Get("X-Webhook-Secret")
		authorized = provided != "" && hmac.Equal([]byte(provided), []byte(h.secret))
	}
	if !authorized {
		writeErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	status := ev.Status
	detail := "status=" + status + " sha=" + ev.SHA

	if h.runner != nil && h.runner.Allowed(ev.Service) {
		ctx := r.Context()
		out, err := h.runner.Deploy(ctx, ev.Service)
		if err != nil {
			slog.Error("deploy failed", "service", ev.Service, "error", err)
			status = "failure"
			detail = "status=failure error=" + err.Error()
		} else {
			status = "success"
			detail = "status=success"
			slog.Info("deploy completed", "service", ev.Service, "output", out)
		}
	} else if status == "" {
		// Not whitelisted and no explicit status: record a plain event.
		status = "notified"
	}

	a := &models.AuditLog{
		UserID:    0,
		Action:    "deploy",
		Resource:  ev.Service,
		Detail:    detail,
		IP:        r.RemoteAddr,
		UserAgent: r.UserAgent(),
	}
	if err := h.db.LogAudit(a); err != nil {
		slog.Warn("failed to log deploy event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": status})
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
