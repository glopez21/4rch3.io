package handlers

import (
	"html/template"
	"strconv"

	"github.com/4rch3/portal/internal/models"
)

type DashboardData struct {
	Authenticated bool
	IsLAN         bool
	Services      []ServiceCardData
	Deploys       []DeployRecordData
	GrafanaDomain string
}

type DeployRecordData struct {
	Service     string
	Status      string
	SHA         string
	Timestamp   string
	StatusBadge string
}

type ServiceCardData struct {
	ID          int64
	Icon        string
	DisplayName string
	Name        string
	Description string
	Status      string
	LatencyStr  string
	StatusColor string
	DotColor    string
}

type AdminData struct {
	AdminUsername string
	UserCount     string
	ServiceCount  string
	Users         []UserData
	Services      []ServiceData
	Logs          []AuditLogData
	CSRFToken     string
}

type UserData struct {
	Username    string
	Role        string
	RoleColor   string
	Active      bool
	StatusText  string
	StatusColor string
}

type ServiceData struct {
	DisplayName string
	Name        string
	Status      string
	StatusColor string
}

type AuditLogData struct {
	CreatedAt string
	UserID    string
	Action    string
	Resource  string
	Detail    string
}

func toServiceCards(services []*models.Service) []ServiceCardData {
	cards := make([]ServiceCardData, 0, len(services))
	for _, svc := range services {
		card := ServiceCardData{
			ID:          svc.ID,
			Icon:        svc.Icon,
			DisplayName: svc.DisplayName,
			Name:        svc.Name,
			Description: svc.Description,
			Status:      svc.Status,
			StatusColor: statusColor(svc.Status),
			DotColor:    svc.Status,
		}
		if svc.Latency > 0 {
			if svc.Latency < 1000 {
				card.LatencyStr = strconv.FormatInt(svc.Latency, 10) + "ms"
			} else {
				card.LatencyStr = strconv.FormatInt(svc.Latency/1000, 10) + "s"
			}
		}
		if card.Icon == "" {
			card.Icon = "\u229e"
		}
		cards = append(cards, card)
	}
	return cards
}

func toDeployRecords(logs []*models.AuditLog) []DeployRecordData {
	records := make([]DeployRecordData, 0, len(logs))
	for _, l := range logs {
		status := "unknown"
		badge := "bg-gray-700 text-gray-300"
		if len(l.Detail) >= 9 && l.Detail[:7] == "status=" {
			end := 7
			for end < len(l.Detail) && l.Detail[end] != ' ' {
				end++
			}
			status = l.Detail[7:end]
			switch status {
			case "success":
				badge = "bg-green-900/50 text-green-400"
			case "failure":
				badge = "bg-red-900/50 text-red-400"
			case "cancelled":
				badge = "bg-yellow-900/50 text-yellow-400"
			}
		}
		records = append(records, DeployRecordData{
			Service:     l.Resource,
			Status:      status,
			Timestamp:   l.CreatedAt.Format("15:04:05"),
			StatusBadge: badge,
		})
	}
	return records
}

func statusColor(status string) string {
	switch status {
	case "online":
		return "text-green-400"
	case "degraded":
		return "text-yellow-400"
	case "offline":
		return "text-red-400"
	default:
		return "text-gray-500"
	}
}

func roleColor(role string) string {
	switch role {
	case "admin":
		return "text-neon-cyan"
	case "user":
		return "text-green-400"
	case "readonly":
		return "text-gray-400"
	case "service":
		return "text-yellow-400"
	default:
		return "text-gray-500"
	}
}

func userStatusColor(active bool) string {
	if active {
		return "text-green-400"
	}
	return "text-red-400"
}

func userStatusText(active bool) string {
	if active {
		return "active"
	}
	return "disabled"
}

const baseStyle = `
    <style>
        body { background: #0a0a0f; font-family: ui-monospace, monospace; }
        .grid-bg {
            background-image:
                linear-gradient(rgba(0, 240, 255, 0.03) 1px, transparent 1px),
                linear-gradient(90deg, rgba(0, 240, 255, 0.03) 1px, transparent 1px);
            background-size: 40px 40px;
        }
    </style>`

