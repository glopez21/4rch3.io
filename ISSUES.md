# 4rch3.io Portal — Known Issues & Conflicts

## Current Issues

### 27. WebSocket Proxy Broken — ResponseWriter Wrappers Don't Implement http.Hijacker
- **Status:** Fixed (2026-05-27)
- **Severity:** Critical
- **Description:** `metrics.responseWriter` and `middleware.responseWriter` wrapped
  `http.ResponseWriter` without implementing `http.Hijacker`. When a WebSocket
  upgrade request hit the proxy, `w.(http.Hijacker)` always returned `false`,
  causing all WebSocket connections to fail with "hijacking not supported".
- **Fix:** Both `responseWriter` types now implement `http.Hijacker` (and
  `http.Flusher`) by delegating to the underlying `ResponseWriter`.
- **Reference:** `internal/metrics/metrics.go`, `internal/middleware/middleware.go`

### 28. WebSocket Scheme Logic Inverted — Always ws://, Never wss://
- **Status:** Fixed (2026-05-27)
- **Severity:** Critical
- **Description:** In `handleWebSocket`, line `target.Scheme = "ws"` was executed
  before `if target.Scheme == "https"`, meaning the check always evaluated to
  false. All WebSocket connections used `ws://` even for HTTPS backends.
  Additionally, mutating `target.Scheme` modified the shared `url.URL` struct
  returned by `resolveTarget`, poisoning future requests to the same service.
- **Fix:** Compute `wsScheme` from the original `target.Scheme` into a local
  variable. Set `r.Host` and `r.URL.Scheme` on the upgrade request so the
  backend receives correct headers.
- **Reference:** `internal/proxy/proxy.go:101-106`

### 29. LAN Bypass Auth Exploit — realIP() Inverted Proxy Logic + Blind X-Real-IP Trust
- **Status:** Fixed (2026-05-27)
- **Severity:** Critical
- **Description:** `realIP()` checked if the **first entry** in `X-Forwarded-For`
  was a trusted proxy and returned that proxy IP (not the client IP). It also
  blindly trusted `X-Real-IP` with no proxy verification. Since `LANDetect`
  uses `realIP()` to grant unauthenticated proxy access (`proxy.go:43`), an
  attacker could set `X-Real-IP: 192.168.1.1` or a spoofed `X-Forwarded-For`
  to bypass auth entirely for all proxied services.
- **Fix:** Rewrote `realIP()` to only trust `X-Forwarded-For` and `X-Real-IP`
  when `r.RemoteAddr` is from a trusted proxy. Walks XFF right-to-left to find
  the first non-trusted IP. Falls back to `RemoteAddr` for untrusted connections.
- **Reference:** `internal/middleware/middleware.go:67-118`

### 30. X-Forwarded-For Not Stripped Before Appending
- **Status:** Fixed (2026-05-27)
- **Severity:** Medium
- **Description:** `setForwardingHeaders` deleted `X-Forwarded-User`/`X-Forwarded-Role`
  (good) but used `Add` for `X-Forwarded-For`, appending to any client-injected
  values instead of replacing them. This let clients inject arbitrary IPs into
  the `X-Forwarded-For` chain forwarded to backend services.
- **Fix:** `Del("X-Forwarded-For")` before `Set("X-Forwarded-For", r.RemoteAddr)`.
- **Reference:** `internal/proxy/proxy.go:91-99`

### 31. Disabled Users Bypass Auth via Session/JWT
- **Status:** Fixed (2026-05-27)
- **Severity:** High
- **Description:** The `Authenticate` middleware's session path and JWT path loaded
  the user from DB but did not check `user.Active`. Only the API key path correctly
  checked `AND active = 1`. A disabled user retained full access until session/JWT expiry.
- **Fix:** Added `if !user.Active` check after loading user in both session and JWT
  paths. Session tokens are deleted for inactive users. JWT requests simply proceed
  without user context (treated as unauthenticated).
