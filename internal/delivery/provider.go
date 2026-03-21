package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/beyzacanbay/notification-service/internal/model"
)

type SendResult struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
}

// Provider is the interface for sending notifications via external services.
type Provider interface {
	Send(ctx context.Context, n *model.Notification) (*SendResult, error)
	Channel() model.Channel
}

// ProviderRegistry holds providers keyed by channel.
type ProviderRegistry struct {
	providers map[model.Channel]Provider
}

func NewProviderRegistry(providers ...Provider) *ProviderRegistry {
	r := &ProviderRegistry{providers: make(map[model.Channel]Provider)}
	for _, p := range providers {
		r.providers[p.Channel()] = p
	}
	return r
}

func (r *ProviderRegistry) Get(channel model.Channel) (Provider, error) {
	p, ok := r.providers[channel]
	if !ok {
		return nil, fmt.Errorf("no provider registered for channel: %s", channel)
	}
	return p, nil
}

// baseProvider contains shared HTTP logic for all channel providers.
type baseProvider struct {
	url    string
	client *http.Client
}

func newBaseProvider(url string, timeout time.Duration) baseProvider {
	return baseProvider{
		url:    url,
		client: &http.Client{Timeout: timeout},
	}
}

func (p *baseProvider) doPost(ctx context.Context, payload interface{}) (*SendResult, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, &RetryableError{Err: fmt.Errorf("send request: %w", err)}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 500 || resp.StatusCode == 429 {
		return nil, &RetryableError{Err: fmt.Errorf("server error %d: %s", resp.StatusCode, string(respBody))}
	}
	if resp.StatusCode >= 400 {
		return nil, &PermanentError{Err: fmt.Errorf("client error %d: %s", resp.StatusCode, string(respBody))}
	}

	var result SendResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		result = SendResult{Status: "accepted"}
	}

	return &result, nil
}
