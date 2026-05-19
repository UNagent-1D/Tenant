# UN_tenant_ms — HTTP Response Reference
### Referencia completa de request/response por endpoint — v2.2

> Este documento lista todos los endpoints con sus requests, responses exitosos
> y todos los posibles errores. Usar como referencia para implementación y testing.

---

## Convenciones generales

**Todos los errores** retornan este formato:
```json
{
  "error": "mensaje legible del error",
  "request_id": "req_7f3a9b12"
}
```
- `request_id` solo se incluye en errores `500`.
- Errores de validación (`400`) incluyen el nombre del campo que falló cuando aplica.

**Headers comunes en todas las respuestas:**
```
Content-Type: application/json
X-Request-ID: req_7f3a9b12
```

---

## 1. POST /api/v1/auth/login

**Auth:** Público

**Request:**
```json
{
  "email": "admin@hospital-san-ignacio.com",
  "password": "supersecret123"
}
```

**200 OK — Login exitoso:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2026-03-05T18:00:00Z",
  "user": {
    "id": "a1b2c3d4-0000-0000-0000-000000000001",
    "email": "admin@hospital-san-ignacio.com",
    "role": "tenant_admin",
    "tenant_id": "f1e2d3c4-0000-0000-0000-000000000010",
    "is_active": true,
    "created_at": "2026-03-04T09:00:00Z"
  }
}
```

**400 Bad Request — Campos faltantes:**
```json
{ "error": "email is required" }
```
```json
{ "error": "password is required" }
```

**401 Unauthorized — Credenciales inválidas:**
```json
{ "error": "invalid credentials" }
```
> Siempre el mismo mensaje. No revelar si el email no existe o si la contraseña es incorrecta.

**403 Forbidden — Cuenta desactivada:**
```json
{ "error": "account is deactivated" }
```

---

## 2. GET /api/v1/auth/me

**Auth:** JWT (cualquier rol)

**200 OK:**
```json
{
  "id": "a1b2c3d4-0000-0000-0000-000000000001",
  "email": "admin@hospital-san-ignacio.com",
  "role": "tenant_admin",
  "tenant_id": "f1e2d3c4-0000-0000-0000-000000000010",
  "is_active": true,
  "created_at": "2026-03-04T09:00:00Z"
}
```

**401 Unauthorized — JWT ausente o inválido:**
```json
{ "error": "missing or invalid authorization token" }
```

**401 Unauthorized — JWT expirado:**
```json
{ "error": "token has expired" }
```

---

## 3. PATCH /api/v1/auth/me/password

**Auth:** JWT (cualquier rol)

**Request:**
```json
{
  "current_password": "supersecret123",
  "new_password": "NewSecurePass!456"
}
```

**200 OK:**
```json
{ "message": "password updated successfully" }
```

**400 Bad Request — Contraseña actual incorrecta:**
```json
{ "error": "current password is incorrect" }
```

**400 Bad Request — Nueva contraseña muy corta:**
```json
{ "error": "new_password must be at least 8 characters" }
```

**400 Bad Request — Campos faltantes:**
```json
{ "error": "current_password is required" }
```
```json
{ "error": "new_password is required" }
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

---

## 4. GET /api/v1/users

**Auth:** JWT (app_admin, tenant_admin)

**200 OK — app_admin (ve todos los usuarios):**
```json
{
  "data": [
    {
      "id": "a1b2c3d4-0000-0000-0000-000000000001",
      "email": "superadmin@platform.com",
      "role": "app_admin",
      "tenant_id": null,
      "is_active": true,
      "created_at": "2026-03-04T09:00:00Z"
    },
    {
      "id": "a1b2c3d4-0000-0000-0000-000000000002",
      "email": "admin@hospital-san-ignacio.com",
      "role": "tenant_admin",
      "tenant_id": "f1e2d3c4-0000-0000-0000-000000000010",
      "is_active": true,
      "created_at": "2026-03-04T09:15:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 2
  }
}
```

