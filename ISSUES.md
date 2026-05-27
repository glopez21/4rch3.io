# 4rch3.io Portal — Known Issues & Conflicts

## Current Issues

### 1. Session Cookie Not Persisting After Login (Browser)
- **Status:** Fixed
- **Severity:** High
- **Description:** After login via HTMX form, the browser redirects to `/` but
  the session cookie (`4rch3_session`) doesn't persist.
- **Root cause:** Two bugs compounded:
  1. `Authenticate` middleware was only applied to `/` and `/svc/` routes.
  2. Cookie tokens were validated as JWT instead of DB session lookup.
- **Fix:** Moved `Authenticate` to global middleware chain. Added session-based
  auth path. Added `GetSession()`/`DeleteSession()` to `auth.Service`.
- **Reference:** `internal/middleware/middleware.go:Authenticate()`

### 2. Dashboard Service Card Rendering
- **Status:** Fixed
- **Severity:** Low
- **Description:** The initial dashboard page load showed an empty grid area
  until HTMX refreshed from `/api/services/status`.
- **Fix:** Dashboard now pre-renders service cards server-side on initial load.
  The HTMX endpoint only handles subsequent refreshes.

### 3. SQLite Concurrent Write Limitations
- **Status:** Fixed
- **Severity:** Medium
- **Description:** `SetMaxOpenConns(1)` caused write contention under
  concurrent health monitoring and dashboard access.
- **Fix:** Increased to `SetMaxOpenConns(4)`, `SetMaxIdleConns(2)`.
- **Reference:** `internal/database/db.go:Open()`

### 4. Docker Socket Permission
- **Status:** Open
- **Severity:** High (deployment)
- **Description:** The container needs access to `/var/run/docker.sock`.
  Dockerfile now runs as non-root `portal` user. The user must be in the
  Docker group or use `docker-socket-proxy` for safe socket access.
- **Fix:** Consider using `docker-socket-proxy` sidecar for production.

### 5. First-Run Bootstrap
- **Status:** Fixed
- **Severity:** Medium
- **Description:** Bootstrap admin creation via env vars.
- **Reference:** `main.go:handleBoostrap()`, `tryBootstrapFromEnv()`,
  `handleResetPassword()`

### 6. Session Token Cleanup
- **Status:** Fixed
- **Severity:** Low
- **Description:** Expired sessions accumulated in the database.
- **Fix:** Added periodic cleanup goroutine at 15-minute intervals.
- **Reference:** `cmd/server/main.go` session cleanup goroutine

### 7. Missing Graceful Docker Connection Handling
- **Status:** Open
- **Severity:** Low
- **Description:** If Docker socket is unavailable on startup, the service
  logs a warning and continues without discovery. No retry mechanism.

### 8. Service Discovery with No Docker Socket
- **Status:** Open
- **Severity:** Low
- **Description:** When running outside Docker, the dashboard is empty.

### 9. Proxy Headers Not Forwarded to Backend Services
- **Status:** Fixed
- **Severity:** Critical
- **Description:** `X-Forwarded-User` and `X-Forwarded-Role` were set on
  `Out.Header` via `ProxyRequest` but the rewritten request was never used.
- **Fix:** Set headers directly on `r` before proxying.

### 10. Port Mismatch Across Config, Dockerfile, and docker-compose
- **Status:** Fixed
- **Severity:** High
- **Description:** Config 8081 vs DefaultConfig 8080 vs Dockerfile 8080.
- **Fix:** Aligned all to port 8081.

### 11. XSS Vulnerability — Unsanitized HTML Rendering
- **Status:** Fixed
- **Severity:** Critical
- **Description:** All HTML used `fmt.Sprintf` instead of `html/template`.
- **Fix:** Migrated to `html/template` with auto-escaping.

### 12. Invalid Go Version in go.mod
- **Status:** Fixed
- **Severity:** Medium
- **Description:** `go 1.26.3` does not exist.
- **Fix:** `go mod tidy` set `go 1.25.0`. Dockerfile updated.

