# 4rch3.io Portal — Plan

## Phase 1: Critical Security Fixes (Completed 2026-05-27)

- [x] Fix WebSocket proxy — `responseWriter` wrappers now implement `http.Hijacker` + `http.Flusher`
- [x] Fix WebSocket scheme — compute `wsScheme` from original target scheme, don't mutate shared URL
- [x] Fix LAN bypass exploit — rewrite `realIP()` with correct trusted-proxy logic
- [x] Strip `X-Forwarded-For` before setting client IP in proxy forwarding

## Phase 2: High-Priority Security Fixes (Completed 2026-05-27)

- [x] **Disabled user access bypass** — Added `!user.Active` check in session and JWT auth paths. Inactive users have session deleted; JWT requests proceed as unauthenticated. (`internal/middleware/middleware.go`)
- [x] **Unauthenticated webhook** — Applied `RequireAuth` + API rate limiter to `/api/webhook/deploy` and `/api/deploys`. (`cmd/server/main.go:184-186`)
- [x] **JSON login CSRF bypass** — JSON login requests must include `X-CSRF-Token` header, validated via HMAC-SHA256. Added `/api/auth/csrf` endpoint for token retrieval. (`internal/handlers/auth.go:50-75`)
- [x] **Admin CSRF protection** — Admin POST requires `csrf_token` in JSON body; DELETE requires `X-CSRF-Token` header. Admin dashboard embeds CSRF token in `<meta>` tag. (`internal/handlers/admin.go`)
- [x] **Admin error message leaking** — Replaced raw `err.Error()` with generic messages; original errors logged via `slog.Error`.

## Phase 3: Medium-Priority Fixes

- [ ] **writeJSONError string injection** — Replace string concatenation with `json.Marshal` in `middleware.go:107-111`
- [ ] **UseInviteCode success/error mismatch** — Return `(false, err)` on `tx.Commit()` failure in `db.go:473`
- [ ] **Health checker uncancelleable** — Use `http.NewRequestWithContext(ctx, ...)` instead of `client.Get(target)` in `health.go:99`
- [ ] **RecordHealthCheck error dropped** — Log the error return from `RecordHealthCheck` in `health.go:146`
- [ ] **Handler getIP bypasses proxy validation** — Replace `handlers/auth.go:getIP()` with call to `middleware.realIP()` or equivalent trusted-proxy logic
- [x] **Admin errors leak schema** — Map SQLite constraint errors to generic messages; log original error server-side. (`admin.go`) (done in Phase 2)
- [ ] **Template errors ignored** — Handle `template.Execute()` errors in dashboard.go and admin.go; return HTTP 500 on failure

## Phase 4: Low-Priority Fixes

- [ ] **Rate limiter goroutine leak** — Add `context.Context` / done channel to `RateLimiter` for graceful shutdown
- [ ] **Float equality in rate limiter cleanup** — Use `limiter.Tokens() >= float64(rl.burst)-0.01` or track last-access time instead
- [ ] **Port env var silent parse failure** — Check `fmt.Sscanf` return value for `PORTAL_PORT` env override (`main.go:279`)
- [ ] **CreateService returns 0 on conflict** — Query existing row ID on upsert conflict in `db.go:265`

## Phase 5: Architecture & Performance

- [ ] SQLite: separate read/write pools or PRAGMA tuning for concurrent reads
- [ ] Add prepared statements for hot database queries
- [ ] Health monitor: `sync.Pool` for HTTP clients; exponential backoff on retries
- [ ] Health monitor: circuit breaker for persistently offline services
- [ ] Reverse proxy: response caching with LRU cache for static content
- [ ] Reverse proxy: `MaxIdleConnsPerHost` on proxy transport
- [ ] Dashboard: replace HTMX polling with Server-Sent Events (SSE)
- [ ] Dashboard: pre-render and cache HTML; invalidate on discovery change only

## Phase 6: Feature Roadmap

### Near-Term
- [ ] Service grouping/tags via `4rch3.io.category` and `4rch3.io.group` labels
- [ ] Client-side dashboard search/filter for services
- [ ] Config hot-reload via `fsnotify` + SIGHUP handler
- [ ] API key management UI (backend supports keys, no UI to create/revoke)

### Mid-Term
- [ ] Proxmox VE discovery via API (config fields exist)
- [ ] OIDC/OAuth2 provider integration (Authentik / Keycloak)
- [ ] CLI tool (`4rch3ctl`) — Cobra-based management CLI
- [ ] Notification alerts — Discord/Slack/NTFY webhooks on health state changes
- [ ] Service dependency map with visual graph on dashboard
- [ ] Audit log expansion — config changes, service creation, proxy access events

### Long-Term
- [ ] Multi-node support — central portal federating with agent instances
- [ ] DNS management — Traefik/Cloudflare API integration for service DNS
- [ ] Container actions — start/stop/restart from dashboard
- [ ] Web terminal (gotty-style) into containers from dashboard
- [ ] GitOps config — watch git repo for config changes and auto-apply