**200 OK — tenant_admin (solo ve usuarios de su tenant):**
```json
{
  "data": [
    {
      "id": "a1b2c3d4-0000-0000-0000-000000000002",
      "email": "admin@hospital-san-ignacio.com",
      "role": "tenant_admin",
      "tenant_id": "f1e2d3c4-0000-0000-0000-000000000010",
      "is_active": true,
      "created_at": "2026-03-04T09:15:00Z"
    },
    {
      "id": "a1b2c3d4-0000-0000-0000-000000000003",
      "email": "operator1@hospital-san-ignacio.com",
      "role": "tenant_operator",
      "tenant_id": "f1e2d3c4-0000-0000-0000-000000000010",
      "is_active": true,
      "created_at": "2026-03-04T09:20:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 2
  }
}
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden — tenant_operator intentando listar:**
```json
{ "error": "insufficient permissions" }
```

---

## 5. POST /api/v1/users

**Auth:** JWT (app_admin, tenant_admin)

**Request — app_admin creando tenant_admin:**
```json
{
  "email": "admin@hospital-nuevo.com",
  "password": "SecurePass!123",
  "role": "tenant_admin",
  "tenant_id": "f1e2d3c4-0000-0000-0000-000000000020"
}
```

**Request — tenant_admin creando tenant_operator:**
```json
{
  "email": "operator2@hospital-san-ignacio.com",
  "password": "SecurePass!123",
  "role": "tenant_operator"
}
```
> `tenant_id` se ignora — se usa el del caller automáticamente.

**201 Created:**
```json
{
  "id": "a1b2c3d4-0000-0000-0000-000000000099",
  "email": "operator2@hospital-san-ignacio.com",
  "role": "tenant_operator",
  "tenant_id": "f1e2d3c4-0000-0000-0000-000000000010",
  "is_active": true,
  "created_at": "2026-03-04T10:00:00Z"
}
```

**400 Bad Request — Campos faltantes o inválidos:**
```json
{ "error": "email is required" }
```
```json
{ "error": "password must be at least 8 characters" }
```
```json
{ "error": "role must be one of: app_admin, tenant_admin, tenant_operator" }
```

**400 Bad Request — tenant_id requerido para roles no-admin:**
```json
{ "error": "tenant_id is required for tenant_admin and tenant_operator roles" }
```

**400 Bad Request — tenant_id no existe:**
```json
{ "error": "tenant_id references a tenant that does not exist" }
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden — tenant_admin intentando crear app_admin:**
```json
{ "error": "tenant_admin cannot create app_admin users" }
```

**403 Forbidden — tenant_admin intentando crear tenant_admin:**
```json
{ "error": "tenant_admin can only create tenant_operator users" }
```

**403 Forbidden — tenant_operator intentando crear:**
```json
{ "error": "insufficient permissions" }
```

**409 Conflict — Email duplicado:**
```json
{ "error": "a user with this email already exists" }
```

---

## 6. GET /api/v1/users/:uid

**Auth:** JWT (app_admin, tenant_admin)

**200 OK:**
```json
{
  "id": "a1b2c3d4-0000-0000-0000-000000000003",
  "email": "operator1@hospital-san-ignacio.com",
  "role": "tenant_operator",
  "tenant_id": "f1e2d3c4-0000-0000-0000-000000000010",
  "is_active": true,
  "created_at": "2026-03-04T09:20:00Z"
}
```

**400 Bad Request — ID con formato inválido:**
```json
{ "error": "invalid user id format" }
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden — tenant_admin viendo usuario de otro tenant:**
```json
{ "error": "access denied to this user" }
```

**403 Forbidden — tenant_operator:**
```json
{ "error": "insufficient permissions" }
```

**404 Not Found:**
```json
{ "error": "user not found" }
```

---

## 7. PATCH /api/v1/users/:uid

**Auth:** JWT (app_admin, tenant_admin)

**Request — Desactivar usuario:**
```json
{
  "is_active": false
}
```

**Request — Cambiar rol:**
```json
{
  "role": "tenant_operator"
}
```

**200 OK:**
```json
{
  "id": "a1b2c3d4-0000-0000-0000-000000000003",
  "email": "operator1@hospital-san-ignacio.com",
  "role": "tenant_operator",
  "tenant_id": "f1e2d3c4-0000-0000-0000-000000000010",
  "is_active": false,
  "created_at": "2026-03-04T09:20:00Z"
}
```

**400 Bad Request — Rol inválido:**
```json
{ "error": "role must be one of: app_admin, tenant_admin, tenant_operator" }
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden — tenant_admin modificando usuario de otro tenant:**
```json
{ "error": "access denied to this user" }
```

**403 Forbidden — tenant_admin promoviendo a app_admin:**
```json
{ "error": "tenant_admin cannot assign app_admin role" }
```

**403 Forbidden — tenant_operator:**
```json
{ "error": "insufficient permissions" }
```

**404 Not Found:**
```json
{ "error": "user not found" }
```

---

## 8. DELETE /api/v1/users/:uid