### 13. Cookie Secure Flag Incorrect Behind TLS Termination
- **Status:** Fixed
- **Severity:** Medium
- **Description:** `r.TLS` is nil behind Traefik, so Secure cookies never set.
- **Fix:** Check `X-Forwarded-Proto` header with `r.TLS` fallback.

### 14. Deferred resp.Body.Close Inside Retry Loop
- **Status:** Fixed
- **Severity:** Medium
- **Description:** `defer resp.Body.Close()` in retry loop leaked connections.
- **Fix:** Close body immediately after reading.

### 15. Weak Default JWT Secret Validation
- **Status:** Fixed
- **Severity:** Critical
- **Description:** `Validate()` only checked for the Go default
  `"change-me-in-production"` but not the config.yaml default
  `"CHANGE-ME-IN-PRODUCTION-USE-A-STRONG-RANDOM-SECRET"`.
- **Fix:** Added both defaults to the validation check.

### 16. Rate Limiter State Wiped Every 10 Minutes
- **Status:** Fixed
- **Severity:** High
- **Description:** Cleanup goroutine replaced the entire visitors map,
  resetting all rate limiter states. Attackers could wait 10 minutes
  for a fresh burst allowance.
- **Fix:** Only evict visitors whose token bucket is completely full.

### 17. Proxy Header Injection via X-Forwarded-{User,Role}
- **Status:** Fixed
- **Severity:** High
- **Description:** Attackers could send forged `X-Forwarded-User` headers
  that were not stripped before forwarding to backend services.
- **Fix:** Strip incoming `X-Forwarded-User` and `X-Forwarded-Role` headers
  before setting authenticated values.

### 18. Session ID Leaks Session Token
- **Status:** Fixed
- **Severity:** Medium
- **Description:** `Session.ID` was set to the same value as `Session.Token`,
  meaning any exposure of the session ID (audit log, error message) would
  leak the authentication token.
- **Fix:** Generate a separate random ID for `Session.ID`.

### 19. SameSite=LaxMode Instead of StrictMode
- **Status:** Fixed
- **Severity:** Medium
- **Description:** Lax mode allows cookies on top-level navigations from
  cross-origin sites, increasing CSRF risk for an admin panel.
- **Fix:** Changed to `http.SameSiteStrictMode`.

### 20. AuthConfig Duplicated Across Packages
- **Status:** Fixed
- **Severity:** Medium
- **Description:** `auth.AuthConfig` duplicated fields from `config.AuthConfig`.
  Both packages required manual sync on config changes.
- **Fix:** Removed `auth.AuthConfig`. `Auth.New()` now accepts individual
  parameters. Auth handler constructor simplified accordingly.

### 21. Hardcoded Cookie Name in Middleware
- **Status:** Fixed
- **Severity:** Medium
- **Description:** Middleware used `"4rch3_session"` hardcoded instead of
  the config value `session_cookie_name`.
- **Fix:** Middleware now receives cookie name via constructor.

### 22. SQLite Time Scan Errors in Test Queries
- **Status:** Fixed
- **Severity:** Medium
- **Description:** `GetRecentHealthChecks` and `GetAuditLogs` scanned TEXT
  timestamp columns directly into `time.Time` fields, causing scan errors.
- **Fix:** Scan into `string` first, then parse with `time.Parse`.

### 23. No Prometheus Metrics
- **Status:** Fixed
- **Severity:** Medium
- **Description:** No observability — no request metrics, no service
  health metrics, no way to monitor in production.
- **Fix:** Added `internal/metrics/metrics.go` with 7 Prometheus collectors
  (request count/duration, active services, auth attempts, health check
  results, active sessions, uptime). Instrument middleware + `/metrics`
  handler wired into main.go.

### 24. Proxy Doesn't Support WebSocket
- **Status:** Fixed
- **Severity:** Medium
- **Description:** `httputil.ReverseProxy` ignores `Upgrade: websocket`
  headers, breaking any WebSocket service behind the portal proxy.
