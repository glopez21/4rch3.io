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
}

type ServiceCardData struct {
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
    <script src="https://unpkg.com/htmx.org@2.0.4"></script>
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
            <form hx-post="/login" hx-target="this" hx-swap="outerHTML" class="space-y-6">
                <div id="login-error" class="text-red-400 text-sm text-center"></div>

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
    ` + tailwindConfig + `
    ` + baseStyle + `
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
    ` + navBar + `
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
            <div class="card-hover bg-gray-900/50 border border-gray-800 rounded-2xl p-5">
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

        <div class="mt-12 border-t border-gray-800 pt-8 text-center">
            <h3 class="text-lg font-semibold text-white mb-4">Quick Links</h3>
            <div class="inline-flex flex-wrap justify-center gap-3">
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
</html>`))

var serviceCardTmpl = template.Must(template.New("serviceCard").Parse(
	`{{range .}}<div class="card-hover bg-gray-900/50 border border-gray-800 rounded-2xl p-5">
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
    <script src="https://unpkg.com/htmx.org@2.0.4"></script>
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