# 4rch3.io Portal — Development Notes

## Project Overview

The `4rch3.io` portal is a Go-based homelab operations hub providing authentication,
service discovery, reverse proxy, and health monitoring for the homelab ecosystem.

## Directory Structure

```
~/projects/4rch3.io/
├── cmd/server/main.go         # Entry point — wires everything together
├── internal/
│   ├── auth/
│   │   ├── auth.go            # JWT + bcrypt + API key generation
│   │   └── service.go         # Auth service (login, create user, validate, session lookup)
│   ├── config/config.go       # YAML config with env var overrides
│   ├── database/
│   │   ├── db.go              # SQLite operations (users, sessions, services, audit)
│   │   └── schema.sql         # Full schema
│   ├── discovery/docker.go    # Docker socket REST client (zero external deps)
│   ├── handlers/
│   │   ├── auth.go            # Login/logout/me endpoints
│   │   ├── dashboard.go       # Dashboard with HTMX live status
│   │   ├── admin.go           # Admin panel with user/service/audit views
│   │   └── templates.go       # html/template definitions with auto-escaping
│   ├── middleware/
│   │   ├── middleware.go        # Auth middleware (session + JWT + API key), LAN detection, security headers
│   │   ├── cors.go              # CORS middleware (config-driven)
│   │   └── ratelimit.go         # Per-IP rate limiting (token bucket)
│   ├── models/models.go       # User, Service, Session, AuditLog, etc.
│   ├── monitor/health.go      # Periodic health checker
│   └── proxy/proxy.go         # Reverse proxy to internal services
├── config.yaml                # Default configuration
├── Dockerfile                 # Multi-stage build (~4.5MB)
├── docker-compose.yml         # Traefik-integrated deployment
├── NOTES.md                   # This file
└── ISSUES.md                  # Known issues & resolved items
```

## Architecture

```
                            Internet
                               |
                         Cloudflare DNS
                               |
                     Cloudflare Tunnel (or DDNS)
                               |
                     ┌─────────┴─────────┐
                     │     Traefik        │
                     │  (reverse proxy)   │
                     └─────────┬─────────┘
                               |
                     ┌─────────▼─────────┐
                     │  4rch3.io Portal   │
                     │  (Go binary)       │
                     │  :8081             │
                     └─────────┬─────────┘
                               |
                ┌──────────────┼──────────────┐
                │              │              │
         ┌──────▼─────┐ ┌─────▼─────┐ ┌─────▼─────┐
         │ ThreatPulse │ │ LogSentry │ │ AlertFlow │
         └────────────┘ └───────────┘ └───────────┘
```

## Auth Strategy (Combined)
- **Primary:** Session-based auth (HttpOnly cookies) for browser users — cookie
  token is looked up in the `sessions` table, not validated as JWT
- **API Auth:** Bearer JWT or API keys (`4rch3_` prefixed) for machine-to-machine
- **Middleware chain:** `Authenticate` runs globally on all routes, populating user
  context from session cookie / JWT / API key. `RequireAuth` and `RequireRole`
  are route-specific gates that check for valid user context.
- **Future:** OIDC/OAuth2 provider integration
- **Roles:** admin, user, readonly, service
- **Bootstrap:** Create initial admin via env vars:
  ```bash
  PORTAL_ADMIN_USERNAME=admin \
  PORTAL_ADMIN_PASSWORD=<your-password> \
  go run ./cmd/server -bootstrap
  ```
- **Password reset:** Reset a user's password via env vars:
  ```bash
  PORTAL_ADMIN_USERNAME=admin PORTAL_ADMIN_PASSWORD=<new-password> go run ./cmd/server -reset-password
  ```
- **Cookie:** `Secure` flag set based on `X-Forwarded-Proto` header with
  `r.TLS != nil` fallback (works correctly behind Traefik TLS termination)

## Docker Service Discovery

Discovery communicates directly with the Docker socket via HTTP REST API
(no heavy SDK dependencies). Containers need labels to be discovered:

```yaml
labels:
  - "4rch3.io.enable=true"
  - "4rch3.io.service=my-service"
  - "4rch3.io.name=My Service"
  - "4rch3.io.description=Does cool things"
  - "4rch3.io.category=security"
  - "4rch3.io.icon=🔒"
  - "4rch3.io.health=/health"
  - "4rch3.io.public=true"
```

Docker API call: `GET /containers/json?filters={"label":{"4rch3.io.enable":true}}`

## Proxy Routing

