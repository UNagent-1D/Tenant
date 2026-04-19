# Documentación de Integración de API (Frontend y Servicios Externos)

Este documento detalla la estructura y configuración de los endpoints expuestos por el microservicio de gestión de _Tenants_ (Inquilinos) y Control de Acceso Basado en Roles (RBAC). El propósito de esta guía es proveer a los equipos de frontend y servicios externos la información necesaria para integrarse y consumir las funcionalidades de manera segura y eficiente.

---

## 1. Consideraciones Generales

### Autenticación y Autorización (JWT)
La API utiliza JSON Web Tokens (JWT) para manejar las sesiones. Todos los endpoints ubicados bajo el prefijo `/api` están protegidos y requieren de la inclusión de un token válido en los headers de la petición:

```http
Authorization: Bearer <TOKEN_JWT>
```

### Roles y Nivel de Acceso (RBAC)
Los accesos están estructurados para verificar tres niveles de roles principales:
- `app_admin`: Administrador global del sistema (no está atado a un tenant en específico).
- `tenant_admin`: Administrador a nivel de tenant.
- `tenant_operator`: Operador estándar a nivel de tenant.

### Formato de Intercambio de Datos
Todas las peticiones para la creación o actualización de recursos deben usar formato JSON, configurando el header de la petición HTTP con:
`Content-Type: application/json`

---

## 2. Endpoints Públicos

### Iniciar Sesión (Login)
Permite a los usuarios autenticarse, devolviendo el JWT necesario para la navegación autenticada en el resto de la aplicación.

- **Ruta:** `POST /auth/login`
- **Acceso:** Público (No requiere token)
- **Body de Petición:**
  ```json
  {
    "email": "usuario@ejemplo.com",
    "password": "Mypassword123*"
  }
  ```
- **Respuestas Posibles:**
  - `200 OK`: Autenticación exitosa.
    ```json
    {
      "token": "eyJh... (JWT)",
      "message": "Bienvenido, autenticación exitosa"
    }
    ```
  - `400 Bad Request`: El payload no cumple con la estructura requerida (ej. email inválido).
  - `401 Unauthorized`: Credenciales inválidas.

---

## 3. Endpoints de Administración Global

Los siguientes recursos están estrictamente reservados para usuarios con el rol `app_admin`.

### Listar todos los Tenants
- **Ruta:** `GET /api/admin/tenants`
- **Acceso:** Requiere rol `app_admin`
- **Respuesta Esperada (`200 OK`):**
  ```json
  {
    "message": "Listado de todos los tenants del sistema global."
  }
  ```

### Crear un Nuevo Tenant
- **Ruta:** `POST /api/admin/tenants`
- **Acceso:** Requiere rol `app_admin`
- **Body de Petición:**
  ```json
  {
    "name": "Acme Corp",       // Requerido
    "domain": "acme.app.com"   // Opcional
  }
  ```
- **Respuestas Posibles:**
  - `201 Created`: Tenant creado correctamente, devuelve los detalles de la base de datos (incluyendo el `id`).
  - `400 Bad Request`: Información de payload faltante.
  - `500 Internal Server Error`: Problemas con la inserción en la base de datos.

### Eliminar un Tenant
- **Ruta:** `DELETE /api/admin/tenants/:id`
- **Acceso:** Requiere rol `app_admin`
- **Comportamiento:** Elimina de manera permanente el Tenant especificado por `:id`. Además, mediante una transacción segura, elimina a todos los usuarios que formaban parte de dicho Tenant para no dejar registros sin referencia en la base de datos.
- **Respuestas Posibles:**
  - `200 OK`: Tenant y sus usuarios eliminados exitosamente.
  - `404 Not Found`: Tenant no encontrado.
  - `500 Internal Server Error`: Error durante la eliminación en la base de datos.

---

## 4. Endpoints Nivel Tenant

Estas rutas están orientadas a la administración o visualización operativa que ocurre _dentro_ del entorno de un inquilino/Tenant.

### Consultar Estadísticas
- **Ruta:** `GET /api/tenant/stats`
- **Acceso:** Requiere rol `tenant_admin`
- **Respuesta Esperada (`200 OK`):**
  ```json
  {
    "message": "Estadísticas del tenant principal"
  }
  ```

### Consultar Operaciones
- **Ruta:** `GET /api/tenant/operations`
- **Acceso:** Requiere rol `tenant_admin` o `tenant_operator`
- **Respuesta Esperada (`200 OK`):**
  ```json
  {
    "message": "Panel de Operaciones de este Tenant"
  }
  ```

---

## 5. Endpoints de Gestión de Usuarios

Este módulo le permite tanto a los administradores globales como a los administradores de Tenant registrar nuevos usuarios. La API valida automáticamente qué nivel de usuario puede ser creado basándose en la autorización de quien realice la petición.

### Crear Usuario
- **Ruta:** `POST /api/users/`
- **Acceso:** Requiere rol `app_admin` o `tenant_admin`
- **Body de Petición:**
  ```json
  {
    "email": "nuevo.operador@ejemplo.com", 
    "password": "PasswordSegura!",
    "first_name": "Juan",
    "last_name": "Pérez",
    "role": "tenant_operator", 
    "tenant_id": "uuid-del-tenant" 
  }
  ```
  _Notas sobre el body_:
  * El `password` debe tener un mínimo de 6 caracteres.
  * Los roles soportados son: `app_admin`, `tenant_admin`, y `tenant_operator`.
  * **Casos de Uso del `tenant_id`**: 
    - Si un `app_admin` crea a un `tenant_admin` o `tenant_operator`, debe proveer obligatoriamente un `tenant_id`.
    - Si un `app_admin` crea a otro `app_admin`, el sistema ignora cualquier `tenant_id` y por defecto asignara *NULL*.
    - Si un `tenant_admin` efectúa la solicitud, el sistema solo le permitirá crear un `tenant_operator`, e intencionalmente asignará al nuevo usuario el mismo `tenant_id` de forma automática, abstrayendo esta validación al frontend.

- **Respuestas Posibles:**
  - `201 Created`: Usuario creado y rol asignado.
    ```json
    {
      "message": "Usuario creado y rol asignado exitosamente",
      "user_id": "uuid-del-nuevo-usuario"
    }
    ```
  - `403 Forbidden`: Un `tenant_admin` intentó crear un rol de nivel superior o fuera de su competencia.
  - `400 Bad Request`: Un `app_admin` no remitió un `tenant_id` al tratar de crear un usuario para de un inquilino.

### Eliminar Usuario
- **Ruta:** `DELETE /api/users/:id`
- **Acceso:** Requiere rol `app_admin` o `tenant_admin`
- **Comportamiento y Reglas de Negocio:**
  * Un usuario con rol `app_admin` puede eliminar cualquier cuenta en el sistema (por seguridad, se le restringe eliminar a otro `app_admin`).
  * Un usuario con rol `tenant_admin` solo puede eliminar cuentas con rol `tenant_operator` que pertenezcan de forma obligatoria a su mismo `tenant_id`.
- **Respuestas Posibles:**
  - `200 OK`: Usuario eliminado exitosamente.
  - `403 Forbidden`: Intento de eliminación no autorizado ya sea por jurisdicción de inquilino o por jerarquía de rol.
  - `404 Not Found`: El usuario a eliminar o evaluar no existe.
  - `500 Internal Server Error`: Problemas procesando la baja del usuario.
