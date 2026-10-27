# 4rch3.io Portal

A self-hosted homelab operations portal that auto-discovers Docker services via labels, monitors their health, and provides a unified dashboard for managing your infrastructure.

**Live site:** [https://4rch3.io](https://4rch3.io)

## Features

- **Auto-discovery** – Automatically detects Docker containers labeled with `4rch3.io.*` and registers them
- **Health monitoring** – Periodic health checks with status tracking and visual indicators
- **Reverse proxy integration** – Works with Traefik for secure ingress and routing
- **Authentication** – Session-based auth with JWT support, CSRF protection, and invite-only registration
- **Admin dashboard** – Manage users, services, and monitor homelab status
- **Service categorization** – Organize services by category (Core SOC, Integrations, Backend, etc.)
- **Public status page** – Optional public view of service health

## Tech Stack

- **Backend:** Go 1.25 with SQLite
- **Frontend:** HTML templates with HTMX
- **Infrastructure:** Docker, Traefik, systemd
- **Auth:** bcrypt, JWT, CSRF tokens

## Quick Start

### Prerequisites
- Docker and Docker Compose
- A reverse proxy (Traefik recommended) on `proxy-net` network
- Docker socket access for service discovery

### Setup

1. Clone the repo:
```bash
git clone https://github.com/glopez21/4rch3.io.git
cd 4rch3.io
```

2. Configure environment:
```bash
cp .env.example .env  # Create from template
# Set PORTAL_JWT_SECRET, admin credentials, etc.
```

3. Update `config.yaml` for your environment (trusted proxies, CORS origins, etc.)

4. Run with Docker Compose:
```bash
docker compose up -d
```

The portal will be available at `http://localhost:8081` by default.

## Configuration

Key configuration via `config.yaml` and environment variables:

| Setting | Description |
| --- | --- |
| `server.external_url` | Public URL of the portal |
| `discovery.docker_socket` | Docker socket for auto-discovery (default: `unix:///var/run/docker.sock`) |
| `discovery.label_prefix` | Label prefix for service discovery (default: `4rch3.io`) |
| `auth.invite_only` | Restrict registration to invite codes |
| `monitor.status_page_public` | Allow unauthenticated access to status page |
| `PORTAL_JWT_SECRET` | JWT signing secret (env var, required) |

## Service Discovery

Add labels to any Docker container to have it discovered:

```yaml
labels:
  - "4rch3.io.enable=true"
  - "4rch3.io.name=My Service"
  - "4rch3.io.description=Service description"
  - "4rch3.io.category=security"
  - "4rch3.io.url=https://service.example.com"
  - "4rch3.io.health=/api/health"
  - "4rch3.io.public=true"
```

## Security

- CSRF protection on state-changing operations
- Rate limiting on auth and API endpoints
- Trusted proxy configuration for correct client IP detection
- Secure headers via reverse proxy
- JWT with secure session management

## License

See repository for license information.