- **Reference:** `internal/middleware/middleware.go:Authenticate()`

### 32. Unauthenticated Webhook Endpoint
- **Status:** Fixed (2026-05-27)
- **Severity:** High
- **Description:** `/api/webhook/deploy` had no authentication, rate limiting, or CSRF
  protection. Anyone who could reach the server could inject arbitrary deploy events
  into the audit log and dashboard.
- **Fix:** Applied `RequireAuth` + API rate limiter to both webhook routes.
  `/api/webhook/deploy` and `/api/deploys` now require authenticated sessions.
- **Reference:** `cmd/server/main.go:184-186`

### 33. JSON Login Bypasses CSRF Validation
- **Status:** Fixed (2026-05-27)
- **Severity:** Medium
- **Description:** Login requests with `Content-Type: application/json` skipped CSRF
  token validation entirely. Cross-origin JSON login requests could bypass CSRF
  protection.
- **Fix:** JSON login requests must now include an `X-CSRF-Token` header, validated
  via the same HMAC-SHA256 mechanism used for form-based login.
- **Reference:** `internal/handlers/auth.go:50-75`

### 34. Admin DELETE Uses Query Params with No CSRF
- **Status:** Fixed (2026-05-27)
- **Severity:** Medium
- **Description:** Admin POST (create user) and DELETE (delete user) had no CSRF
  token validation. Raw SQLite errors were also leaked via `err.Error()`.
- **Fix:** Admin POST now requires `csrf_token` in the JSON body. Admin DELETE now
  requires `X-CSRF-Token` header. Both validated via HMAC-SHA256. Error messages
  are now generic; original errors logged server-side. Admin dashboard template
  includes CSRF token in `<meta name="csrf-token">` tag. Added `/api/auth/csrf`
  endpoint for JSON clients to obtain tokens.
- **Reference:** `internal/handlers/admin.go:110-153`