- **Fix:** Added WebSocket detection and hijack-based bidirectional copy
  in `internal/proxy/proxy.go`.

### 25. Plain Text Error Responses
- **Status:** Fixed
- **Severity:** Low
- **Description:** Handlers returned plain text errors (`http.Error`),
  making API consumption inconsistent.
- **Fix:** All handlers return `{"error":"..."}` JSON. Shared helpers in
  `internal/handlers/errors.go`, `internal/middleware/middleware.go`,
  `internal/proxy/errors.go`.

### 26. No CSRF Protection on Login Form
- **Status:** Fixed
- **Severity:** Medium
- **Description:** Login form had no CSRF token validation.
- **Fix:** Added HMAC-based CSRF token generation/validation in
  `internal/auth/csrf.go`. Token embedded as hidden field, validated
  on form POST with 15-minute expiry.

## Resolved Issues

### R-1. Docker SDK Dependency Conflict
- **Status:** Resolved (2026-05-11)
- **Fix:** Replaced official Docker Go SDK with lightweight REST API client.

### R-2. Auth Service Self-Import
- **Status:** Resolved (2026-05-11)
- **Fix:** Removed self-import of `auth` package in `service.go`.

### R-3. AuthConfig Missing SessionExpiry
- **Status:** Resolved (2026-05-11)
- **Fix:** Added `SessionExpiry` field to `AuthConfig`.

### R-4. Dashboard Type Mismatch
- **Status:** Resolved (2026-05-11)
- **Fix:** Fixed `[]interface{}` to `[]*models.Service`.

### R-5. Docker API Filter Format
- **Status:** Resolved (2026-05-11)
- **Fix:** Label filter `["true"]` → `true`. Added URL encoding.

### R-6. SQLite NULL Column Scanning
- **Status:** Resolved (2026-05-11)
- **Fix:** Changed to `sql.NullString` for nullable columns.

### R-7. SQLite Timestamp Parsing
- **Status:** Resolved (2026-05-11)
- **Fix:** TEXT timestamps scanned as `sql.NullString` and manually parsed.

### R-8. CORS Config Not Wired
- **Status:** Resolved (2026-05-14)
- **Fix:** Added `internal/middleware/cors.go`.

### R-9. Rate Limiting Config Not Wired
- **Status:** Resolved (2026-05-14)
- **Fix:** Added `internal/middleware/ratelimit.go`.

### R-10. Unstructured Logging (log.Printf)
- **Status:** Resolved (2026-05-14)
- **Fix:** Replaced with `log/slog` across 6 files.

### R-11. Port Conflict with aegisSec
- **Status:** Resolved (2026-05-11)
- **Fix:** Switched portal to port 8081.

### R-12. Container Runs as Root
- **Status:** Resolved (2026-05-20)
- **Fix:** Added `portal` non-root user with `adduser -S -G portal portal`.
  Added `HEALTHCHECK` with wget at 30s intervals.

### R-13. CDN Dependencies Without Subresource Integrity
- **Status:** Resolved (2026-05-20)
- **Fix:** Added SRI `integrity` attributes to all `unpkg.com` script tags.

## Conflicts

### Cookie Secure Flag Logic (Fixed)
- Previously used `r.TLS != nil` which fails behind TLS-terminating proxies.
- Now checks `X-Forwarded-Proto` header with `r.TLS != nil` as fallback.

### Auth Handler Config Duplication (Fixed)
- `auth.AuthConfig` duplicated from `config.AuthConfig`. Removed — now
  uses individual constructor parameters.

### Lean Docker Client
- Raw HTTP to Docker socket avoids dependency hell but lacks API version
  negotiation. Should pin API version or add capability negotiation.

### Non-Root User + Docker Socket
- Container runs as `portal` user but Docker socket requires root/docker
  group. Production deployment should use `docker-socket-proxy` sidecar.
