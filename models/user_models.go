package models

// CreateUserRequest mapea los datos enviados para crear un nuevo usuario y asignarle un rol/tenant
type CreateUserRequest struct {
	Email     string  `json:"email" binding:"required,email"`
	Password  string  `json:"password" binding:"required,min=6"`
	FirstName string  `json:"first_name" binding:"required"`
	LastName  string  `json:"last_name" binding:"required"`
	Role      string  `json:"role" binding:"required,oneof=app_admin tenant_admin tenant_operator"`
	TenantID  *string `json:"tenant_id"` // Opcional, dependiendo del rol a crear
}