const tailwindConfig = `
    <script>
        tailwind.config = {
            darkMode: 'class',
            theme: {
                extend: {
                    colors: {
                        neon: {
                            cyan: '#00f0ff',
                            purple: '#a855f7',
                            pink: '#ec4899',
                        }
                    }
                }
            }
        }
    </script>`

const navBar = `
    <nav class="border-b border-gray-800 bg-gray-900/50 backdrop-blur-xl sticky top-0 z-50">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div class="flex items-center justify-between h-16">
                <div class="flex items-center gap-3">
                    <span class="text-2xl font-bold bg-gradient-to-r from-neon-cyan via-neon-purple to-neon-pink bg-clip-text text-transparent">4rch3</span>`

var loginTmpl = template.Must(template.New("login").Parse(`<!DOCTYPE html>
<html lang="en" class="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login - 4rch3.io</title>
    <link rel="icon" type="image/svg+xml" href="/favicon.svg">
    <script src="https://unpkg.com/htmx.org@2.0.4" integrity="sha384-7uoN2oWEkBN1YI9ZOKj1jF3OGfC2jqF6elWMrbFU+dxl3NTah3MsiGZ+36I96u7Y" crossorigin="anonymous"></script>
    <script src="https://cdn.tailwindcss.com"></script>
    ` + tailwindConfig + `
    <style>
        @keyframes glow {
            0%, 100% { box-shadow: 0 0 20px rgba(0, 240, 255, 0.3); }
            50% { box-shadow: 0 0 40px rgba(0, 240, 255, 0.6); }
        }
        .glow-border { animation: glow 3s ease-in-out infinite; }
        body { background: #0a0a0f; }
        .grid-bg {
            background-image:
                linear-gradient(rgba(0, 240, 255, 0.03) 1px, transparent 1px),
                linear-gradient(90deg, rgba(0, 240, 255, 0.03) 1px, transparent 1px);
            background-size: 40px 40px;
        }
    </style>
</head>
<body class="grid-bg min-h-screen flex items-center justify-center p-4">
    <div class="w-full max-w-md">
        <div class="text-center mb-8">
            <h1 class="text-4xl font-bold bg-gradient-to-r from-neon-cyan via-neon-purple to-neon-pink bg-clip-text text-transparent">4rch3.io</h1>
            <p class="text-gray-500 mt-2 text-sm">homelab operations portal</p>
        </div>

        <div class="bg-gray-900/80 backdrop-blur-xl rounded-2xl p-8 border border-gray-800 glow-border">
            <!-- Add native form fallback so login works even if htmx fails to load -->
            <form action="/login" method="post" hx-post="/login" hx-target="this" hx-swap="outerHTML" class="space-y-6">
                <div id="login-error" class="text-red-400 text-sm text-center"></div>

                <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">

                <div>
                    <label class="block text-sm font-medium text-gray-400 mb-2">Username</label>
                    <input type="text" name="username" required
                           class="w-full px-4 py-3 bg-gray-800/50 border border-gray-700 rounded-xl text-gray-100 placeholder-gray-600 focus:border-neon-cyan focus:ring-1 focus:ring-neon-cyan outline-none transition-all"
                           placeholder="enter username">
                </div>

                <div>
                    <label class="block text-sm font-medium text-gray-400 mb-2">Password</label>
                    <input type="password" name="password" required
                           class="w-full px-4 py-3 bg-gray-800/50 border border-gray-700 rounded-xl text-gray-100 placeholder-gray-600 focus:border-neon-cyan focus:ring-1 focus:ring-neon-cyan outline-none transition-all"
                           placeholder="enter password">
                </div>

                <button type="submit"
                        class="w-full py-3 px-4 bg-gradient-to-r from-neon-cyan to-neon-purple text-black font-semibold rounded-xl hover:opacity-90 transition-all duration-200">
                    Sign In
                </button>
            </form>
        </div>

        <p class="text-center text-gray-600 text-xs mt-6">
            authorized personnel only &middot; all activity is logged
        </p>
    </div>
</body>
</html>`))