**Auth:** JWT (app_admin)

**204 No Content** — Sin body.

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden — No es app_admin:**
```json
{ "error": "only app_admin can delete users" }
```

**403 Forbidden — Intentando borrarse a sí mismo:**
```json
{ "error": "cannot delete yourself" }
```

**404 Not Found:**
```json
{ "error": "user not found" }
```

**409 Conflict — Usuario activo:**
```json
{ "error": "user must be inactive before deletion" }
```
> Un usuario debe ser desactivado primero (`PATCH /api/v1/users/:uid` con `"is_active": false`) antes de poder eliminarse.

---

## 9. GET /api/v1/tenants

**Auth:** JWT (app_admin)

**Query params opcionales:** `?page=1&limit=20`

**200 OK:**
```json
{
  "data": [
    {
      "id": "f1e2d3c4-0000-0000-0000-000000000010",
      "slug": "hospital-san-ignacio",
      "name": "Hospital San Ignacio",
      "plan": "pro",
      "status": "active",
      "branding_logo_url": "https://cdn.example.com/logo.png",
      "branding_primary_color": "#1A73E8",
      "created_at": "2026-03-01T00:00:00Z",
      "updated_at": "2026-03-04T12:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 1
  }
}
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden — No es app_admin:**
```json
{ "error": "only app_admin can list all tenants" }
```

---

## 10. POST /api/v1/tenants

**Auth:** JWT (app_admin)

**Request:**
```json
{
  "slug": "hospital-san-ignacio",
  "name": "Hospital San Ignacio",
  "plan": "pro",
  "branding_logo_url": "https://cdn.example.com/logo.png",
  "branding_primary_color": "#1A73E8"
}
```

**201 Created:**
```json
{
  "id": "f1e2d3c4-0000-0000-0000-000000000010",
  "slug": "hospital-san-ignacio",
  "name": "Hospital San Ignacio",
  "plan": "pro",
  "status": "active",
  "branding_logo_url": "https://cdn.example.com/logo.png",
  "branding_primary_color": "#1A73E8",
  "created_at": "2026-03-01T00:00:00Z",
  "updated_at": "2026-03-01T00:00:00Z"
}
```

**400 Bad Request:**
```json
{ "error": "slug is required" }
```
```json
{ "error": "name is required" }
```
```json
{ "error": "plan must be one of: free, starter, pro, enterprise" }
```
```json
{ "error": "slug can only contain lowercase letters, numbers, and hyphens" }
```
```json
{ "error": "slug is reserved and cannot be used" }
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "only app_admin can create tenants" }
```

**409 Conflict:**
```json
{ "error": "a tenant with slug 'hospital-san-ignacio' already exists" }
```

---

## 11. GET /api/v1/tenants/:id

**Auth:** JWT (app_admin, tenant_admin)

**200 OK:**
```json
{
  "id": "f1e2d3c4-0000-0000-0000-000000000010",
  "slug": "hospital-san-ignacio",
  "name": "Hospital San Ignacio",
  "plan": "pro",
  "status": "active",
  "branding_logo_url": "https://cdn.example.com/logo.png",
  "branding_primary_color": "#1A73E8",
  "created_at": "2026-03-01T00:00:00Z",
  "updated_at": "2026-03-04T12:00:00Z"
}
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden — tenant_admin viendo otro tenant:**
```json
{ "error": "access denied to this tenant" }
```

**404 Not Found:**
```json
{ "error": "tenant not found" }
```

---

## 12. PATCH /api/v1/tenants/:id

**Auth:** JWT (app_admin, tenant_admin)

**Request — tenant_admin (solo campos permitidos):**
```json
{
  "name": "Hospital San Ignacio - Sede Norte",
  "branding_primary_color": "#FF5722"
}
```

**Request — app_admin (cualquier campo):**
```json
{
  "status": "suspended",
  "plan": "enterprise"
}
```