Requests to `/svc/<service-name>` are reverse-proxied to the discovered internal
URL of that service. User identity forwarded via `X-Forwarded-User` and
`X-Forwarded-Role` headers set directly on the proxied request.

## Monitoring

Periodic HTTP health checks on discovered services. Status stored in SQLite
with rolling history. Live status via HTMX on the dashboard.

## Running Locally

```bash
# 1. Bootstrap admin user (first time only)
cd ~/projects/4rch3.io
PORTAL_ADMIN_USERNAME=admin \
PORTAL_ADMIN_PASSWORD=<your-password> \
go run ./cmd/server -bootstrap

# 2. Start server
go run ./cmd/server

# 3. Access from LAN
# http://<host-ip>:8081
```

## Known Bugs / Edge Cases

- **SQLite time scanning:** go-sqlite3 driver only auto-parses timestamps for
  columns with `DATE`/`TIME` in their declared type name. All `TEXT` time fields
  are scanned as `sql.NullString` and manually parsed with `time.Parse`.
- **NULL handling:** `mfa_secret` and `api_key` columns can be NULL in DB but
  were originally scanned into `string` — fixed with `sql.NullString`.
- **Docker filter encoding:** Docker API filters must be URL-encoded JSON.
  The value must be `true` (not `["true"]`).
- **XSS hardening:** All HTML rendering uses `html/template` with auto-escaping.
  Dynamic content (service names, descriptions, usernames, audit logs) is
  properly escaped to prevent stored XSS via Docker labels.
- **Session auth flow:** Cookie tokens are session tokens (random base64), not
  JWTs. The `Authenticate` middleware routes cookie tokens through
  `db.GetSession()` → `db.GetUserByID()`. Bearer tokens go through
  `ValidateJWT()`. API keys go through `ValidateAPIKey()`.

## Deployment (Docker)

```bash
docker compose up -d
```

Traefik handles TLS termination. Portal sits behind Traefik on the `devnet`
network.

## Phase 1 Complete — Observability & Guard Rails

The following were implemented to close the gap between config.yaml and actual runtime behavior:

- **CORS middleware** (`internal/middleware/cors.go`): Respects `cors.enabled` and `cors.allowed_origins`. Handles preflight OPTIONS, sets credentials/allowed methods/headers. No-ops when disabled or no Origin header.
- **Rate limiting** (`internal/middleware/ratelimit.go`): Per-IP token bucket using `golang.org/x/time/rate`. Auth endpoints limited to `rate_limit_per_min: 10` (burst 1). API endpoints limited to `api_rate_limit_per_min: 60` (burst 5). Background cleanup every 10 minutes.
- **Structured logging** (`log/slog`): All `log.Printf` replaced with `slog.Info/Warn/Error/Debug` across 6 source files. Config-driven via `logging.level` (debug/info/warn/error) and `logging.format` (text/json). Request logging in middleware emits structured attrs (method, path, status, duration).

## Completed Upgrades (May 2026)

- [x] **Prometheus /metrics endpoint** (`internal/metrics/metrics.go`) — request count, duration, active services, auth attempts, health check results, sessions, uptime
- [x] **WebSocket proxy support** (`internal/proxy/proxy.go`) — detects `Upgrade: websocket`, hijacks connections, bidirectional copy
- [x] **Structured JSON error responses** — all handlers return `{"error":"..."}` JSON instead of plain text
- [x] **Docker HEALTHCHECK** — `wget`-based health check at 30s interval
- [x] **Non-root container user** — `portal` user in Dockerfile
- [x] **JWT secret validation** — catches both Go default and config.yaml default
- [x] **Rate limiter state leak** — cleanup only evicts full-bucket visitors, doesn't wipe map
- [x] **Proxy header injection fix** — strips incoming `X-Forwarded-User`/`X-Forwarded-Role` before setting
- [x] **Session ID ≠ token** — separate random ID value, no longer leaks session token
- [x] **SameSite=Strict** — changed from LaxMode for CSRF mitigation
- [x] **CSRF protection** — HMAC-SHA256 tokens on login form POST
- [x] **AuthConfig deduplication** — removed duplicated struct; individual constructor params
- [x] **Hardcoded cookie name** — middleware now reads from config
- [x] **SQLite time scan fix** — TEXT columns scanned as string, parsed with `time.Parse`

## Optimization Recommendations

### Priority 1: Critical Bugs (Completed)