var dashboardTmpl = template.Must(template.New("dashboard").Parse(`<!DOCTYPE html>
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
                        neon: {
                            cyan: '#00f0ff',
                            purple: '#a855f7',
                            pink: '#ec4899',
                        }
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
    </style>
    <style>
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
                <div class="flex items-center gap-3">
                {{if .Authenticated}}
                    <a href="/admin" class="text-gray-500 hover:text-white transition-colors" title="admin">
                        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
                    </a>
                    <a href="/logout" class="text-gray-500 hover:text-red-400 transition-colors" title="logout">
                        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
                    </a>
                {{else}}
                    <a href="/login" class="text-neon-cyan hover:text-white transition-colors" title="login">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/><polyline points="10 17 15 12 10 7"/><line x1="15" y1="12" x2="3" y2="12"/></svg>
                    </a>
                {{end}}
                </div>
            </div>
        </div>
    </nav>

    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">

        <div class="text-center pt-12 pb-10 md:pt-20 md:pb-14">
            <div class="inline-flex flex-col items-center">
                <div class="relative mb-5">
                    <div class="absolute inset-0 bg-gradient-to-r from-neon-cyan/20 via-neon-purple/20 to-neon-pink/20 blur-[80px] rounded-full scale-150"></div>
                    <div class="relative flex items-center gap-3 md:gap-4 text-5xl md:text-7xl lg:text-8xl font-mono font-bold tracking-tight">
                        <span class="text-gray-700 text-4xl md:text-5xl lg:text-6xl">&lt;</span>
                        <span class="bg-gradient-to-r from-neon-cyan via-neon-purple to-neon-pink bg-clip-text text-transparent">4rch3</span>
                        <span class="text-gray-700 text-4xl md:text-5xl lg:text-6xl">/&gt;</span>
                    </div>
                </div>
                <p class="text-gray-500 text-base md:text-lg">homelab operations hub</p>
            </div>
        </div>

        <div class="flex items-center justify-between mb-8">
            <h2 class="text-2xl font-bold text-white">Services</h2>
            <button hx-get="/api/services/status" hx-target="#services-grid" hx-swap="innerHTML"
                    class="px-4 py-2 text-sm bg-gray-800 hover:bg-gray-700 text-gray-300 rounded-xl transition-colors">
                refresh
            </button>
        </div>

        <div id="services-grid" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {{if and (not .Authenticated) (eq (len .Services) 0)}}
            <div class="col-span-full text-center py-12">
                <p class="text-gray-500">no services discovered yet</p>
                <p class="text-gray-600 text-sm mt-2">login to access the portal</p>
            </div>
        {{else}}
            {{range .Services}}
            <div class="card-hover bg-gray-900/50 border border-gray-800 rounded-2xl p-5 cursor-pointer"
                 hx-get="/api/service-panel?id={{.ID}}"
                 hx-trigger="click"
                 hx-target="#service-modal-content"
                 hx-swap="innerHTML"
                 onclick="document.getElementById('service-modal').classList.remove('hidden')">
                <div class="flex items-start justify-between mb-3">
                    <div class="flex items-center gap-3">
                        <span class="text-xl">{{.Icon}}</span>
                        <div>
                            <h3 class="text-gray-200 font-semibold text-sm">{{.DisplayName}}</h3>
                            <span class="text-gray-600 text-xs">{{.Name}}</span>
                        </div>
                    </div>
                    <span class="status-dot {{.DotColor}}"></span>
                </div>
                <p class="text-gray-500 text-xs mb-3 line-clamp-1">{{.Description}}</p>
                <div class="flex items-center justify-between mt-3 pt-3 border-t border-gray-800">
                    <span class="{{.StatusColor}} text-xs font-medium">{{.Status}}</span>
                    <span class="text-gray-600 text-xs">{{.LatencyStr}}</span>
                </div>
            </div>
            {{end}}
        {{end}}
        </div>

        <div class="mt-12 grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div class="bg-gray-900/50 border border-gray-800 rounded-2xl p-6">
                <h3 class="text-lg font-semibold text-white mb-4">Recent Deployments</h3>
                <div id="deploy-panel" hx-get="/api/deploy-panel" hx-swap="innerHTML">
                    <div class="space-y-2">
                    {{range .Deploys}}
                        <div class="flex items-center justify-between py-2 border-b border-gray-800 last:border-0">
                            <div class="flex items-center gap-2">
                                <span class="text-gray-200 text-sm">{{.Service}}</span>
                                <span class="text-xs {{.StatusBadge}} px-2 py-0.5 rounded-full">{{.Status}}</span>
                            </div>
                            <span class="text-gray-500 text-xs font-mono">{{.Timestamp}}</span>
                        </div>
                    {{else}}
                        <p class="text-gray-500 text-sm">no deployments yet</p>
                    {{end}}
                    </div>
                </div>
            </div>

            <div class="bg-gray-900/50 border border-gray-800 rounded-2xl p-6">
                <h3 class="text-lg font-semibold text-white mb-4">Logs & Monitoring</h3>
                <div class="space-y-4">
                    <p class="text-gray-500 text-sm">Centralized logging via Loki. View, search, and filter logs from all services in Grafana.</p>
                    <div class="flex flex-wrap gap-3">
                        <a href="https://grafana.{{.GrafanaDomain}}/explore?orgId=1&left=%7B%22datasource%22:%22Loki%22,%22queries%22:%5B%7B%22refId%22:%22A%22,%22expr%22:%22%7Bcompose_project%3D~%60.%2B%60%7D%20%7C%3D%20%60%60%22%7D%5D,%22range%22:%7B%22from%22:%22now-1h%22,%22to%22:%22now%22%7D%7D"
                           target="_blank"
                           class="px-4 py-3 bg-gray-800/50 border border-gray-700 rounded-xl text-gray-300 hover:text-white hover:border-gray-600 transition-all text-sm text-center inline-flex items-center gap-2">
                            <span>Explore Logs</span>
                            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
                        </a>
                        <a href="https://prometheus.{{.GrafanaDomain}}/targets"
                           target="_blank"
                           class="px-4 py-3 bg-gray-800/50 border border-gray-700 rounded-xl text-gray-300 hover:text-white hover:border-gray-600 transition-all text-sm text-center inline-flex items-center gap-2">
                            <span>Prometheus Targets</span>
                            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
                        </a>
                    </div>
                </div>
            </div>
        </div>

        <div class="mt-8 border-t border-gray-800 pt-8 text-center">
            <h3 class="text-lg font-semibold text-white mb-4">Quick Links</h3>
            <div class="inline-flex flex-wrap justify-center gap-3">
                <a href="/admin" class="px-4 py-3 bg-gray-900/50 border border-gray-800 rounded-xl text-gray-400 hover:text-white hover:border-gray-700 transition-all text-sm text-center">
                    admin panel
                </a>
                <a href="/status" class="px-4 py-3 bg-gray-900/50 border border-gray-800 rounded-xl text-gray-400 hover:text-white hover:border-gray-700 transition-all text-sm text-center">
                    system status
                </a>
                <a href="https://grafana.{{.GrafanaDomain}}"
                   target="_blank"
                   class="px-4 py-3 bg-gray-900/50 border border-gray-800 rounded-xl text-gray-400 hover:text-white hover:border-gray-700 transition-all text-sm text-center">
                    grafana
                </a>
            </div>
        </div>
    </main>

    <!-- Modal Container -->
    <div id="service-modal" class="fixed inset-0 bg-black/60 hidden flex items-center justify-center z-50">
        <div class="max-w-5xl w-full mx-4">
            <div id="service-modal-content" class="bg-gray-900 border border-gray-800 rounded-2xl p-6 shadow-2xl text-gray-400 text-sm">loading...</div>
            <div class="text-center mt-3">
                <button onclick="document.getElementById('service-modal').classList.add('hidden'); const es=window.__svcStats; if(es){es.close(); window.__svcStats=null;}" class="mt-2 px-4 py-2 bg-gray-800/70 border border-gray-700 rounded-xl text-gray-300 hover:text-white hover:border-gray-600 transition-all text-sm">Close</button>
            </div>
        </div>
    </div>

    <script>
        function refreshPanels() {
            htmx.ajax('GET', '/api/services/status', { target: '#services-grid', swap: 'innerHTML' });
            htmx.ajax('GET', '/api/deploy-panel', { target: '#deploy-panel', swap: 'innerHTML' });
        }
        document.addEventListener('DOMContentLoaded', refreshPanels);
        setInterval(refreshPanels, 30000);
    </script>
</body>
</html>`))