**200 OK:**
```json
{
  "id": "f1e2d3c4-0000-0000-0000-000000000010",
  "slug": "hospital-san-ignacio",
  "name": "Hospital San Ignacio - Sede Norte",
  "plan": "enterprise",
  "status": "suspended",
  "branding_logo_url": "https://cdn.example.com/logo.png",
  "branding_primary_color": "#FF5722",
  "created_at": "2026-03-01T00:00:00Z",
  "updated_at": "2026-03-05T14:30:00Z"
}
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden — tenant_admin intentando cambiar status:**
```json
{ "error": "only app_admin can change status" }
```

**403 Forbidden — tenant_admin intentando cambiar plan:**
```json
{ "error": "only app_admin can change plan" }
```

**403 Forbidden — tenant_admin de otro tenant:**
```json
{ "error": "access denied to this tenant" }
```

**404 Not Found:**
```json
{ "error": "tenant not found" }
```

---

## 13. GET /api/v1/tenants/:id/channels

**Auth:** JWT (tenant_admin)

**200 OK:**
```json
{
  "data": [
    {
      "id": "c1d2e3f4-0000-0000-0000-000000000001",
      "tenant_id": "f1e2d3c4-0000-0000-0000-000000000010",
      "channel_type": "whatsapp",
      "channel_key": "+573001234567",
      "is_active": true,
      "created_at": "2026-03-02T10:00:00Z",
      "updated_at": "2026-03-02T10:00:00Z"
    },
    {
      "id": "c1d2e3f4-0000-0000-0000-000000000002",
      "tenant_id": "f1e2d3c4-0000-0000-0000-000000000010",
      "channel_type": "web_widget",
      "channel_key": "widget-hospital-san-ignacio",
      "is_active": true,
      "created_at": "2026-03-02T10:05:00Z",
      "updated_at": "2026-03-02T10:05:00Z"
    }
  ]
}
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden — tenant de otro tenant o rol insuficiente:**
```json
{ "error": "access denied to this tenant" }
```

**404 Not Found:**
```json
{ "error": "tenant not found" }
```

---

## 14. POST /api/v1/tenants/:id/channels

**Auth:** JWT (tenant_admin)

**Request:**
```json
{
  "channel_type": "whatsapp",
  "channel_key": "+573001234567",
  "webhook_secret_ref": "whatsapp_secret_ref_001"
}
```

**201 Created:**
```json
{
  "id": "c1d2e3f4-0000-0000-0000-000000000001",
  "tenant_id": "f1e2d3c4-0000-0000-0000-000000000010",
  "channel_type": "whatsapp",
  "channel_key": "+573001234567",
  "webhook_secret_ref": "whatsapp_secret_ref_001",
  "is_active": true,
  "created_at": "2026-03-02T10:00:00Z",
  "updated_at": "2026-03-02T10:00:00Z"
}
```

**400 Bad Request:**
```json
{ "error": "channel_type must be one of: whatsapp, web_widget" }
```
```json
{ "error": "channel_key is required" }
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "access denied to this tenant" }
```

**404 Not Found:**
```json
{ "error": "tenant not found" }
```

**409 Conflict:**
```json
{ "error": "a channel with key '+573001234567' already exists for this tenant" }
```

---

## 15. PATCH /api/v1/tenants/:id/channels/:cid

**Auth:** JWT (tenant_admin)

**Request:**
```json
{
  "is_active": false
}
```

**200 OK:**
```json
{
  "id": "c1d2e3f4-0000-0000-0000-000000000001",
  "tenant_id": "f1e2d3c4-0000-0000-0000-000000000010",
  "channel_type": "whatsapp",
  "channel_key": "+573001234567",
  "is_active": false,
  "created_at": "2026-03-02T10:00:00Z",
  "updated_at": "2026-03-05T16:00:00Z"
}
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "access denied to this tenant" }
```

**404 Not Found:**
```json
{ "error": "channel not found" }
```

---

## 16. GET /api/v1/tenants/:id/profiles

**Auth:** JWT (tenant_admin, tenant_operator)

**200 OK:**
```json
{
  "data": [
    {
      "id": "p1a2b3c4-0000-0000-0000-000000000001",
      "name": "Agendamiento General",
      "description": "Bot de agendamiento para todas las especialidades",
      "scheduling_flow_rules": {
        "require_document": true,
        "max_advance_days": 30
      },
      "escalation_rules": {
        "triggers": ["no_availability", "patient_frustrated"],
        "target": "human_agent"
      },
      "allowed_specialties": ["cardiologia", "medicina_general", "pediatria"],
      "allowed_locations": ["sede-norte", "sede-centro"],
      "agent_config_id": "ac001-0000-0000-0000-000000000001",
      "created_at": "2026-03-03T08:00:00Z",
      "updated_at": "2026-03-03T08:00:00Z"
    }
  ]
}
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "access denied to this tenant" }
```

**404 Not Found:**
```json
{ "error": "tenant not found" }
```

---

## 17. POST /api/v1/tenants/:id/profiles

**Auth:** JWT (tenant_admin)

