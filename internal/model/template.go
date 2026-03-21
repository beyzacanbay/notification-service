package model

import (
	"time"

	"github.com/google/uuid"
)

type Template struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Channel         Channel   `json:"channel"`
	ContentTemplate string    `json:"content_template"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
