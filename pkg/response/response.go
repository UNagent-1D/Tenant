package response

import "github.com/gin-gonic/gin"

type ErrorBody struct {
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

type PaginatedBody struct {
	Data       any        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

func OK(c *gin.Context, data any) {
	c.JSON(200, data)
}

func Created(c *gin.Context, data any) {
	c.JSON(201, data)
}

func NoContent(c *gin.Context) {
	c.Status(204)
}

func BadRequest(c *gin.Context, msg string) {
	c.JSON(400, ErrorBody{Error: msg, RequestID: getRequestID(c)})
}

func Unauthorized(c *gin.Context, msg string) {
	c.JSON(401, ErrorBody{Error: msg})
}

func Forbidden(c *gin.Context, msg string) {
	c.JSON(403, ErrorBody{Error: msg})
}

func NotFound(c *gin.Context, msg string) {
	c.JSON(404, ErrorBody{Error: msg})
}

func Conflict(c *gin.Context, msg string) {
	c.JSON(409, ErrorBody{Error: msg})
}

func InternalError(c *gin.Context) {
	c.JSON(500, ErrorBody{Error: "internal server error", RequestID: getRequestID(c)})
}

func BadGateway(c *gin.Context, msg string) {
	c.JSON(502, ErrorBody{Error: msg})
}

func GatewayTimeout(c *gin.Context) {
	c.JSON(504, ErrorBody{Error: "external system did not respond in time"})
}

func getRequestID(c *gin.Context) string {
	return c.GetString("request_id")
}
