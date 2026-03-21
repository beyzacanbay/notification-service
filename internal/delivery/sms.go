package delivery

import (
	"context"
	"time"

	"github.com/beyzacanbay/notification-service/internal/model"
)

type smsProvider struct {
	baseProvider
}

func NewSMSProvider(url string, timeout time.Duration) Provider {
	return &smsProvider{baseProvider: newBaseProvider(url, timeout)}
}

func (p *smsProvider) Channel() model.Channel {
	return model.ChannelSMS
}

func (p *smsProvider) Send(ctx context.Context, n *model.Notification) (*SendResult, error) {
	payload := map[string]string{
		"to":      n.Recipient,
		"channel": "sms",
		"content": n.Content,
	}
	return p.doPost(ctx, payload)
}