**Request:**
```json
{
  "name": "Agendamiento General",
  "description": "Bot de agendamiento para todas las especialidades",
  "scheduling_flow_rules": {
    "require_document": true,
    "max_advance_days": 30
  },
  "escalation_rules": {
    "triggers": ["no_availability", "patient_frustrated"],
    "target": "human_agent"
  },
  "allowed_specialties": ["cardiologia", "medicina_general"],
  "allowed_locations": ["sede-norte"]
}
```

**201 Created:**
```json
{
  "id": "p1a2b3c4-0000-0000-0000-000000000001",
  "name": "Agendamiento General",
  "description": "Bot de agendamiento para todas las especialidades",
  "scheduling_flow_rules": {
    "require_document": true,
    "max_advance_days": 30
  },
  "escalation_rules": {
    "triggers": ["no_availability", "patient_frustrated"],
    "target": "human_agent"
  },
  "allowed_specialties": ["cardiologia", "medicina_general"],
  "allowed_locations": ["sede-norte"],
  "agent_config_id": null,
  "created_at": "2026-03-03T08:00:00Z",
  "updated_at": "2026-03-03T08:00:00Z"
}
```

**400 Bad Request:**
```json
{ "error": "name is required" }
```
```json
{ "error": "allowed_specialties must have at least one entry" }
```
```json
{ "error": "allowed_locations must have at least one entry" }
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "access denied to this tenant" }
```
```json
{ "error": "insufficient permissions" }
```

**404 Not Found:**
```json
{ "error": "tenant not found" }
```

---

## 18. PATCH /api/v1/tenants/:id/profiles/:pid

**Auth:** JWT (tenant_admin)

**Request:**
```json
{
  "allowed_specialties": ["cardiologia", "medicina_general", "pediatria", "neurologia"],
  "escalation_rules": {
    "triggers": ["no_availability", "patient_frustrated", "emergency_keywords"],
    "target": "human_agent"
  }
}
```

**200 OK:**
```json
{
  "id": "p1a2b3c4-0000-0000-0000-000000000001",
  "name": "Agendamiento General",
  "description": "Bot de agendamiento para todas las especialidades",
  "scheduling_flow_rules": {
    "require_document": true,
    "max_advance_days": 30
  },
  "escalation_rules": {
    "triggers": ["no_availability", "patient_frustrated", "emergency_keywords"],
    "target": "human_agent"
  },
  "allowed_specialties": ["cardiologia", "medicina_general", "pediatria", "neurologia"],
  "allowed_locations": ["sede-norte"],
  "agent_config_id": "ac001-0000-0000-0000-000000000001",
  "created_at": "2026-03-03T08:00:00Z",
  "updated_at": "2026-03-05T14:00:00Z"
}
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "access denied to this tenant" }
```
```json
{ "error": "insufficient permissions" }
```

**404 Not Found:**
```json
{ "error": "profile not found" }
```

---

## 19. GET /api/v1/tenants/:id/data-sources

**Auth:** JWT (tenant_admin)

**200 OK:**
```json
{
  "data": [
    {
      "id": "ds001-0000-0000-0000-000000000001",
      "name": "API Hospital San Ignacio",
      "source_type": "scheduling",
      "base_url": "https://mock-hospital-api.internal",
      "route_configs": {
        "list_doctors": { "method": "GET", "path": "/doctors" },
        "get_doctor_schedule": { "method": "GET", "path": "/doctors/{id}/schedule" },
        "create_appointment": { "method": "POST", "path": "/appointments" },
        "cancel_appointment": { "method": "DELETE", "path": "/appointments/{id}" }
      },
      "is_active": true,
      "created_at": "2026-03-03T09:00:00Z",
      "updated_at": "2026-03-03T09:00:00Z"
    }
  ]
}
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "access denied to this tenant" }
```

**404 Not Found:**
```json
{ "error": "tenant not found" }
```

---

## 20. POST /api/v1/tenants/:id/data-sources

**Auth:** JWT (tenant_admin)

**Request:**
```json
{
  "name": "API Hospital San Ignacio",
  "source_type": "scheduling",
  "base_url": "https://mock-hospital-api.internal",
  "credential_ref": "hospital-api-key-001",
  "route_configs": {
    "list_doctors": { "method": "GET", "path": "/doctors" },
    "get_doctor_schedule": { "method": "GET", "path": "/doctors/{id}/schedule" },
    "create_appointment": { "method": "POST", "path": "/appointments" }
  }
}
```

