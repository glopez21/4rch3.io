package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/deploy"
	"github.com/4rch3/portal/internal/middleware"
)

type DashboardHandler struct {
	db            *database.DB
	grafanaDomain string
	dockerSocket  string
	deployRunner  *deploy.Runner
}

func NewDashboardHandler(db *database.DB, grafanaDomain string, dockerSocket string) *DashboardHandler {
	return &DashboardHandler{db: db, grafanaDomain: grafanaDomain, dockerSocket: dockerSocket}
}

// SetDeployRunner wires the deploy whitelist used by the manage page to
// decide whether the Deploy action is offered for a service.
func (h *DashboardHandler) SetDeployRunner(r *deploy.Runner) {
	h.deployRunner = r
}

func (h *DashboardHandler) Index(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	isLAN := middleware.IsLAN(r)

	services, err := h.db.ListServices()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "internal error")
		return
	}

	deploys, err := h.db.GetDeployEvents(10)
	if err != nil {
		deploys = nil
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := DashboardData{
		Authenticated: user != nil,
		IsLAN:         isLAN,
		Services:      toServiceCards(services),
		Deploys:       toDeployRecords(deploys),
		GrafanaDomain: h.grafanaDomain,
	}
	dashboardTmpl.Execute(w, data)
}

func (h *DashboardHandler) ServiceStatus(w http.ResponseWriter, r *http.Request) {
	services, err := h.db.ListServices()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	serviceCardTmpl.Execute(w, toServiceCards(services))
}

type ServicePanelData struct {
	ID          int64
	DisplayName string
	Description string
	URL         string
	PublicURL   string
}

// ServicePanel renders the service modal fragment (htmx target).
func (h *DashboardHandler) ServicePanel(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id < 1 {
		writeErrorJSON(w, http.StatusBadRequest, "invalid id")
		return
	}

	svc, err := h.db.GetService(id)
	if err != nil || svc == nil {
		writeErrorJSON(w, http.StatusNotFound, "service not found")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := servicePanelTmpl.Execute(w, ServicePanelData{
		ID:          svc.ID,
		DisplayName: svc.DisplayName,
	Description: svc.Description,
		URL:         svc.URL,
		PublicURL:   svc.PublicURL,
	}); err != nil {
		slog.Error("service panel render failed", "error", err)
	}
}

type ServiceManageData struct {
	ID            int64
	Name          string
	DisplayName   string
	Description   string
	URL           string
	InternalURL   string
	HealthPath    string
	Status        string
	DeployAllowed bool
}

// ServiceManage renders the per-service management page at /services/{id}.
func (h *DashboardHandler) ServiceManage(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/services/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	svc, err := h.db.GetService(id)
	if err != nil || svc == nil {
		http.NotFound(w, r)
		return
	}

	deployAllowed := false
	if h.deployRunner != nil {
		deployAllowed = h.deployRunner.Allowed(svc.Name)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := serviceManageTmpl.Execute(w, ServiceManageData{
		ID:            svc.ID,
		Name:          svc.Name,
		DisplayName:   svc.DisplayName,
		Description:   svc.Description,
		URL:           svc.URL,
		InternalURL:   svc.InternalURL,
		HealthPath:    svc.HealthPath,
		Status:        svc.Status,
		DeployAllowed: deployAllowed,
	}); err != nil {
		slog.Error("service manage render failed", "error", err)
	}
}

func (h *DashboardHandler) DeployPanel(w http.ResponseWriter, r *http.Request) {
	deploys, err := h.db.GetDeployEvents(10)
	if err != nil {
		w.Write([]byte(`<p class="text-gray-500 text-sm">unable to load deploys</p>`))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	deployPanelTmpl.Execute(w, toDeployRecords(deploys))
}
