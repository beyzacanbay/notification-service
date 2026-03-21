package delivery

import (
	"context"
	"time"

	"github.com/beyzacanbay/notification-service/internal/model"
)

type pushProvider struct {
	baseProvider
}

func NewPushProvider(url string, timeout time.Duration) Provider {
	return &pushProvider{baseProvider: newBaseProvider(url, timeout)}
}

func (p *pushProvider) Channel() model.Channel {
	return model.ChannelPush
}

func (p *pushProvider) Send(ctx context.Context, n *model.Notification) (*SendResult, error) {
	payload := map[string]string{
		"to":      n.Recipient,
		"channel": "push",
		"content": n.Content,
	}
	return p.doPost(ctx, payload)
}