var deployPanelTmpl = template.Must(template.New("deployPanel").Parse(
	`{{range .}}<div class="flex items-center justify-between py-2 border-b border-gray-800 last:border-0">
		<div class="flex items-center gap-2">
			<span class="text-gray-200 text-sm">{{.Service}}</span>
			<span class="text-xs {{.StatusBadge}} px-2 py-0.5 rounded-full">{{.Status}}</span>
		</div>
		<span class="text-gray-500 text-xs font-mono">{{.Timestamp}}</span>
	</div>{{end}}`))

var serviceCardTmpl = template.Must(template.New("serviceCard").Parse(
	`{{range .}}<div class="card-hover bg-gray-900/50 border border-gray-800 rounded-2xl p-5 cursor-pointer"
        hx-get="/api/service-panel?id={{.ID}}"
        hx-trigger="click"
        hx-target="#service-modal-content"
        hx-swap="innerHTML"
        onclick="document.getElementById('service-modal').classList.remove('hidden')">
        <div class="flex items-start justify-between mb-3">
            <div class="flex items-center gap-3">
                <span class="text-xl">{{.Icon}}</span>
                <div>
                    <h3 class="text-gray-200 font-semibold text-sm">{{.DisplayName}}</h3>
                    <span class="text-gray-600 text-xs">{{.Name}}</span>
                </div>
            </div>
            <span class="status-dot {{.DotColor}}"></span>
        </div>
        <p class="text-gray-500 text-xs mb-3 line-clamp-1">{{.Description}}</p>
        <div class="flex items-center justify-between mt-3 pt-3 border-t border-gray-800">
            <span class="{{.StatusColor}} text-xs font-medium">{{.Status}}</span>
            <span class="text-gray-600 text-xs">{{.LatencyStr}}</span>
        </div>
    </div>{{end}}`))

