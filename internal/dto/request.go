package dto

import "github.com/beyzacanbay/notification-service/internal/model"

type CreateNotificationRequest struct {
	Channel   model.Channel  `json:"channel"`
	Recipient string         `json:"recipient"`
	Content   string         `json:"content"`
	Priority  *model.Priority `json:"priority,omitempty"`
}
