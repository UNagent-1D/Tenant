package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Tool struct {
	ID                uuid.UUID       `json:"id"`
	Name              string          `json:"name"`
	Description       *string         `json:"description,omitempty"`
	OpenAIFunctionDef json.RawMessage `json:"openai_function_def"`
	IsActive          bool            `json:"is_active"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}