var adminTmpl = template.Must(template.New("admin").Parse(`<!DOCTYPE html>
<html lang="en" class="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Admin - 4rch3.io</title>
    <link rel="icon" type="image/svg+xml" href="/favicon.svg">
    <meta name="csrf-token" content="{{.CSRFToken}}">
    <script src="https://unpkg.com/htmx.org@2.0.4" integrity="sha384-7uoN2oWEkBN1YI9ZOKj1jF3OGfC2jqF6elWMrbFU+dxl3NTah3MsiGZ+36I96u7Y" crossorigin="anonymous"></script>
    <script src="https://cdn.tailwindcss.com"></script>
    ` + tailwindConfig + `
    ` + baseStyle + `
</head>
<body class="grid-bg min-h-screen">
    ` + navBar + `
                    <span class="text-gray-600 text-sm">/</span>
                    <span class="text-neon-cyan text-sm font-semibold">admin</span>
                </div>
                <div class="flex items-center gap-3">
                    <a href="/" class="text-gray-500 hover:text-gray-300 transition-colors" title="dashboard">
                        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>
                    </a>
                    <span class="text-gray-500 text-sm">{{.AdminUsername}}</span>
                    <a href="/logout" class="text-gray-500 hover:text-red-400 transition-colors" title="logout">
                        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
                    </a>
                </div>
            </div>
        </div>
    </nav>

    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <h2 class="text-2xl font-bold text-white mb-6">Admin Panel</h2>

        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
            <div class="bg-gray-900/50 border border-gray-800 rounded-2xl p-6">
                <h3 class="text-lg font-semibold text-white mb-4">Users ({{.UserCount}})</h3>
                <div class="space-y-2">
                    {{range .Users}}
                    <div class="flex items-center justify-between py-2 border-b border-gray-800 last:border-0">
                        <div>
                            <span class="text-gray-200">{{.Username}}</span>
                            <span class="{{.RoleColor}} text-xs ml-2">{{.Role}}</span>
                        </div>
                        <span class="{{.StatusColor}} text-xs">{{.StatusText}}</span>
                    </div>
                    {{end}}
                </div>
            </div>

            <div class="bg-gray-900/50 border border-gray-800 rounded-2xl p-6">
                <h3 class="text-lg font-semibold text-white mb-4">Services ({{.ServiceCount}})</h3>
                <div class="space-y-2">
                    {{range .Services}}
                    <div class="flex items-center justify-between py-2 border-b border-gray-800 last:border-0">
                        <div>
                            <span class="text-gray-200">{{.DisplayName}}</span>
                            <span class="text-gray-500 text-xs ml-2">{{.Name}}</span>
                        </div>
                        <span class="{{.StatusColor}} text-xs">{{.Status}}</span>
                    </div>
                    {{end}}
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
                    {{range .Logs}}
                        <tr class="border-b border-gray-800 text-gray-400">
                            <td class="py-2 pr-4 text-xs">{{.CreatedAt}}</td>
                            <td class="py-2 pr-4">{{.UserID}}</td>
                            <td class="py-2 pr-4">{{.Action}}</td>
                            <td class="py-2 pr-4">{{.Resource}}</td>
                            <td class="py-2 text-xs text-gray-600">{{.Detail}}</td>
                        </tr>
                    {{end}}
                    </tbody>
                </table>
            </div>
        </div>
    </main>
</body>
</html>`))