### 35. JSON Injection in writeJSONError
- **Status:** Open
- **Severity:** Low
- **Description:** `writeJSONError` in middleware.go builds JSON via string
  concatenation: `` `{"error":"` + msg + `"}` ``. If `msg` ever contains
  `"`, `\`, or other JSON-special characters, the output is malformed JSON.
  Currently only called with simple strings, but fragile for future changes.
- **Fix:** Use `json.Marshal` like `writeErrorJSON` in handlers/errors.go.
- **Reference:** `internal/middleware/middleware.go:107-111`

### 36. Rate Limiter Goroutine Leak
- **Status:** Open
- **Severity:** Low
- **Description:** `NewRateLimiter` starts a cleanup goroutine via `go rl.cleanup()`
  with no cancel mechanism. Two rate limiters are created in main.go, so two
  goroutines leak permanently and never terminate on graceful shutdown.
- **Reference:** `internal/middleware/ratelimit.go:25`

### 37. Rate Limiter Cleanup Uses Unreliable Float Equality
- **Status:** Open
- **Severity:** Low
- **Description:** Cleanup evicts visitors when `limiter.Tokens() == float64(rl.burst)`.
  Due to floating-point behavior, `Tokens()` may asymptotically approach `burst`
  but never exactly equal it, causing the visitors map to grow unbounded.
- **Reference:** `internal/middleware/ratelimit.go:66`

### 38. UseInviteCode Returns (true, error) on Commit Failure
- **Status:** Open
- **Severity:** Low
- **Description:** If `tx.Commit()` fails, `UseInviteCode` returns `(true, err)`.
  The caller receives `true` (success) alongside a non-nil error. If the commit
  failed, the use count was NOT incremented.
- **Reference:** `internal/database/db.go:473`

### 39. Health Checker Ignores Request Context
- **Status:** Open
- **Severity:** Medium
- **Description:** `CheckService` receives `ctx context.Context` but uses
  `m.client.Get(target)` without it. Health check HTTP requests cannot be
  cancelled during server shutdown (up to 30+ seconds per check with retries).
- **Fix:** Use `http.NewRequestWithContext(ctx, "GET", target, nil)`.
- **Reference:** `internal/monitor/health.go:99`

### 40. Handler getIP Bypasses Trusted Proxy Validation
- **Status:** Open
- **Severity:** Medium
- **Description:** `getIP()` in auth.go returns the raw `X-Forwarded-For` header
  (which may contain multiple IPs) with no validation. This IP is stored as
  `last_ip` and in audit logs. Attackers can inject any IP string.
- **Fix:** Use `middleware.realIP()` or the same trusted-proxy logic.
- **Reference:** `internal/handlers/auth.go:202-206`

### 41. Admin Errors Leak SQLite Schema Details
- **Status:** Fixed (2026-05-27)
- **Severity:** Low
- **Description:** `CreateUser` and `DeleteUser` in admin handler returned raw
  `err.Error()` which could expose SQLite constraint names like
  `UNIQUE constraint failed: users.username`.
- **Fix:** Replaced with generic error messages (`"failed to create user"`,
  `"failed to delete user"`). Original errors logged server-side via `slog.Error`.
- **Reference:** `internal/handlers/admin.go:127-153`

### 42. Template Execution Errors Silently Ignored
- **Status:** Open
- **Severity:** Low
- **Description:** All `template.Execute()` calls in dashboard.go and admin.go
  ignore the returned error. Failed templates send partial or malformed HTML
  with no error status code.
- **Reference:** `internal/handlers/dashboard.go:42,53,64`,
  `internal/handlers/admin.go:89`

### 43. Session Cookie Not Persisting Over HTTP (Secure Flag)
- **Status:** Fixed (2026-05-27)
- **Severity:** High
- **Description:** Login set `Secure=true` on the session cookie even when
  accessing via plain HTTP (localhost:8081). Browsers silently refuse to send
  `Secure` cookies over HTTP, causing the session to be lost after login redirect.
  The old logic only checked `r.TLS` and `X-Forwarded-Proto`, both of which are
  absent on direct HTTP access. The logout cookie had the same issue.
- **Fix:** When `Secure` would be `false` from TLS/proxy checks, additionally
  check if the remote address is a loopback or private IP; if so, force
  `Secure=false`. Applied to both login and logout cookie setters.
- **Reference:** `internal/handlers/auth.go:129-137, 168-177`

### 44. SameSite=Strict Blocks Login Redirect
- **Status:** Fixed (2026-05-27)
- **Severity:** High
- **Description:** Session cookie was set with `SameSite=StrictMode`. After a
  successful form POST login, the server redirects to `/` via 303 or HTMX
  `HX-Redirect`. `SameSite=Strict` prevents the browser from sending the cookie
  on this top-level navigation, so the user appears unauthenticated and gets
  bounced back to the login page indefinitely. This was previously changed from
  Lax to Strict for CSRF hardening (issue #19), but Strict is too restrictive
  for the login redirect flow.
- **Fix:** Changed to `SameSite=LaxMode`. Lax still blocks CSRF on POST/PUT/DELETE
  but allows cookies on top-level GET navigations (which is what the redirect is).
- **Reference:** `internal/handlers/auth.go:144`

### 45. Auth Rate Limiter Burst Too Aggressive
- **Status:** Fixed (2026-05-27)
- **Severity:** Low
- **Description:** Auth rate limiter had burst=1 with rate=10/min. After a single
  login attempt, the bucket was empty and subsequent attempts (including legitimate
  retries) were rate-limited for ~6 seconds. Combined with the cookie bugs above,
  users who failed once would get locked out while trying to debug.
- **Fix:** Increased burst from 1 to 3, allowing short bursts of login attempts
  without throttling legitimate users.
- **Reference:** `cmd/server/main.go:94`

### 46. Debug Logging Needed for Auth Flow
- **Status:** In Progress
- **Severity:** Low
- **Description:** Added `slog.Debug` statements to `Authenticate` middleware to
  trace token lookup, session validation, user active checks, and JWT validation.
  Login handler also logs successful logins with cookie settings. These are
  gated behind `logging.level: "debug"` in config.
- **Reference:** `internal/middleware/middleware.go:Authenticate()`,
  `internal/handlers/auth.go:136`

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

### 47. Health Monitor Fails on HTTPS/Self-Signed Backends
- **Status:** Fixed (2026-08-15)
- **Severity:** Medium
- **Description:** The health monitor's `http.Client` had no
  `TLSClientConfig`, so any backend serving HTTPS with a self-signed cert
  (e.g. the Wazuh dashboard) always failed with `Get "...": EOF` and was
  reported offline even when healthy.
- **Fix:** Added `TLSClientConfig: &tls.Config{InsecureSkipVerify: true}`
  to the monitor client. Also defaulted to an `https://` internal URL for
  TLS backends.
