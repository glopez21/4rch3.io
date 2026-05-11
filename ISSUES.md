# 4rch3.io Portal — Known Issues & Conflicts

## Current Issues

### 1. Dashboard Service Card Rendering
- **Status:** Open
- **Severity:** Low
- **Description:** The `serviceCard` function in `internal/handlers/dashboard.go`
  renders HTML-based cards. The initial page load shows empty grid until HTMX
  refreshes. Should consider server-side rendering the initial set.

### 2. SQLite Concurrent Write Limitations
- **Status:** Open
- **Severity:** Medium
- **Description:** SQLite WAL mode is enabled but only 1 connection is used
  (`SetMaxOpenConns(1)`). Under heavy monitoring (many services, short intervals),
  write contention could occur. Monitor for `database is locked` errors.
- **Potential fix:** Use write batching in the monitor or switch to Postgres.

### 3. Docker Socket Permission
- **Status:** Open
- **Severity:** High (deployment)
- **Description:** The container needs access to `/var/run/docker.sock` which
  requires either running as root or adding the container user to the `docker` group.
  The Dockerfile currently runs as root.
- **Fix:** Consider using `docker-socket-proxy` for safer socket access.

### 4. First-Run Bootstrap
- **Status:** Open
- **Severity:** Medium
- **Description:** No initial admin user is created on first run. Need to decide:
  env var seeding, CLI bootstrap command, or registration flow.
- **Plan:** Add a `--bootstrap-admin` flag or env-based initial user creation.

### 5. Session Token Storage
- **Status:** Open
- **Severity:** Low
- **Description:** Session tokens are stored in SQLite. For a small deployment
  this is fine, but session cleanup (expired sessions) is not yet implemented.
- **Fix:** Add a periodic session cleanup goroutine.

### 6. Missing Graceful Docker Connection Handling
- **Status:** Open
- **Severity:** Low
- **Description:** If the Docker socket is unavailable on startup, the service
  logs a warning and continues without discovery. However, it doesn't retry
  automatically.
- **Fix:** Add retry logic or a health-dependent restart.

## Resolved Issues

### R-1. Docker SDK Dependency Conflict
- **Resolved:** 2026-05-11
- **Description:** The official Docker Go SDK had complex dependency chains
  and platform-specific build failures (DialPipe on Linux). Replaced with a
  lightweight REST API client that talks directly to the Docker socket.

### R-2. Auth Service Self-Import
- **Resolved:** 2026-05-11
- **Description:** `internal/auth/service.go` had a self-import of the `auth`
  package. Fixed by removing the import and using local references.

### R-3. AuthConfig Missing SessionExpiry
- **Resolved:** 2026-05-11
- **Description:** `AuthConfig` in `auth.go` was missing the `SessionExpiry`
  field needed by the auth handler. Added the field.

### R-4. Dashboard Type Mismatch
- **Resolved:** 2026-05-11
- **Description:** `renderDashboard` was typed with `[]interface{}` instead of
  `[]*models.Service` and used a `serviceCard` with `interface{}` parameter.
  Fixed with proper types.

## Conflicts

### Auth Handler Config Duplication
- The auth handler takes an `auth.AuthConfig` which duplicates fields from the
  main `config.AuthConfig`. These should be unified or the handler should
  accept the config struct directly.