var servicePanelTmpl = template.Must(template.New("servicePanel").Parse(`<div class="flex items-start justify-between mb-4">
        <div>
            <h3 class="text-lg font-semibold text-white">{{.DisplayName}}</h3>
            <p class="text-gray-500 text-sm">{{.Description}}</p>
        </div>
        <div class="flex items-center gap-2">
            {{if .PublicURL}}<a href="{{.PublicURL}}" target="_blank" class="px-3 py-2 bg-gray-800/50 border border-gray-700 rounded-lg text-gray-300 hover:text-white hover:border-gray-600 text-sm">Open</a>{{end}}
            <a href="/services/{{.ID}}" class="px-3 py-2 bg-neon-purple/20 border border-neon-purple/30 rounded-lg text-neon-purple hover:bg-neon-purple/30 text-sm">Manage Service</a>
        </div>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div class="bg-gray-950/50 border border-gray-800 rounded-xl p-3"><h4 class="text-gray-300 text-sm mb-2">CPU %</h4><div id="cpuChart"></div></div>
        <div class="bg-gray-950/50 border border-gray-800 rounded-xl p-3"><h4 class="text-gray-300 text-sm mb-2">Memory</h4><div id="memChart"></div></div>
        <div class="bg-gray-950/50 border border-gray-800 rounded-xl p-3"><h4 class="text-gray-300 text-sm mb-2">Network (KB/s)</h4><div id="netChart"></div></div>
    </div>
    <link rel="stylesheet" href="https://unpkg.com/uplot@1.6.30/dist/uPlot.min.css"/>
    <script src="https://unpkg.com/uplot@1.6.30/dist/uPlot.iife.min.js"></script>
    <script>
    (function(){
      function initCharts(){
        if(typeof uPlot==='undefined'){setTimeout(initCharts,100);return;}
        var id = {{.ID}};
        var maxPoints = 300;
        function mkData(){
          var xs = [];
          var ys1 = []; var ys2 = []; var ys3a = []; var ys3b = [];
          return {xs: xs, cpu: ys1, mem: ys2, rx: ys3a, tx: ys3b};
        }
        var d = mkData();
        function push(pt){
          d.xs.push(pt.ts);
          d.cpu.push(pt.cpu);
          d.mem.push(pt.mem_used);
          d.rx.push(pt.rx_kbps);
          d.tx.push(pt.tx_kbps);
          if(d.xs.length>maxPoints){ d.xs.shift(); d.cpu.shift(); d.mem.shift(); d.rx.shift(); d.tx.shift(); }
        }
        var optsBase = {width: 320, height: 140, legend: {show:false}, axes:[{stroke:'#666'},{stroke:'#666'}], cursor: {drag: {x:true, y:false}} };
        var cpuPlot = new uPlot(Object.assign({}, optsBase, {title: '', series:[{}, {stroke:'#22c55e'}]}), [[],[]], document.getElementById('cpuChart'));
        var memPlot = new uPlot(Object.assign({}, optsBase, {series:[{}, {stroke:'#60a5fa'}]}), [[],[]], document.getElementById('memChart'));
        var netPlot = new uPlot(Object.assign({}, optsBase, {series:[{}, {label:'rx', stroke:'#a78bfa'}, {label:'tx', stroke:'#f472b6'}]}), [[],[],[]], document.getElementById('netChart'));
        var es = new EventSource('/api/service-stats?id='+id);
        window.__svcStats = es;
        es.onmessage = function(ev){
          try {
            var pt = JSON.parse(ev.data);
            pt.rx_kbps = (pt.rx/1024).toFixed(1)*1; pt.tx_kbps = (pt.tx/1024).toFixed(1)*1;
            push(pt);
            cpuPlot.setData([d.xs, d.cpu]);
            memPlot.setData([d.xs, d.mem]);
            netPlot.setData([d.xs, d.rx, d.tx]);
          } catch(e){}
        };
        es.onerror = function(){};
      }
      initCharts();
    })();
    </script>`))