**201 Created:**
```json
{
  "id": "ds001-0000-0000-0000-000000000001",
  "name": "API Hospital San Ignacio",
  "source_type": "scheduling",
  "base_url": "https://mock-hospital-api.internal",
  "route_configs": {
    "list_doctors": { "method": "GET", "path": "/doctors" },
    "get_doctor_schedule": { "method": "GET", "path": "/doctors/{id}/schedule" },
    "create_appointment": { "method": "POST", "path": "/appointments" }
  },
  "is_active": true,
  "created_at": "2026-03-03T09:00:00Z",
  "updated_at": "2026-03-03T09:00:00Z"
}
```

**400 Bad Request:**
```json
{ "error": "name is required" }
```
```json
{ "error": "source_type must be one of: scheduling, patient_registry" }
```
```json
{ "error": "base_url must be a valid URL" }
```
```json
{ "error": "route_configs must have at least one operation" }
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "access denied to this tenant" }
```

**404 Not Found:**
```json
{ "error": "tenant not found" }
```

---

## 21. PATCH /api/v1/tenants/:id/data-sources/:did

**Auth:** JWT (tenant_admin)

**Request:**
```json
{
  "route_configs": {
    "list_doctors": { "method": "GET", "path": "/doctors" },
    "get_doctor_schedule": { "method": "GET", "path": "/doctors/{id}/schedule" },
    "create_appointment": { "method": "POST", "path": "/appointments" },
    "cancel_appointment": { "method": "DELETE", "path": "/appointments/{id}" }
  }
}
```

**200 OK:**
```json
{
  "id": "ds001-0000-0000-0000-000000000001",
  "name": "API Hospital San Ignacio",
  "source_type": "scheduling",
  "base_url": "https://mock-hospital-api.internal",
  "route_configs": {
    "list_doctors": { "method": "GET", "path": "/doctors" },
    "get_doctor_schedule": { "method": "GET", "path": "/doctors/{id}/schedule" },
    "create_appointment": { "method": "POST", "path": "/appointments" },
    "cancel_appointment": { "method": "DELETE", "path": "/appointments/{id}" }
  },
  "is_active": true,
  "created_at": "2026-03-03T09:00:00Z",
  "updated_at": "2026-03-05T15:00:00Z"
}
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "access denied to this tenant" }
```

**404 Not Found:**
```json
{ "error": "data source not found" }
```

---

## 22. GET /api/v1/tenants/:id/profiles/:pid/configs

**Auth:** JWT (tenant_admin, tenant_operator)

**200 OK:**
```json
{
  "data": [
    {
      "id": "ac001-0000-0000-0000-000000000001",
      "agent_profile_id": "p1a2b3c4-0000-0000-0000-000000000001",
      "version": 2,
      "status": "active",
      "conversation_policy": {
        "greeting": "Hola, soy el asistente de agendamiento del Hospital San Ignacio.",
        "max_turns": 20,
        "language": "es"
      },
      "escalation_rules": {
        "triggers": ["no_availability", "patient_frustrated"],
        "target": "human_agent"
      },
      "tool_permissions": [
        { "tool_name": "list_doctors", "constraints": {} },
        { "tool_name": "get_doctor_schedule", "constraints": {} },
        { "tool_name": "create_appointment", "constraints": { "require_confirmation": true } }
      ],
      "llm_params": {
        "model": "gpt-4o",
        "temperature": 0.3,
        "max_tokens": 1024,
        "system_prompt": "Eres un asistente de agendamiento médico..."
      },
      "channel_format_rules": {
        "whatsapp": { "max_chars": 4096 },
        "web_widget": { "max_chars": 8192 }
      },
      "created_by": "a1b2c3d4-0000-0000-0000-000000000002",
      "created_at": "2026-03-04T11:00:00Z",
      "activated_at": "2026-03-04T12:00:00Z"
    },
    {
      "id": "ac001-0000-0000-0000-000000000002",
      "agent_profile_id": "p1a2b3c4-0000-0000-0000-000000000001",
      "version": 1,
      "status": "archived",
      "conversation_policy": { "greeting": "Hola, bienvenido.", "max_turns": 15, "language": "es" },
      "escalation_rules": { "triggers": ["no_availability"], "target": "human_agent" },
      "tool_permissions": [{ "tool_name": "list_doctors", "constraints": {} }],
      "llm_params": { "model": "gpt-4o-mini", "temperature": 0.5, "max_tokens": 512, "system_prompt": "..." },
      "channel_format_rules": null,
      "created_by": "a1b2c3d4-0000-0000-0000-000000000002",
      "created_at": "2026-03-03T10:00:00Z",
      "activated_at": "2026-03-03T10:30:00Z"
    }
  ]
}
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "access denied to this tenant" }
```

