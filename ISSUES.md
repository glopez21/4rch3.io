# 4rch3.io Portal — Known Issues & Conflicts

## Current Issues

### 1. Session Cookie Not Persisting After Login (Browser)
- **Status:** Fixed
- **Severity:** High
- **Description:** After login via HTMX form, the browser redirects to `/` but
  the session cookie (`4rch3_session`) doesn't persist. Login works correctly
  via curl/JSON API — cookie is set and subsequent requests are authenticated.
- **Root cause:** Two bugs compounded:
  1. The `Authenticate` middleware was only applied to `/` and `/svc/` routes,
     not to `/admin` or `/api/services/status`. `RequireAuth` on those routes
     always saw `nil` user context and redirected to login. Moved `Authenticate`
     to the global middleware chain so it runs on every request.
  2. The `Authenticate` middleware took the session token from the cookie and
     tried to validate it as a JWT. Since session tokens are random base64
     (not JWTs), validation always failed and the user was never authenticated.
     Added `fromCookie` flag to route cookie tokens through `db.GetSession()`
     → `db.GetUserByID()` instead of `ValidateJWT()`.
- **Fix:** Moved `Authenticate` to global middleware chain in `cmd/server/main.go`.
  Added session-based auth path in `Authenticate`. Added `GetSession()`/`DeleteSession()`
  to `auth.Service`.
- **Reference:** `internal/middleware/middleware.go:Authenticate()`, `cmd/server/main.go`

### 2. Dashboard Service Card Rendering
- **Status:** Open
- **Severity:** Low
- **Description:** The initial dashboard page load shows an empty grid area
  until HTMX refreshes from `/api/services/status`. Should pre-render the
  initial service cards server-side.
- **Reference:** `internal/handlers/dashboard.go:renderDashboard()`

### 3. SQLite Concurrent Write Limitations
- **Status:** Open
- **Severity:** Medium
- **Description:** SQLite WAL mode is enabled but only 1 connection is used
  (`SetMaxOpenConns(1)`). Under heavy monitoring (many services, short intervals),
  write contention could occur. Monitor for `database is locked` errors.
- **Potential fix:** Write batching in the monitor or switch to Postgres.

### 4. Docker Socket Permission
- **Status:** Open
- **Severity:** High (deployment)
- **Description:** The container needs access to `/var/run/docker.sock` which
  requires either running as root or adding the container user to the `docker`
  group. The Dockerfile currently runs as root.
- **Fix:** Consider using `docker-socket-proxy` for safer socket access.

### 5. First-Run Bootstrap
- **Status:** Fixed
- **Severity:** Medium
- **Description:** The `-bootstrap` CLI flag creates the initial admin user, and
  `tryBootstrapFromEnv` auto-creates an admin at startup if none exists. A
  `-reset-password` flag has been added for resetting existing user passwords.
- **Reference:** `cmd/server/main.go:handleBoostrap()`, `tryBootstrapFromEnv()`,
  `handleResetPassword()`

### 6. Session Token Cleanup
- **Status:** Open
- **Severity:** Low
- **Description:** Expired sessions are never cleaned up from the database.
  Over time this will accumulate stale rows.
- **Fix:** Add a periodic session cleanup goroutine.

### 7. Missing Graceful Docker Connection Handling
- **Status:** Open
- **Severity:** Low
- **Description:** If the Docker socket is unavailable on startup, the service
  logs a warning and continues without discovery. However, it doesn't retry
  automatically.
- **Fix:** Add retry logic or a health-dependent restart.

### 8. Service Discovery with No Docker Socket
- **Status:** Open
- **Severity:** Low
- **Description:** When running outside Docker (directly on host), the Docker
  socket might not be available. The current code handles this gracefully but
  the dashboard is empty without services.

### 9. Proxy Headers Not Forwarded to Backend Services
- **Status:** Fixed
- **Severity:** Critical
- **Description:** In `proxy.go`, `X-Forwarded-User` and `X-Forwarded-Role`
  headers are set on `rewrite.Out.Header` via `httputil.ProxyRequest`, but the
  rewritten request is never used. `proxy.ServeHTTP(w, r)` receives the
  original request, so the custom headers are never sent to backend services.
