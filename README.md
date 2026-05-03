# Tenant Service — UNagent

Multi-tenant management microservice for the **UNagent** platform. Provides tenant provisioning, user management with RBAC, channel configuration, agent profiles, data source execution, and a tool registry — all scoped to isolated tenants.

## Architecture Overview

```
┌──────────┐   HTTP/REST   ┌──────────────┐  X-Internal-Key  ┌──────────────┐  pgxpool  ┌──────────┐
│          │ ─────────────▶│              │ ────────────────▶│              │──────────▶│          │
│ Frontend │               │ Orchestrator │                  │   Tenant     │           │PostgreSQL│
│  / Web   │               │ (UN_chat_    │                  │   Service    │           │          │
│   App    │◀──────────────│  orch_ms)    │◀─────────────────│  (Gin HTTP)  │◀──────────│          │
└──────────┘               └──────────────┘                  └──────────────┘           └──────────┘
```

The Frontend never communicates directly with this service. All requests go through the **Orchestrator** (`UN_chat_orch_ms`), which forwards instructions to the Tenant Service using the `X-Internal-Key` header for authentication. This makes the Tenant Service a purely internal component.

The service follows a **two-schema multi-tenant model**:

| Schema | Purpose | Tables |
|---|---|---|
| **global** | Shared across all tenants | `tenants`, `users`, `tool_registry` |
| **tenant** | One schema per tenant | `channels`, `agent_profiles`, `agent_configs`, `data_sources`, `end_users` |

Each tenant gets its own PostgreSQL schema, provisioned automatically on creation. This provides strong data isolation between organizations.

## RBAC Model

Three roles are enforced via JWT claims and middleware chains:

| Role | Scope | Capabilities |
|---|---|---|
| `app_admin` | Platform-wide | Full access to all tenants, users, tool registry |
| `tenant_admin` | Single tenant | Manage tenant resources, channels, profiles, configs, data sources, end users |
| `tenant_operator` | Single tenant | Read-only access to profiles, configs, end users; can execute operations |

The `app_admin` role always bypasses tenant-scoping middleware. All other roles are scoped to a single `tenant_id` enforced at the database query level.

## Authentication Flow

1. Client POSTs `/auth/login` with email + password
2. Service validates credentials and returns a JWT containing `sub`, `email`, `role`, and `tenant_id`
3. Subsequent requests include `Authorization: Bearer <token>`
4. `AuthMiddleware` validates the token; `RoleMiddleware` enforces role access
5. For tenant-scoped routes, `TenantScopeMiddleware` resolves and validates the tenant

Internal calls from the orchestrator use `X-Internal-Key` header matching `INTERNAL_API_KEY`.

## Environment Variables

| Variable | Default (dev only) | Required |
|---|---|---|
| `DATABASE_URL` | `postgres://postgres:admin@localhost:5432/postgres?sslmode=disable` | In prod |
| `JWT_SECRET` | `super-secreto-cambiar-en-produccion` | In prod |
| `INTERNAL_API_KEY` | — | Always |
| `PORT` | `8080` | No |

## Project Structure

```
├── cmd/                  — CLI entry points (seed, etc.)
├── config/               — Application configuration (Load, env vars)
├── internal/
│   ├── auth/             — Login handler, JWT middleware, user CRUD, role enforcement
│   └── tenant/           — Tenant CRUD, channels, profiles, configs, data sources, end users, executor
├── migrations/
│   ├── global/           — Schema shared across all tenants
│   └── tenant/           — Schema applied per-tenant on provisioning
├── pkg/
│   ├── db/               — pgxpool initialization and connection management
│   ├── middleware/       — Request ID injection, structured JSON logging
│   └── response/         — Standard HTTP response helpers
├── main.go               — Entry point: config → db → router
├── router.go             — All route definitions with middleware chains
└── docker-compose.yml    — Local dev infrastructure
```

## Quick Start

### Prerequisites

- Go 1.25+
- PostgreSQL 14+

### Run locally

