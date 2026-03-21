package dto

import (
	"time"

	"github.com/beyzacanbay/notification-service/internal/model"
)

type CreateNotificationRequest struct {
	Channel   model.Channel   `json:"channel"`
	Recipient string          `json:"recipient"`
	Content   string          `json:"content"`
	Priority  *model.Priority `json:"priority,omitempty"`
}

type BatchCreateRequest struct {
	Notifications []CreateNotificationRequest `json:"notifications"`
}

type ListNotificationsRequest struct {
	Status    *model.Status  `json:"status"`
	Channel   *model.Channel `json:"channel"`
	StartDate *time.Time     `json:"start_date"`
	EndDate   *time.Time     `json:"end_date"`
	Page      int            `json:"page"`
	PerPage   int            `json:"per_page"`
}

func (r *ListNotificationsRequest) SetDefaults() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PerPage < 1 || r.PerPage > 100 {
		r.PerPage = 20
	}
}

func (r *ListNotificationsRequest) Offset() int {
	return (r.Page - 1) * r.PerPage
}
