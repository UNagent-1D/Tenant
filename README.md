# Tenant Service

Microservicio REST que gestiona tenants, usuarios, canales, perfiles de agente, fuentes de datos, usuarios finales y el Agent Config Registry (ACR). Expone la API central del sistema UNagent-1D.

## Stack

- **Go 1.25** + **Gin** (HTTP framework)
- **PostgreSQL 16** con aislamiento schema-per-tenant (`tenant_<slug>`)
- **pgx v5 / pgxpool** (driver nativo, connection pooling)
- **JWT HS256** para autenticación

## Estructura del proyecto

```
Tenant/
├── config/
│   └── database.go          # Inicialización de pgxpool
├── handlers/
│   ├── auth_handler.go       # POST /auth/login
│   ├── tenant_handler.go     # CRUD tenants
│   ├── user_handler.go       # CRUD usuarios del sistema
│   ├── channel_handler.go    # CRUD canales
│   ├── profile_handler.go    # CRUD perfiles de agente
│   ├── data_source_handler.go
│   ├── end_user_handler.go   # Lookup de usuarios finales
│   ├── acr_handler.go        # Agent Config Registry
│   └── tool_registry_handler.go
├── middlewares/
│   ├── auth_middleware.go    # Validación JWT
│   ├── rbac_middleware.go    # Validación de rol
│   ├── tenant_scope.go       # Scoping cross-tenant
│   └── request_id.go        # X-Request-ID + structured logs
├── models/                   # Structs de request/response
├── sql/
│   └── init_schema.sql       # Schema global + provision_tenant_schema()
├── router.go
├── main.go
├── api_test.go
└── Dockerfile
```

## Diseño de base de datos

### Schema global
| Tabla | Descripción |
|---|---|
| `tenants` | Registro de tenants (slug, plan, status, branding) |
| `users` | Usuarios del sistema (app_admin, tenant_admin, tenant_operator) |
| `tool_registry` | Catálogo global de herramientas para agentes |

### Schema por tenant (`tenant_<slug>`)
Creado automáticamente con `provision_tenant_schema(slug)` al crear un tenant.

| Tabla | Descripción |
|---|---|
| `channels` | Canales de comunicación (whatsapp, webchat, etc.) |
| `agent_profiles` | Perfiles de agente con config activa |
| `agent_configs` | Versiones de configuración LLM (ACR) |
| `data_sources` | Fuentes de datos conectadas al agente |
| `end_users` | Usuarios finales del tenant |

## Variables de entorno

| Variable | Descripción | Default |
|---|---|---|
| `DATABASE_URL` | Connection string de PostgreSQL | — (requerida) |
| `JWT_SECRET` | Secreto para firmar/verificar JWT | — (requerida) |
| `SERVER_PORT` | Puerto del servidor HTTP | `8080` |

Ejemplo `.env`:
```env
DATABASE_URL=postgres://postgres:password@localhost:5432/tenantdb?sslmode=disable
JWT_SECRET=supersecret
SERVER_PORT=8080
```

## Levantar con Docker Compose

Desde la raíz del repositorio (donde vive `docker-compose.yml`):

```bash
docker compose up --build tenant
```

El servicio queda disponible en `http://localhost:8080`.

Para inicializar el schema la primera vez:

```bash
docker compose exec db psql -U postgres -d tenantdb -f /docker-entrypoint-initdb.d/init_schema.sql
```

## Levantar localmente

```bash
# 1. Dependencias
go mod download

# 2. Variables de entorno
cp .env.example .env   # editar con tus valores

# 3. Correr
go run .
```

## Endpoints

Base path: `/api/v1` (excepto `/auth/login` y `/health`)

### Auth & Health
| Método | Path | Rol requerido |
|---|---|---|
| `POST` | `/auth/login` | Público |
| `GET` | `/health` | Público |

### Usuarios del sistema
| Método | Path | Rol requerido |
|---|---|---|
| `GET` | `/api/v1/users` | app_admin |
| `POST` | `/api/v1/users` | app_admin, tenant_admin |
| `PATCH` | `/api/v1/users/:uid` | app_admin, tenant_admin |

### Tenants
| Método | Path | Rol requerido |
|---|---|---|
| `GET` | `/api/v1/tenants` | app_admin |
| `POST` | `/api/v1/tenants` | app_admin |
| `GET` | `/api/v1/tenants/:id` | tenant_admin (propio) |
| `PATCH` | `/api/v1/tenants/:id` | tenant_admin (propio) |

### Canales
| Método | Path | Rol requerido |
|---|---|---|
| `GET` | `/api/v1/tenants/:id/channels` | tenant_admin |
| `POST` | `/api/v1/tenants/:id/channels` | tenant_admin |
| `PATCH` | `/api/v1/tenants/:id/channels/:cid` | tenant_admin |

### Perfiles de agente
| Método | Path | Rol requerido |
|---|---|---|
| `GET` | `/api/v1/tenants/:id/profiles` | tenant_admin, tenant_operator |
| `POST` | `/api/v1/tenants/:id/profiles` | tenant_admin |
| `PATCH` | `/api/v1/tenants/:id/profiles/:pid` | tenant_admin |

### Agent Config Registry (ACR)
| Método | Path | Rol requerido |
|---|---|---|
| `GET` | `/api/v1/tenants/:id/profiles/:pid/configs` | tenant_admin, tenant_operator |
| `GET` | `/api/v1/tenants/:id/profiles/:pid/configs/active` | tenant_admin, tenant_operator |
| `POST` | `/api/v1/tenants/:id/profiles/:pid/configs` | tenant_admin |
| `PATCH` | `/api/v1/tenants/:id/profiles/:pid/configs/:cid` | tenant_admin |
| `POST` | `/api/v1/tenants/:id/profiles/:pid/configs/:cid/activate` | tenant_admin |

### Fuentes de datos
| Método | Path | Rol requerido |
|---|---|---|
| `GET` | `/api/v1/tenants/:id/data-sources` | tenant_admin |
| `POST` | `/api/v1/tenants/:id/data-sources` | tenant_admin |
| `PATCH` | `/api/v1/tenants/:id/data-sources/:did` | tenant_admin |

### Usuarios finales
| Método | Path | Rol requerido |
|---|---|---|
| `POST` | `/api/v1/tenants/:id/end-users` | tenant_admin |
| `GET` | `/api/v1/tenants/:id/end-users/lookup/phone/:number` | tenant_admin, tenant_operator |
| `GET` | `/api/v1/tenants/:id/end-users/lookup/national-id/:nid` | tenant_admin, tenant_operator |

### Tool Registry
| Método | Path | Rol requerido |
|---|---|---|
| `GET` | `/api/v1/tool-registry` | app_admin, tenant_admin |
| `POST` | `/api/v1/tool-registry` | app_admin |
| `PATCH` | `/api/v1/tool-registry/:tid` | app_admin |

## Roles y permisos

| Rol | Alcance |
|---|---|
| `app_admin` | Acceso total — puede operar sobre cualquier tenant |
| `tenant_admin` | Acceso completo a su propio tenant |
| `tenant_operator` | Solo lectura sobre recursos de su tenant |

El middleware `TenantScopeMiddleware` valida que `tenant_admin` y `tenant_operator` solo accedan a su propio tenant (cross-tenant → 403).

## Tests

```bash
# Solo tests sin DB (middlewares, validación, RBAC)
go test ./... -run TestAuth
go test ./... -run TestRBAC
go test ./... -run TestValidation

# Tests de integración (requiere PostgreSQL)
TEST_DATABASE_URL=postgres://postgres:password@localhost:5432/tenantdb?sslmode=disable go test ./... -v
```