**404 Not Found:**
```json
{ "error": "profile not found" }
```

---

## 23. GET /api/v1/tenants/:id/profiles/:pid/configs/active

**Auth:** JWT (tenant_admin, tenant_operator)

**200 OK:** Mismo formato que un elemento individual del listado anterior (el que tiene `status: "active"`).

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "access denied to this tenant" }
```

**404 Not Found — Ningún config activo:**
```json
{ "error": "no active config found for this profile" }
```

**404 Not Found — Profile no existe:**
```json
{ "error": "profile not found" }
```

---

## 24. POST /api/v1/tenants/:id/profiles/:pid/configs

**Auth:** JWT (tenant_admin)

**Request:**
```json
{
  "conversation_policy": {
    "greeting": "Hola, soy el asistente de agendamiento del Hospital San Ignacio.",
    "max_turns": 20,
    "language": "es"
  },
  "escalation_rules": {
    "triggers": ["no_availability", "patient_frustrated"],
    "target": "human_agent"
  },
  "tool_permissions": [
    { "tool_name": "list_doctors", "constraints": {} },
    { "tool_name": "get_doctor_schedule", "constraints": {} }
  ],
  "llm_params": {
    "model": "gpt-4o",
    "temperature": 0.3,
    "max_tokens": 1024,
    "system_prompt": "Eres un asistente de agendamiento médico..."
  },
  "channel_format_rules": {
    "whatsapp": { "max_chars": 4096 }
  }
}
```

**201 Created:** Mismo formato que un item del listado, con `status: "draft"`, `activated_at: null`, y `version` auto-incrementado.

**400 Bad Request — Validaciones de LLM params:**
```json
{ "error": "model 'gpt-3.5' is not in the allowed model list" }
```
```json
{ "error": "temperature must be between 0.0 and 2.0" }
```
```json
{ "error": "max_tokens must be between 1 and 4096" }
```

**400 Bad Request — Validaciones de tools:**
```json
{ "error": "tool 'nonexistent_tool' is not registered in the tool registry" }
```
```json
{ "error": "tool 'list_doctors' is currently deactivated globally" }
```

**400 Bad Request — Validaciones de escalation:**
```json
{ "error": "escalation_rules must have at least one trigger" }
```

**400 Bad Request — Campos faltantes:**
```json
{ "error": "conversation_policy is required" }
```
```json
{ "error": "tool_permissions must have at least one entry" }
```
```json
{ "error": "llm_params is required" }
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "access denied to this tenant" }
```
```json
{ "error": "insufficient permissions" }
```

**404 Not Found:**
```json
{ "error": "profile not found" }
```

---

## 25. PATCH /api/v1/tenants/:id/profiles/:pid/configs/:cid

**Auth:** JWT (tenant_admin)

**Request:** Misma estructura que el POST (campos parciales permitidos).

**200 OK:** Config actualizado completo.

**400 Bad Request:** Mismos errores de validación que el POST.

**400 Bad Request — Solo drafts son editables:**
```json
{ "error": "only draft configs can be updated" }
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "access denied to this tenant" }
```

**404 Not Found:**
```json
{ "error": "config not found" }
```

---

## 26. POST /api/v1/tenants/:id/profiles/:pid/configs/:cid/activate

**Auth:** JWT (tenant_admin)

**Request:** Sin body.

**200 OK:** Config con `status: "active"` y `activated_at` poblado.

**400 Bad Request:**
```json
{ "error": "only a draft config can be activated" }
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "access denied to this tenant" }
```
```json
{ "error": "insufficient permissions" }
```

**404 Not Found:**
```json
{ "error": "config not found" }
```

---

## 27. GET /api/v1/tool-registry

**Auth:** JWT (app_admin, tenant_admin)

**200 OK:**
```json
{
  "data": [
    {
      "id": "tr001-0000-0000-0000-000000000001",
      "name": "list_doctors",
      "description": "Lista doctores disponibles filtrados por especialidad y ubicación",
      "openai_function_def": {
        "name": "list_doctors",
        "description": "...",
        "parameters": {
          "type": "object",
          "properties": {
            "specialty": { "type": "string" },
            "location": { "type": "string" }
          }
        }
      },
      "is_active": true,
      "created_at": "2026-03-01T00:00:00Z",
      "updated_at": "2026-03-01T00:00:00Z"
    }
  ]
}
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