- **Reference:** `internal/monitor/health.go`

### 48. Service Discovery Doesn't Refresh Label-Driven Fields
- **Status:** Fixed (2026-08-15)
- **Severity:** Medium
- **Description:** The `CreateService` upsert only refreshed
  `display_name/url/internal_url/icon/docker_id/docker_name` on
  `ON CONFLICT`. Label changes to `category`, `group_name`, or
  `health_path` (e.g. setting `4rch3.io.health=/app/login`) were never
  picked up on re-discovery, so services kept a stale health path and
  stayed offline. The Wazuh row had to be deleted manually to re-sync.
- **Fix:** Added `category`, `group_name`, and `health_path` to the
  `DO UPDATE SET` clause so rediscovery refreshes all label-driven fields.
- **Reference:** `internal/database/db.go:CreateService()`

### 49. Wazuh Single-Node All-in-One Deployment (v4.14.7)
- **Status:** Deployed (2026-08-15)
- **Severity:** N/A (deployment)
- **Description:** Deployed Wazuh indexer + manager + dashboard
  (single-node, tag `v4.14.7` from `wazuh-docker`) on clu5t3r for
  learning. Prereqs: `vm.max_map_count` (already 1048576), certs generated
  via `generate-indexer-certs.yml` with `wazuh/wazuh-certs-generator:0.0.4`
  (CERT_TOOL_VERSION=4.14), ownership fixed to uid 1000 / mode 400.
