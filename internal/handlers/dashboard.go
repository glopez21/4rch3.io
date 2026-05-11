package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/middleware"
	"github.com/4rch3/portal/internal/models"
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
	w.Write([]byte(h.renderDashboard(user != nil, isLAN, services)))
}

func (h *DashboardHandler) ServiceStatus(w http.ResponseWriter, r *http.Request) {
	services, err := h.db.ListServices()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var html strings.Builder
	for _, svc := range services {
		html.WriteString(serviceCard(svc))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html.String()))
}

func (h *DashboardHandler) renderDashboard(authenticated bool, isLAN bool, services []*models.Service) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="en" class="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>4rch3.io</title>
    <script src="https://unpkg.com/htmx.org@2.0.4"></script>
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        tailwind.config = {
            darkMode: 'class',
            theme: {
                extend: {
                    colors: {
                        neon: { cyan: '#00f0ff', purple: '#a855f7', pink: '#ec4899' }
                    }
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
        .status-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
        .status-dot.online { background: #22c55e; box-shadow: 0 0 8px rgba(34,197,94,0.6); }
        .status-dot.degraded { background: #eab308; box-shadow: 0 0 8px rgba(234,179,8,0.6); }
        .status-dot.offline { background: #ef4444; box-shadow: 0 0 8px rgba(239,68,68,0.6); }
        .status-dot.unknown { background: #6b7280; }
        @keyframes pulse-dot { 0%,100% { opacity: 1; } 50% { opacity: 0.4; } }
        .status-dot.loading { animation: pulse-dot 1s ease-in-out infinite; }
        .card-hover { transition: all 0.2s ease; }
        .card-hover:hover { transform: translateY(-2px); border-color: rgba(0,240,255,0.3); }
    </style>
</head>
<body class="grid-bg min-h-screen">
    <nav class="border-b border-gray-800 bg-gray-900/50 backdrop-blur-xl sticky top-0 z-50">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div class="flex items-center justify-between h-16">
                <div class="flex items-center gap-3">
                    <span class="text-2xl font-bold bg-gradient-to-r from-neon-cyan via-neon-purple to-neon-pink bg-clip-text text-transparent">4rch3</span>
                    <span class="text-gray-600 text-sm">/</span>
                    <span class="text-gray-400 text-sm">portal</span>
                </div>
                <div class="flex items-center gap-4">`)

	if authenticated {
		sb.WriteString(`
                    <a href="/admin" class="text-gray-500 hover:text-white text-sm transition-colors">admin</a>
                    <a href="/logout" class="text-gray-500 hover:text-red-400 text-sm transition-colors">logout</a>`)
	} else {
		sb.WriteString(`
                    <a href="/login" class="text-neon-cyan hover:text-white text-sm transition-colors">login</a>`)
	}

	sb.WriteString(`
                </div>
            </div>
        </div>
    </nav>

    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div class="flex items-center justify-between mb-8">
            <div>
                <h2 class="text-2xl font-bold text-white">Services</h2>
                <p class="text-gray-500 text-sm mt-1">homelab operations hub</p>
            </div>
            <button hx-get="/api/services/status" hx-target="#services-grid" hx-swap="innerHTML"
                    class="px-4 py-2 text-sm bg-gray-800 hover:bg-gray-700 text-gray-300 rounded-xl transition-colors">
                refresh
            </button>
        </div>

        <div id="services-grid" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">`)

	if len(services) == 0 && !authenticated {
		sb.WriteString(`
            <div class="col-span-full text-center py-12">
                <p class="text-gray-500">no services discovered yet</p>
                <p class="text-gray-600 text-sm mt-2">login to access the portal</p>
            </div>`)
	} else {
		for _, svc := range services {
			sb.WriteString(serviceCard(svc))
		}
	}

	sb.WriteString(`
        </div>

        <div class="mt-12 border-t border-gray-800 pt-8">
            <h3 class="text-lg font-semibold text-white mb-4">Quick Links</h3>
            <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
                <a href="/admin" class="px-4 py-3 bg-gray-900/50 border border-gray-800 rounded-xl text-gray-400 hover:text-white hover:border-gray-700 transition-all text-sm text-center">
                    admin panel
                </a>
                <a href="#" class="px-4 py-3 bg-gray-900/50 border border-gray-800 rounded-xl text-gray-400 hover:text-white hover:border-gray-700 transition-all text-sm text-center">
                    documentation
                </a>
                <a href="/status" class="px-4 py-3 bg-gray-900/50 border border-gray-800 rounded-xl text-gray-400 hover:text-white hover:border-gray-700 transition-all text-sm text-center">
                    system status
                </a>
            </div>
        </div>
    </main>

    <script>
        htmx.onLoad(function() {
            htmx.ajax('GET', '/api/services/status', { target: '#services-grid', swap: 'innerHTML' });
        });
    </script>
</body>
</html>`)

	return sb.String()
}

func serviceCard(svc *models.Service) string {
	statusColor := map[string]string{
		"online":   "text-green-400",
		"degraded": "text-yellow-400",
		"offline":  "text-red-400",
		"unknown":  "text-gray-500",
	}[svc.Status]

	dotColor := map[string]string{
		"online": "online", "degraded": "degraded",
		"offline": "offline", "unknown": "unknown",
	}[svc.Status]

	latencyStr := ""
	if svc.Latency > 0 {
		if svc.Latency < 1000 {
			latencyStr = strconv.FormatInt(svc.Latency, 10) + "ms"
		} else {
			latencyStr = strconv.FormatInt(svc.Latency/1000, 10) + "s"
		}
	}

	icon := svc.Icon
	if icon == "" {
		icon = "⊞"
	}

	return `<div class="card-hover bg-gray-900/50 border border-gray-800 rounded-2xl p-5">
        <div class="flex items-start justify-between mb-3">
            <div class="flex items-center gap-3">
                <span class="text-xl">` + icon + `</span>
                <div>
                    <h3 class="text-gray-200 font-semibold text-sm">` + svc.DisplayName + `</h3>
                    <span class="text-gray-600 text-xs">` + svc.Name + `</span>
                </div>
            </div>
            <span class="status-dot ` + dotColor + `"></span>
        </div>
        <p class="text-gray-500 text-xs mb-3 line-clamp-1">` + svc.Description + `</p>
        <div class="flex items-center justify-between mt-3 pt-3 border-t border-gray-800">
            <span class="` + statusColor + ` text-xs font-medium">` + svc.Status + `</span>
            <span class="text-gray-600 text-xs">` + latencyStr + `</span>
        </div>
    </div>`
}