- [x] Fix proxy.go — X-Forwarded-User/Role headers built on ProxyRequest but never sent
- [x] Fix port mismatch — config.yaml:8081 vs Dockerfile/docker-compose:8080
- [x] Fix XSS vulnerability — migrate from string concatenation to `html/template`
- [x] Fix go.mod version — `go 1.26.3` does not exist; ran `go mod tidy` → `1.25.0`
- [x] Fix cookie Secure flag — use `X-Forwarded-Proto` instead of `r.TLS != nil`
- [x] Fix deferred `resp.Body.Close()` inside retry loop in monitor/health.go
- [x] Fix auth middleware — `Authenticate` was only on `/` and `/svc/`, not `/admin`;
      moved to global middleware chain
- [x] Fix session auth — cookie tokens were validated as JWT (always failing);
      added `fromCookie` path that looks up sessions in DB
- [x] Add password reset CLI flag (`-reset-password`)

### Priority 2: Performance

- [ ] SQLite: separate read/write pools or PRAGMA tuning for concurrent reads
- [ ] Add prepared statements for hot database queries
- [ ] Health monitor: `sync.Pool` for HTTP clients; exponential backoff on retries
- [ ] Health monitor: circuit breaker for persistently offline services
- [ ] Reverse proxy: response caching with LRU cache for static content
- [ ] Reverse proxy: `MaxIdleConnsPerHost` on proxy transport
- [ ] Dashboard: replace HTMX polling with Server-Sent Events (SSE)
- [ ] Dashboard: pre-render and cache HTML; invalidate on discovery change only

### Priority 3: Architecture

- [ ] Move all HTML to `web/templates/` with `html/template` and `template.ParseGlob`
- [ ] Add `html/template` template composition (layout, partials, service_card)
- [x] Replace `log.Printf` with `slog` structured logging; wire `config.logging.level`
- [x] Implement CORS middleware (config keys exist but unused)
- [x] Implement rate limiting with `golang.org/x/time/rate` (config keys exist)
- [ ] Add `X-Request-ID` middleware for request tracing
- [ ] Add session garbage collection goroutine
- [x] Remove `replace` directive for `docker/distribution` — already removed
- [x] Unify `auth.AuthConfig` with `config.AuthConfig` (remove duplication in main.go)
- [x] Add config validation at startup (fail fast on missing jwt_secret, etc.)
- [ ] Add a `Makefile` with build, test, lint, docker-build targets

#### Near-Term (High Value, Low Effort)

- [x] WebSocket proxy support (`Upgrade` header detection in reverse proxy)
- [x] Prometheus `/metrics` endpoint (service count, health distribution, auth attempts)
- [ ] Service grouping/tags via `4rch3.io.category` and `4rch3.io.group` labels
- [ ] Client-side dashboard search/filter for services
- [ ] Config hot-reload via `fsnotify` + SIGHUP handler
- [ ] API key management UI (backend supports keys, no UI to create/revoke)

#### Mid-Term

- [ ] Proxmox VE discovery via API (config fields exist)
- [ ] OIDC/OAuth2 provider integration (Authentik / Keycloak)
- [ ] CLI tool (`4rch3ctl`) — Cobra-based management CLI
- [ ] Notification alerts — Discord/Slack/NTFY webhooks on health state changes
- [ ] Service dependency map with visual graph on dashboard
- [ ] Audit log expansion — config changes, service creation, proxy access events

#### Long-Term

- [ ] Multi-node support — central portal federating with agent instances
- [ ] DNS management — Traefik/Cloudflare API integration for service DNS
- [ ] Container actions — start/stop/restart from dashboard
- [ ] Web terminal (gotty-style) into containers from dashboard
- [ ] GitOps config — watch git repo for config changes and auto-apply

## Future Roadmap

- [x] Fix session cookie persistence in browser (HTMX redirect flow)
- [x] Fix auth middleware not running on all routes
- [x] Fix session tokens validated as JWT instead of DB lookup
- [x] Add password reset CLI flag (`-reset-password`)
- [ ] OIDC/OIDC provider integration (Keycloak / Authentik)
- [ ] Proxmox VM/LXC discovery via API
- [ ] WebSocket terminal for containers
- [ ] TOTP/WebAuthn MFA
- [ ] Service-specific permissions (IAM)
- [ ] Public status page
- [ ] CLI tool for management
- [ ] Audit log viewer in admin panel
- [ ] Tailscale integration for LAN-less access
- [x] Metrics export (Prometheus)
- [x] Bootstrap admin via env vars at startup (not just CLI flag)