- **Fix / config:**
  - Deploy dir `/home/x5pyd3rx/wazuh` (compose + `generate-indexer-certs.yml`
    + `config/`).
  - Dashboard wired to `proxy-net` + Traefik `wazuh.4rch3.io`; removed host
    `443:5601` bind (conflicts with Traefik).
  - Backend is HTTPS/self-signed → Traefik uses `insecure-backend@file`
    serversTransport + `server.scheme=https` (see homelab
    `configs/traefik/dynamic/transports.yml`).
  - Portal labels: `4rch3.io.url=https://wazuh.4rch3.io`,
    `internal_url=https://wazuh-wazuh.dashboard-1:5601`,
    `health=/app/login` (dashboard's `/health` returns 401).
  - Default creds: dashboard `admin`/`SecretPassword`,
    `kibanaserver`/`kibanaserver`, API `wazuh-wui`/`MyS3cr37P450r.*-`.
    **TODO:** change defaults.
  - Ports exposed on host: manager 1514/1515/514udp/55000, indexer 9200.
- **Issue:** after a failed `--force-recreate` (443 bind conflict) the
  dashboard container came up with an **empty** `NetworkSettings.Networks`
  map and `getaddrinfo EBUSY wazuh.indexer`; a clean `docker compose up -d`
  restored networking.
- **Reference:** `/home/x5pyd3rx/wazuh/docker-compose.yml` (clu5t3r)

### 50. Splunk Enterprise (Free/500MB) Deployment
- **Status:** Deployed (2026-08-15)
- **Severity:** N/A (deployment)
- **Description:** Deployed Splunk Enterprise via `splunk/splunk:latest`
  on clu5t3r for learning, single standalone node.
- **Fix / config:**
  - Deploy dir `/home/x5pyd3rx/splunk` (`docker-compose.yml`).
  - Env: `SPLUNK_LICENSE_URI=Free`,
    `SPLUNK_PASSWORD=${SPLUNK_PASSWORD:-Chang3Me_N0w!}`,
    `SPLUNK_ROLE=splunk_standalone`,
    `SPLUNK_GENERAL_TERMS=--accept-sgt-current-at-splunk-com` (required by
    current images, else container exits 1 refusing to start without
    `--accept-license`).
  - Web UI is plain **HTTP on 8000** in current images (no
    `enableSplunkWebSSL`); Traefik labels use default HTTP scheme + port
    8000. Management 8089, forwarder 9997, HEC 8088 exposed.
  - Removed host `514/udp` bind (owned by Wazuh manager).
  - Portal: `internal_url=http://splunk:8000`,
    `health=/en-US/account/login`, `url=https://splunk.4rch3.io`.
  - Login: `admin` / the `SPLUNK_PASSWORD` value. **TODO:** set a real
    password.
- **Reference:** `/home/x5pyd3rx/splunk/docker-compose.yml` (clu5t3r)

### 51. Disk Full — threatpulse-agent SQLite WAL Runaway (166GB)
- **Status:** Resolved (2026-08-15)
- **Severity:** Critical
- **Description:** clu5t3r disk hit 100% (343G/344G). `threatpulse-agent`
  was stuck retrying failed batch POSTs to `siem-app:8000` (siem-app 500s
  because its Postgres was in recovery mode). Each retry appended to a
  SQLite WAL that grew unboundedly: `threat-pulse_agent-queue` volume
  reached **161G** (`queue.db-wal` = 156G) and was still growing. This also
  blocked Splunk from booting (No space left on device).
- **Fix:** Stopped `threatpulse-agent`, killed stray sqlite/python check
  containers that held a lock on the DB, then `truncate -s 0` the WAL and
  removed the `-shm` file. Volume dropped 161G → 6G (disk 100% → 58%,
  ~142G free). Pending uncheckpointed retry data was dropped (acceptable:
  the agent was failing to deliver anyway). `threatpulse-agent` left
  stopped until siem-app's DB recovers.
- **Reference:** `/var/lib/docker/volumes/threat-pulse_agent-queue/_data/`
  (clu5t3r)

### 52. Traefik Stale Backend After Container Recreate
- **Status:** Resolved (2026-08-15)
- **Severity:** Medium
- **Description:** After recreating the Wazuh dashboard container, the
  Traefik route for `wazuh.4rch3.io` opened the TLS/HTTP2 stream but never
  received a response (curl hung until timeout) even though the backend was
  directly reachable on `proxy-net`. Splunk on the same proxy-net worked
  fine.
- **Fix:** `docker restart infrastructure-traefik-1` cleared the stale
  backend state; route returned 200 immediately. (No code change.)
- **Reference:** `infrastructure-traefik-1` (clu5t3r)

### 53. Traefik serversTransport Needed for Self-Signed HTTPS Backends
- **Status:** Resolved (2026-08-15)
- **Severity:** Medium
- **Description:** Traefik could not proxy to backends serving HTTPS with
  self-signed certs (Wazuh dashboard) without explicit skip-verify.
- **Fix:** Added `configs/traefik/dynamic/transports.yml` defining
  `http.serversTransports.insecure-backend` with `insecureSkipVerify: true`.
  Wazuh labels reference it via
  `traefik.http.services.wazuh.loadbalancer.serversTransport=insecure-backend@file`
  + `server.scheme=https`.
- **Reference:** homelab repo `configs/traefik/dynamic/transports.yml`

### 54. Portal Sync Overwrote Newer clu5t3r Code (deploy feature lost)
- **Status:** Open / Deferred — recovery pending
- **Severity:** Critical
- **Date:** 2026-08-16
- **Description:** The clu5t3r working copy of `4rch3.io` was **ahead** of the
  local git repo: it contained a `deploy` feature that was never synced back
  to git. Local-only files on clu5t3r: `internal/deploy/deploy.go` and
  `internal/handlers/stats.go`. Deploy support existed in `config.go`
  (`DeployTarget`/`DeployConfig`), `dashboard.go` (`dockerSocket` field on
  `DashboardHandler`), `webhook.go`/`main.go` (deploy runner wiring),
  `config.yaml` (`deploy:` block), `.env` (`PORTAL_DEPLOY_SECRET`), and
  `docker-compose.yml` (extra augur/threat-pulse/threatwirenews bind mounts).
- **Incident:** While adding the SVG favicon, an `rsync -az` from local →
  clu5t3r overwrote clu5t3r's newer `config.go`, `dashboard.go`, `main.go`,
  `webhook.go`, `config.yaml`, `.env`, `docker-compose.yml`, `NOTES.md`,
  `PLAN.md` with older local versions. The extra files survived, so the
  container build now fails:
  `internal/deploy/deploy.go:16: undefined: config.DeployTarget` and
  `internal/handlers/stats.go:57: h.dockerSocket undefined`.
- **What was recovered intact:**
  - `internal/deploy/deploy.go` and `internal/handlers/stats.go` (still on
    clu5t3r, unclobbered).
  - `config.yaml` deploy block recovered from the running container image
    (`docker exec 4rch3-portal cat /app/config.yaml`): targets = augur,
    threat-pulse, threatwirenews with their compose files; timeout 5m;
    `secret: "${PORTAL_DEPLOY_SECRET}"`.
  - `PORTAL_DEPLOY_SECRET=bd9041d0b3ca6a5bf07faea88c0ec4bf71e99a5dd7d4868a`
    and `PORTAL_JWT_SECRET` from running container env (`docker inspect`).
- **What is NOT recoverable:** the original newer `config.go`/`dashboard.go`/
  `webhook.go`/`main.go`/`docker-compose.yml` source — the BuildKit build
  cache was pruned (content store only 752K, no source blobs), and the image
  final stage contains only the compiled binary + `config.yaml` + `schema.sql`.
- **Current state:** `4rch3-portal` still running and healthy on the
  pre-incident image (`4rch3io-portal:latest`, built 2026-08-15 16:12Z), so
  the site is up. The favicon changes are committed locally (`0b13819`) and
  synced, but the container could NOT be rebuilt (build fails on the missing
  deploy symbols above). Do not rebuild until recovery steps are done.
- **Recovery plan (deferred):**
  1. Rebuild `DeployTarget`/`DeployConfig` structs in `internal/config/config.go`
     to match `deploy.go` usage + recovered YAML block.
  2. Add `dockerSocket` field + `NewDashboardHandler` wiring in
     `internal/handlers/dashboard.go`; keep `stats.go` `ServiceStats` handler.
  3. Wire `deploy.Runner` into the deploy webhook in `webhook.go`/`main.go`
     (confirm handler/route names from the compiled binary's exported symbols
     first, since the original is gone).
  4. Restore `docker-compose.yml` (add `PORTAL_DEPLOY_SECRET` + 3 bind mounts)
     and `.env` (`PORTAL_DEPLOY_SECRET`).
  5. Rebuild the image and verify the deploy webhook works end-to-end.
- **Lessons learned:** Always check for clu5t3r-only files / diff before
  `rsync`-ing local → clu5t3r; treat clu5t3r as a possible newer source;
  keep the local git repo as the single source of truth and commit deploy
  feature back to git once reconstructed.
- **Reference:** `/home/x5pyd3rx/4rch3.io/` (clu5t3r), image
  `4rch3io-portal:latest`, local repo `/home/w01f/projects/4rch3.io`.

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
