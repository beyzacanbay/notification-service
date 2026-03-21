package delivery

import (
	"context"
	"time"

	"github.com/beyzacanbay/notification-service/internal/model"
)

type emailProvider struct {
	baseProvider
}

func NewEmailProvider(url string, timeout time.Duration) Provider {
	return &emailProvider{baseProvider: newBaseProvider(url, timeout)}
}

func (p *emailProvider) Channel() model.Channel {
	return model.ChannelEmail
}

func (p *emailProvider) Send(ctx context.Context, n *model.Notification) (*SendResult, error) {
	payload := map[string]string{
		"to":      n.Recipient,
		"channel": "email",
		"content": n.Content,
	}
	return p.doPost(ctx, payload)
}
