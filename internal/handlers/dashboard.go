package handlers

import (
	"net/http"

	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/middleware"
)

type DashboardHandler struct {
	db            *database.DB
	grafanaDomain string
}

func NewDashboardHandler(db *database.DB, grafanaDomain string) *DashboardHandler {
	return &DashboardHandler{db: db, grafanaDomain: grafanaDomain}
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

func (h *DashboardHandler) DeployPanel(w http.ResponseWriter, r *http.Request) {
	deploys, err := h.db.GetDeployEvents(10)
	if err != nil {
		w.Write([]byte(`<p class="text-gray-500 text-sm">unable to load deploys</p>`))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	deployPanelTmpl.Execute(w, toDeployRecords(deploys))
}
