# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Implementation spec

Before writing any code, read `UN_tenant_ms_IMPL.md`. It defines the target architecture, exact struct names, service interfaces, SQL schema, RBAC rules, and business logic constraints. Names defined there (tables, columns, structs, routes) are definitive — do not rename them. Do not add external dependencies without confirming first.

## Commands

```bash
# Run the service
go run .

# Build binary
go build -o tenant .

# Run all tests
go test ./...

# Run a single test
go test ./... -run TestName

# Sync dependencies
go mod tidy
```

## Environment variables

| Variable | Default (dev only) | Required |
|---|---|---|
| `DATABASE_URL` | `host=localhost port=5432 user=postgres password=admin dbname=postgres sslmode=disable` | In prod |
| `JWT_SECRET` | `super-secreto-cambiar-en-produccion` | In prod |
| `INTERNAL_API_KEY` | — | Always (no fallback) |
| `PORT` | `8080` | No |

`INTERNAL_API_KEY` is the shared secret between this service and the orchestrator (`UN_chat_orch_ms`). The orchestrator sends it in the `X-Internal-Key` header when calling `/api/v1/internal/*` endpoints. Both services must have the same value configured. When `config.Load()` is implemented it must use `mustEnv()` so the service fails at startup if this var is missing.

The fallbacks exist only for local dev. In any deployed environment all vars must be set explicitly.

## Database setup

No migration tool is wired yet. Apply the schema manually before running:

```bash
psql $DATABASE_URL -f sql/init_schema.sql
```

## Architecture

The service is a Go + Gin HTTP API that manages tenants and their users for the UNagent platform.

**Entry point:** `main.go` → calls `config.InitDB()` then `SetupRouter()` (defined in `router.go`).

**Current package layout:**

```
config/         — database pool (sql.DB via lib/pq)
handlers/       — HTTP handler functions (auth_handler.go: login)
middlewares/    — JWT auth + RBAC role enforcement
models/         — request/response structs and JWT Claims
sql/            — raw SQL schema (init_schema.sql)
router.go       — all route definitions + inline tenant/user handlers
```

**Target layout (per UN_tenant_ms_IMPL.md, not yet implemented):**

```
config/         — config.Load() with mustEnv()
pkg/db/         — pgxpool connection
pkg/response/   — standard HTTP response helpers
pkg/middleware/ — request ID + structured JSON logger
internal/auth/  — login, JWT middleware, user CRUD service
internal/tenant/— tenant CRUD, channels, agent profiles, data sources, agent configs, executor
migrations/     — golang-migrate files (global/ and tenant/ schemas)
cmd/seed/       — bootstrap first app_admin
```

## DB schema (current)

Three tables in the `public` schema:

- **`tenants`** — `id, name, domain, is_active, created_at, updated_at`
- **`users`** — `id, email, password_hash, first_name, last_name, is_active, created_at, updated_at`
- **`user_tenants`** — junction table: `user_id`, `tenant_id` (nullable for app_admin), `role` (enum: `app_admin | tenant_admin | tenant_operator`)

A CHECK constraint on `user_tenants` enforces: `app_admin` → `tenant_id IS NULL`; other roles → `tenant_id IS NOT NULL`.

## RBAC model

Three roles encoded in the JWT and enforced by `middlewares.RoleMiddleware`:

- `app_admin` — no tenant, can do everything (`RoleMiddleware` always passes for this role)
- `tenant_admin` — scoped to one tenant
- `tenant_operator` — scoped to one tenant, read/operate only

`AuthMiddleware` validates the JWT and injects `user_claims` and `user_role` into the Gin context. Downstream handlers read role via `c.Get("user_role")` or the full claims via `c.Get("user_claims")`.

## Current routes

| Method | Path | Auth | Role |
|---|---|---|---|
| GET | `/health` | None | — |
| POST | `/auth/login` | None | — |
| GET | `/api/admin/tenants` | JWT | app_admin |
| POST | `/api/admin/tenants` | JWT | app_admin |
| POST | `/api/admin/users` | JWT | app_admin |
| GET | `/api/tenant/stats` | JWT | tenant_admin |
| GET | `/api/tenant/operations` | JWT | tenant_admin, tenant_operator |