- **Fix:** Set headers directly on `r` before proxying, remove unused
  `ProxyRequest` construction.
- **Reference:** `internal/proxy/proxy.go:Handler()`

### 10. Port Mismatch Across Config, Dockerfile, and docker-compose
- **Status:** Fixed
- **Severity:** High
- **Description:** `config.yaml` sets port 8081, `config.go` DefaultConfig sets
  8080, `Dockerfile` EXPOSEs 8080, and `docker-compose.yml` maps 8080:8080 —
  plus Traefik label points to port 8080. Since docker-compose mounts
  config.yaml into the container, the app listens on 8081 but the container
  port mapping and Traefik point to 8080. This is a deployment-breaking bug.
- **Fix:** Aligned all to port 8081: DefaultConfig, Dockerfile, docker-compose,
  and Traefik label.

### 11. XSS Vulnerability — Unsanitized HTML Rendering
- **Status:** Fixed
- **Severity:** Critical
- **Description:** All HTML rendering uses `fmt.Sprintf` / string concatenation
  instead of `html/template`. Service names, descriptions, display names, and
  user data from Docker labels (attacker-controllable) are interpolated
  directly into HTML without escaping. This allows stored XSS via crafted
  Docker label values.
- **Fix:** Migrated all handlers to `html/template` with auto-escaping.
  Created `templates.go` with parsed templates and data structs for dashboard,
  service cards, and admin panel.
- **Affects:** `handlers/dashboard.go`, `handlers/admin.go`,
  `handlers/auth_templates.go`

### 12. Invalid Go Version in go.mod
- **Status:** Fixed
- **Severity:** Medium
- **Description:** `go.mod` declared `go 1.26.3` which does not exist as a stable
  release. The Dockerfile used `golang:1.23-alpine`.
- **Fix:** Ran `go mod tidy` which set `go 1.25.0` based on the installed
  toolchain. Updated Dockerfile to `golang:1.25-alpine` for compatibility.

### 13. Cookie Secure Flag Incorrect Behind TLS Termination
- **Status:** Fixed
- **Severity:** Medium
- **Description:** Cookie `Secure` flag is set based on `r.TLS != nil`, but
  behind Traefik with TLS termination, the Go server sees plain HTTP, so
  `r.TLS` is always nil and Secure cookies are never set. This causes session
  cookies to be sent over HTTP-only even when the client connection is HTTPS.
- **Fix:** Changed to check `r.Header.Get("X-Forwarded-Proto") == "https"`,
  falling back to `r.TLS != nil`.
- **Reference:** `internal/handlers/auth.go:Login()`, `auth.go:Logout()`

### 14. Deferred resp.Body.Close Inside Retry Loop
- **Status:** Fixed
- **Severity:** Medium
- **Description:** In `monitor/health.go:CheckService()`, `defer resp.Body.Close()`
  is called inside a retry loop, which defers all response body closes until
  the function returns. This leaks HTTP connections under retry scenarios.
- **Fix:** Close `resp.Body` immediately after reading, before the next retry.
- **Reference:** `internal/monitor/health.go:CheckService()`

## Resolved Issues

### R-1. Docker SDK Dependency Conflict
- **Status:** Resolved (2026-05-11)
- **Fix:** Replaced official Docker Go SDK with a lightweight REST API client
  that talks directly to the Docker socket. Avoids heavy dependency chain and
  platform-specific build failures (`sockets.DialPipe` on Linux).

### R-2. Auth Service Self-Import
- **Status:** Resolved (2026-05-11)
- **Fix:** `internal/auth/service.go` had a self-import of the `auth` package.
  Removed the import prefix since both are in the same package.

### R-3. AuthConfig Missing SessionExpiry
- **Status:** Resolved (2026-05-11)
- **Fix:** `AuthConfig` in `auth.go` was missing the `SessionExpiry` field.
  Added it to match what the auth handler needs.

