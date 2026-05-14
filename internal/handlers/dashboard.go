package handlers

import (
	"net/http"

	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/middleware"
)

type DashboardHandler struct {
	db *database.DB
}

func NewDashboardHandler(db *database.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

func (h *DashboardHandler) Index(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	isLAN := middleware.IsLAN(r)

	services, err := h.db.ListServices()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := DashboardData{
		Authenticated: user != nil,
		IsLAN:         isLAN,
		Services:      toServiceCards(services),
	}
	dashboardTmpl.Execute(w, data)
}

func (h *DashboardHandler) ServiceStatus(w http.ResponseWriter, r *http.Request) {
	services, err := h.db.ListServices()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	serviceCardTmpl.Execute(w, toServiceCards(services))
}