var serviceManageTmpl = template.Must(template.New("serviceManage").Parse(`<!DOCTYPE html>
<html lang="en" class="dark">
<head>
    <meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.DisplayName}} - Manage</title>
    <script src="https://unpkg.com/htmx.org@2.0.4"></script>
    <script src="https://cdn.tailwindcss.com"></script>
    
    <script>
        tailwind.config = {
            darkMode: 'class',
            theme: {
                extend: {
                    colors: {
                        neon: {
                            cyan: '#00f0ff',
                            purple: '#a855f7',
                            pink: '#ec4899',
                        }
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
    </style>
</head>
<body class="grid-bg min-h-screen">
    <main class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div class="mb-4">
            <a href="/" class="inline-flex items-center gap-2 text-gray-400 hover:text-white text-sm">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/><line x1="20" y1="12" x2="9" y2="12"/></svg>
                Back to Portal
            </a>
        </div>
        <div class="flex items-center justify-between mb-6">
            <div>
                <h2 class="text-2xl font-bold text-white">{{.DisplayName}}</h2>
                <p class="text-gray-500 text-sm">{{.Description}}</p>
            </div>
            <div class="flex gap-2">
            {{if .PublicURL}}<a href="{{.PublicURL}}" target="_blank" class="px-3 py-2 bg-gray-800/50 border border-gray-700 rounded-lg text-gray-300 hover:text-white hover:border-gray-600 text-sm">Open</a>{{end}}
                <a href="/svc/{{.Name}}" class="px-3 py-2 bg-gray-800/50 border border-gray-700 rounded-lg text-gray-300 hover:text-white hover:border-gray-600 text-sm">Proxy</a>
            </div>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div class="bg-gray-900/50 border border-gray-800 rounded-2xl p-6">
                <h3 class="text-lg font-semibold text-white mb-3">Info</h3>
                <div class="text-gray-400 text-sm space-y-1">
                    <div><span class="text-gray-500">Internal:</span> {{.InternalURL}}</div>
                    <div><span class="text-gray-500">Health:</span> {{.HealthPath}}</div>
                    <div><span class="text-gray-500">Status:</span> {{.Status}}</div>
                </div>
            </div>
            <div class="bg-gray-900/50 border border-gray-800 rounded-2xl p-6">
                <h3 class="text-lg font-semibold text-white mb-3">Actions</h3>
                {{if .DeployAllowed}}
                <button id="deployBtn" class="px-4 py-2 bg-neon-cyan/20 border border-neon-cyan/30 rounded-xl text-neon-cyan hover:bg-neon-cyan/30 text-sm">Deploy</button>
                <script>
                    document.getElementById('deployBtn').addEventListener('click', function(){
                        fetch('/api/webhook/deploy', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({service: '{{.Name}}'})})
                          .then(r=>r.json()).then(()=>alert('Deploy triggered'));
                    });
                </script>
                {{else}}
                <p class="text-gray-500 text-sm">Deploy not enabled for this service.</p>
                {{end}}
            </div>
        </div>
    </main>
</body></html>`))