---

## 28. POST /api/v1/tool-registry

**Auth:** JWT (app_admin)

**Request:**
```json
{
  "name": "list_doctors",
  "description": "Lista doctores disponibles filtrados por especialidad y ubicación",
  "openai_function_def": {
    "name": "list_doctors",
    "description": "List available doctors by specialty and location",
    "parameters": {
      "type": "object",
      "properties": {
        "specialty": { "type": "string", "description": "Medical specialty" },
        "location": { "type": "string", "description": "Hospital location" }
      }
    }
  }
}
```

**201 Created:** Tool completo con id, is_active, timestamps.

**400 Bad Request:**
```json
{ "error": "name is required" }
```
```json
{ "error": "openai_function_def is required" }
```

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "only app_admin can register tools" }
```

**409 Conflict:**
```json
{ "error": "a tool with name 'list_doctors' already exists" }
```

---

## 29. PATCH /api/v1/tool-registry/:tid

**Auth:** JWT (app_admin)

**Request:**
```json
{
  "is_active": false
}
```

**200 OK:** Tool actualizado completo.

**401 Unauthorized:**
```json
{ "error": "missing or invalid authorization token" }
```

**403 Forbidden:**
```json
{ "error": "only app_admin can modify tools" }
```

**404 Not Found:**
```json
{ "error": "tool not found" }
```

---

## 30. POST /api/v1/internal/tenants/:id/data-sources/:did/execute

**Auth:** API key interna (`X-Internal-Key`)

**Request:**
```json
{
  "operation": "get_doctor_schedule",
  "path_params": {
    "id": "doctor-uuid-123"
  },
  "query_params": {
    "date": "2026-04-15"
  },
  "body": null
}
```

**200 OK — Sistema externo respondió exitosamente:**
```json
{
  "status_code": 200,
  "headers": {
    "content-type": "application/json"
  },
  "body": {
    "doctor_id": "doctor-uuid-123",
    "available_slots": [
      { "start": "2026-04-15T09:00:00Z", "end": "2026-04-15T09:30:00Z" },
      { "start": "2026-04-15T10:00:00Z", "end": "2026-04-15T10:30:00Z" }
    ]
  }
}
```

**200 OK — Sistema externo retornó error (el execute funcionó, el externo no):**
```json
{
  "status_code": 404,
  "headers": {
    "content-type": "application/json"
  },
  "body": {
    "error": "doctor not found"
  }
}
```

**200 OK — Operación POST (crear cita):**

Request:
```json
{
  "operation": "create_appointment",
  "path_params": {},
  "query_params": {},
  "body": {
    "doctor_id": "doctor-uuid-123",
    "patient_id": "patient-uuid-456",
    "date": "2026-04-15",
    "time": "09:00"
  }
}
```

Response:
```json
{
  "status_code": 201,
  "headers": {
    "content-type": "application/json"
  },
  "body": {
    "appointment_id": "appt-uuid-789",
    "status": "confirmed",
    "doctor_id": "doctor-uuid-123",
    "date": "2026-04-15T09:00:00Z"
  }
}
```

**400 Bad Request — Operación no registrada:**
```json
{ "error": "operation 'unknown_operation' not found in route_configs" }
```

**400 Bad Request — Faltan path params:**
```json
{ "error": "missing path_param: id" }
```

**401 Unauthorized — API key ausente:**
```json
{ "error": "missing X-Internal-Key header" }
```

**401 Unauthorized — API key inválida:**
```json
{ "error": "invalid internal API key" }
```

**404 Not Found — Tenant no existe:**
```json
{ "error": "tenant not found" }
```

**404 Not Found — Tenant no activo:**
```json
{ "error": "tenant is not active" }
```

**404 Not Found — Data source no existe:**
```json
{ "error": "data source not found" }
```

**502 Bad Gateway — Error de conexión al sistema externo:**
```json
{ "error": "failed to connect to external system: connection refused" }
```

**504 Gateway Timeout:**
```json
{ "error": "external system did not respond in time" }
```

---

## 31. GET /api/v1/health

**Auth:** Público

**200 OK — Todo bien:**
```json
{
  "status": "ok",
  "db": "ok",
  "version": "2.2.0",
  "timestamp": "2026-03-04T09:00:00Z"
}
```

**200 OK — DB caída (degradado pero responde):**
```json
{
  "status": "degraded",
  "db": "error: connection timeout",
  "version": "2.2.0",
  "timestamp": "2026-03-04T09:00:00Z"
}
```