```bash
# 1. Set required env vars
export INTERNAL_API_KEY=dev-key-change-in-prod
export SEED_EMAIL=admin@example.com
export SEED_PASSWORD=changeme123

# 2. Run migrations
migrate -path migrations/global -database "$DATABASE_URL" up
# Tenant migrations are applied automatically during tenant provisioning

# 3. Seed the first app_admin user (safe to re-run, skips if one exists)
go run cmd/seed/main.go

# 4. Start the service
go run .
```

### Run with Docker Compose

The compose stack includes four services: `db` (PostgreSQL 16), `migrate` (runs global migrations), `seed` (creates the first `app_admin`), and `app` (the service itself).

```bash
# Set seed credentials before starting (required on first run)
export SEED_EMAIL=admin@example.com
export SEED_PASSWORD=changeme123

docker compose up -d
```

## API Routes

### Public

| Method | Path | Description |
|---|---|---|
| POST | `/auth/login` | Authenticate and get JWT |
| GET | `/health` | Health check with DB ping |

### Authenticated (`/api/v1`, JWT required)

| Method | Path | Role | Description |
|---|---|---|---|
| GET | `/users` | app_admin | List users |
| POST | `/users` | app_admin, tenant_admin | Create user |
| PATCH | `/users/:uid` | app_admin, tenant_admin | Update user |
| GET | `/tenants` | app_admin | List all tenants |
| POST | `/tenants` | app_admin | Create + provision tenant |
| GET | `/tenants/:id` | tenant_admin | Get tenant details |
| PATCH | `/tenants/:id` | tenant_admin | Update tenant |

#### Tenant-scoped resources (`/tenants/:id/*`)

| Method | Path | Role | Description |
|---|---|---|---|
| GET | `/:id/channels` | tenant_admin | List channels |
| POST | `/:id/channels` | tenant_admin | Create channel |
| PATCH | `/:id/channels/:cid` | tenant_admin | Update channel |
| GET | `/:id/profiles` | tenant_admin, tenant_operator | List agent profiles |
| POST | `/:id/profiles` | tenant_admin | Create profile |
| PATCH | `/:id/profiles/:pid` | tenant_admin | Update profile |
| GET | `/:id/profiles/:pid/configs` | tenant_admin, tenant_operator | List agent configs |
| GET | `/:id/profiles/:pid/configs/active` | tenant_admin, tenant_operator | Get active config |
| POST | `/:id/profiles/:pid/configs` | tenant_admin | Create config |
| PATCH | `/:id/profiles/:pid/configs/:cid` | tenant_admin | Update config |
| POST | `/:id/profiles/:pid/configs/:cid/activate` | tenant_admin | Activate config |
| GET | `/:id/data-sources` | tenant_admin | List data sources |
| POST | `/:id/data-sources` | tenant_admin | Create data source |
| PATCH | `/:id/data-sources/:did` | tenant_admin | Update data source |
| POST | `/:id/end-users` | tenant_admin | Create end user |
| GET | `/:id/end-users/lookup/phone/:number` | tenant_admin, tenant_operator | Lookup by phone |
| GET | `/:id/end-users/lookup/national-id/:nid` | tenant_admin, tenant_operator | Lookup by national ID |

#### Tool Registry

| Method | Path | Role | Description |
|---|---|---|---|
| GET | `/tool-registry` | app_admin, tenant_admin | List tools |
| POST | `/tool-registry` | app_admin | Register tool |
| PATCH | `/tool-registry/:tid` | app_admin | Update tool |

### Internal (orchestrator only)

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/internal/:id/execute` | `X-Internal-Key` | Execute a data source for a tenant |

## Migrations

Run global migrations:

```bash
migrate -path migrations/global -database "$DATABASE_URL" up
```

Tenant schema migrations are applied automatically by the provisioner when a new tenant is created. The provisioner:

1. Creates a dedicated PostgreSQL schema `tenant_<id>`
2. Runs all `migrations/tenant/` migrations against that schema
3. Returns the provisioned tenant details

## Testing

```bash
# Run all tests
go test ./...

# Run a single test
go test ./... -run TestName
```

## Version

Current: 2.2.0
