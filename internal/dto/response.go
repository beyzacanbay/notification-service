package dto

import "github.com/beyzacanbay/notification-service/internal/model"

type NotificationResponse struct {
	model.Notification
}

type BatchCreateResponse struct {
	BatchID       string                 `json:"batch_id"`
	TotalCreated  int                    `json:"total_created"`
	Notifications []NotificationResponse `json:"notifications"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalItems int64       `json:"total_items"`
	TotalPages int         `json:"total_pages"`
}

type ErrorResponse struct {
	Error   string      `json:"error"`
	Details interface{} `json:"details,omitempty"`
}

type HealthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services,omitempty"`
}
