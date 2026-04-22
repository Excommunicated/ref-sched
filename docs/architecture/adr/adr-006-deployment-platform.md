# ADR-006: Deployment Platform

## Status: Accepted

## Context

The application must be deployed in a way that is:
- Accessible over HTTPS without VPN to all ~22 users.
- Maintainable by a single developer.
- Low or zero additional cost (developer has Azure free monthly credits).
- Consistent between local development and production (Docker containers required per the developer's profile).

The developer has Azure free monthly credits and a self-hosted Proxmox environment. The PRD explicitly states to prefer Azure for v1.

## Decision

**Azure Container Apps** for the Go backend and Caddy containers. **Azure Database for PostgreSQL Flexible Server** for the database.

### Component placement

| Component | Azure Service |
|-----------|--------------|
| Go backend | Azure Container Apps (single container, 0.5 vCPU / 1 GB RAM) |
| SvelteKit static files | Served by Caddy sidecar, or Azure Static Web Apps |
| PostgreSQL | Azure Database for PostgreSQL Flexible Server (Burstable, 1 vCore, 32 GB) |
| TLS termination | Azure Container Apps built-in HTTPS ingress, or Caddy with Let's Encrypt |

### Why Azure Container Apps over Azure App Service

Azure Container Apps:
- Native container deployment (push an image, it runs).
- Built-in HTTPS ingress with automatic TLS (no Caddy needed if using the built-in ingress).
- Scale-to-zero is available (not needed here, but reduces cost when idle).
- Simple revision management — deploy a new container image, it becomes the active revision.
- Free tier: 180,000 vCPU-seconds and 360,000 GiB-seconds per month — sufficient for 22 users at light load.

### Deployment workflow

1. Build Docker image locally or in CI: `docker build -t refsched-api:$(git rev-parse --short HEAD) .`
2. Push to Azure Container Registry (included in Azure free tier at basic tier).
3. Update the Container App to use the new image revision: `az containerapp update --name refsched-api --image refsched-api:SHA`.
4. Container App pulls the new image and starts it; the old revision is deactivated.

Database migrations run automatically at backend startup (via `golang-migrate`) before the HTTP server begins accepting traffic.

### Environment configuration

All secrets and configuration are injected as environment variables via Azure Container Apps secrets:
- `DATABASE_URL` — PostgreSQL connection string (SSL mode required).
- `GOOGLE_CLIENT_ID` — OAuth2 client ID.
- `GOOGLE_CLIENT_SECRET` — OAuth2 client secret.
- `SESSION_SECRET` — 32-byte random key used for PKCE state cookie signing (HMAC).
- `APP_BASE_URL` — Public base URL (used for OAuth2 redirect URI).
- `CORS_ALLOWED_ORIGINS` — Set to the frontend URL if frontend is on a different origin.

Locally, these are provided via a `.env` file loaded by `docker-compose`. The `.env` file is in `.gitignore`.

### Database

Azure Database for PostgreSQL Flexible Server (Burstable, 1 vCPU tier):
- Estimated cost: ~$15–20/month, covered by Azure free credits.
- Daily automated backups with 7-day retention.
- SSL enforcement on connections (required setting, enabled by default).
- Connection pooling via built-in PgBouncer (optional for this scale, but available).
- Public access with firewall rules restricting to the Container App outbound IP range. Alternatively, use Azure Virtual Network integration (Container Apps + PostgreSQL in the same VNet) for private connectivity — the VNet option is more secure and recommended if the developer is comfortable with the setup.

### Self-hosted Proxmox

The Proxmox environment is available as a fallback or for staging. Using Docker containers throughout means the same images that run locally and in Azure can also run on Proxmox with `docker compose up`. No code changes are needed to switch environments. For v1, Azure is the production target; Proxmox can serve as a local staging environment.

## Consequences

**Positive:**
- Docker containers work identically in local development (docker-compose) and Azure (Container Apps).
- Azure free credits cover all infrastructure costs in v1.
- Built-in HTTPS with Azure Container Apps managed certificates eliminates Let's Encrypt certificate management.
- No Kubernetes or orchestration knowledge required — Container Apps abstracts that.
- Automated database backups from day one.
- Simple deployment process (push image, update revision).

**Negative:**
- Azure Container Apps has a learning curve for initial setup (VNet, ingress, secrets, Container Registry). One-time cost.
- If Azure free credits run out, the cost is ~$20–35/month (Container App + PostgreSQL + Container Registry). Acceptable for a club app, but not free.
- Azure-specific tooling (`az` CLI). The developer must have the Azure CLI installed and configured. Alternatively, deployments can be done through the Azure portal UI.

## Alternatives Considered

**Self-hosted Proxmox (v1 target)**: Would use existing infrastructure with no cloud cost. Rejected for v1 because it requires the developer to manage networking, TLS, firewall, uptime, and backup — operational overhead that Azure managed services eliminate. Proxmox is explicitly noted as a fallback, not the preferred v1 target.

**Azure App Service (Web App for Containers)**: Also a valid choice. Container Apps has a simpler pricing model and better native container support. Both options are similar for this use case.

**Azure Kubernetes Service (AKS)**: Overkill for a single-container application. Kubernetes management complexity is not justified at this scale.

**Railway / Render / Fly.io**: Simple PaaS platforms that support Docker containers. No Azure credits are usable here. The developer has Azure credits — this is the correct choice for free hosting.

**Azure Static Web Apps (frontend only)**: Could host the SvelteKit static output separately. This is a viable option and reduces the frontend serving concern. The tradeoff is managing two separate deployments (Static Web App for frontend, Container App for backend) versus a single-container deployment that serves both. Either approach works — the static adapter makes the frontend deployment flexible.