### R-4. Dashboard Type Mismatch
- **Status:** Resolved (2026-05-11)
- **Fix:** `renderDashboard` had `[]interface{}` parameter instead of
  `[]*models.Service`. Fixed type signature and `serviceCard` parameter.

### R-5. Docker API Filter Format
- **Status:** Resolved (2026-05-11)
- **Fix:** Docker label filter used `["true"]` (array) but expects `true`
  (boolean). Also added URL encoding via `url.QueryEscape()`.

### R-6. SQLite NULL Column Scanning
- **Status:** Resolved (2026-05-11)
- **Fix:** `mfa_secret` and `api_key` columns are NULL by default but were
  scanned directly into `string`. Changed to `sql.NullString` for all user
  queries (`GetUserByUsername`, `GetUserByID`, `GetUserByAPIKey`).

### R-7. SQLite Timestamp Parsing
- **Status:** Resolved (2026-05-11)
- **Fix:** The go-sqlite3 driver only auto-parses timestamps for columns with
  `DATE`/`TIME` in their declared SQL type name. All `TEXT` timestamp columns
  (`created_at`, `updated_at`, `last_login`, `expires_at`) now scanned as
  `sql.NullString` and manually parsed with `time.Parse("2006-01-02 15:04:05")`.

### R-8. CORS Config Not Wired
- **Status:** Resolved (2026-05-14)
- **Severity:** Low
- **Description:** `config.yaml` has `cors.enabled` and `cors.allowed_origins` fields
  but no CORS middleware was implemented. API calls from browser-based UIs on
  different origins would be blocked.
- **Fix:** Added `internal/middleware/cors.go` with config-driven CORS middleware.
  Handles preflight OPTIONS, credentials, allowed methods/headers. Wired in
  `main.go` as the outermost middleware layer.

### R-9. Rate Limiting Config Not Wired
- **Status:** Resolved (2026-05-14)
- **Severity:** Low
- **Description:** `config.yaml` has `auth.rate_limit_per_min` and
  `auth.api_rate_limit_per_min` but no rate limiter was implemented.
- **Fix:** Added `internal/middleware/ratelimit.go` with per-IP token bucket
  using `golang.org/x/time/rate`. Auth endpoints limited to 10 req/min (burst 1),
  API endpoints to 60 req/min (burst 5). Background cleanup every 10 minutes.

### R-10. Unstructured Logging (log.Printf)
- **Status:** Resolved (2026-05-14)
- **Severity:** Medium
- **Description:** All logging used `log.Printf` with no structured context,
  log levels, or format control. The `logging.level` and `logging.format` config
  fields were ignored.
- **Fix:** Replaced all `log.Printf`/`log.Fatalf` with `log/slog` across 6 files.
  Config-driven via `logging.level` (debug/info/warn/error) and `logging.format`
  (text/json). Request logging emits structured attrs (method, path, status,
  duration). Server startup/fatal errors use `slog.Error` + `os.Exit(1)`.

### R-11. Port Conflict with aegisSec
- **Status:** Resolved (2026-05-11)
- **Fix:** Port 8080 was in use by `aegissec-api` (run by `run-api.sh` restart
  loop in `~/projects/aegisSec/services/api/`). Switched portal to port 8081.
  aegisSec was later killed via `kill -9`.

## Conflicts

### Cookie Secure Flag Logic (Fixed)
- Previously used `r.TLS != nil` which fails behind TLS-terminating proxies.
- Now checks `X-Forwarded-Proto` header with `r.TLS != nil` as fallback.

### Auth Handler Config Duplication
- The auth handler takes an `auth.AuthConfig` which duplicates fields from the
  main `config.AuthConfig`. These should be unified or the handler should
  accept the config struct directly.

### Lean Docker Client
- The current Docker discovery uses raw HTTP to the Docker socket instead of
  the official SDK. This was done to avoid dependency hell but means we don't
  get automatic API version negotiation. Should pin an API version or add
  capability negotiation.

### Unused `replace` Directive in go.mod (Fixed)
- `replace github.com/docker/distribution v2.8.3+incompatible` existed in go.mod
  from the old Docker SDK dependency. Removed during go.mod cleanup.
