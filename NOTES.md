# 4rch3.io Portal — Development Notes

## Project Overview

The `4rch3.io` portal is a Go-based homelab operations hub providing authentication,
service discovery, reverse proxy, and health monitoring for the homelab ecosystem.

## Architecture

```
4rch3.io (Go binary)
├── cmd/server/          # Entry point
├── internal/
│   ├── auth/            # JWT + API key + session auth
│   ├── config/          # YAML configuration
│   ├── database/        # SQLite persistence
│   ├── discovery/       # Docker container auto-discovery
│   ├── handlers/        # HTTP handlers (dashboard, admin, auth)
│   ├── middleware/      # Auth middleware, LAN detection, logging
│   ├── models/          # Domain types
│   ├── monitor/         # Health check / uptime monitoring
│   └── proxy/           # Reverse proxy for service routing
├── web/                 # Web assets
├── cli/                 # CLI tools (future)
├── config.yaml          # Server configuration
├── Dockerfile
└── docker-compose.yml
```

## Key Design Decisions

### Auth Strategy (Combined)
- **Primary:** JWT-based sessions (HttpOnly cookies) for browser users
- **Service Auth:** API keys (`4rch3_` prefixed) for machine-to-machine
- **Future:** OIDC/OAuth2 provider integration
- **Roles:** admin, user, readonly, service
- **On-prem LAN:** Requests from local subnets can optionally bypass MFA

### Docker Service Discovery
Containers are auto-discovered via Docker socket using labels:
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

### Proxy Routing
Authenticated requests to `/svc/<service-name>` are reverse-proxied to the
discovered internal URL of that service. User identity is forwarded via headers.

### Monitoring
Periodic HTTP health checks on discovered services. Status is stored in SQLite
with a rolling history. Live status via HTMX on the dashboard.

## Deployment

Currently runs as a Docker container on the homelab:
```bash
docker compose up -d
```

Public access via Cloudflare Tunnel → Traefik → Portal.
LAN access via `homelab.local` → Traefik → Portal.

## Future Roadmap

- [ ] OIDC/OIDC provider integration (Keycloak / Authentik)
- [ ] Proxmox VM/LXC discovery via API
- [ ] WebSocket terminal for containers
- [ ] TOTP/WebAuthn MFA
- [ ] Service-specific permissions (IAM)
- [ ] Status page (public)
- [ ] CLI tool for management
- [ ] Audit log viewer in admin panel
- [ ] Tailscale integration for LAN-less access
- [ ] Metrics export (Prometheus)
