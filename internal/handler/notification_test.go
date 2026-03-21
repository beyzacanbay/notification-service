package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/model"
	"github.com/beyzacanbay/notification-service/internal/repository"
)

type mockProducer struct {
	lastNotification *model.Notification
	err              error
}

func (m *mockProducer) Enqueue(_ context.Context, n *model.Notification) error {
	m.lastNotification = n
	return m.err
}

type mockRepo struct {
	notifications map[uuid.UUID]*model.Notification
	err           error
}

func newMockRepo() *mockRepo {
	return &mockRepo{notifications: make(map[uuid.UUID]*model.Notification)}
}

func (m *mockRepo) Create(_ context.Context, n *model.Notification) error {
	if m.err != nil {
		return m.err
	}
	m.notifications[n.ID] = n
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Notification, error) {
	if n, ok := m.notifications[id]; ok {
		return n, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockRepo) UpdateStatus(_ context.Context, id uuid.UUID, status model.Status) error {
	if n, ok := m.notifications[id]; ok {
		n.Status = status
		return nil
	}
	return repository.ErrNotFound
}

func setupApp(repo repository.NotificationRepository, producer Enqueuer) *fiber.App {
	app := fiber.New()
	h := NewNotificationHandler(repo, producer)
	h.RegisterRoutes(app.Group("/api/v1/notifications"))
	return app
}

func TestCreate_Success(t *testing.T) {
	repo := newMockRepo()
	mock := &mockProducer{}
	app := setupApp(repo, mock)

	body := `{"channel":"sms","recipient":"+905551234567","content":"Hello"}`
	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != fiber.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	if mock.lastNotification == nil {
		t.Fatal("expected notification to be enqueued")
	}

	if mock.lastNotification.Channel != model.ChannelSMS {
		t.Fatalf("expected channel sms, got %s", mock.lastNotification.Channel)
	}

	if len(repo.notifications) != 1 {
		t.Fatalf("expected 1 notification in repo, got %d", len(repo.notifications))
	}
}

func TestCreate_MissingFields(t *testing.T) {
	repo := newMockRepo()
	mock := &mockProducer{}
	app := setupApp(repo, mock)

	body := `{"channel":"sms"}`
	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreate_InvalidChannel(t *testing.T) {
	repo := newMockRepo()
	mock := &mockProducer{}
	app := setupApp(repo, mock)

	body := `{"channel":"telegram","recipient":"+905551234567","content":"Hello"}`
	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreate_WithPriority(t *testing.T) {
	repo := newMockRepo()
	mock := &mockProducer{}
	app := setupApp(repo, mock)

	body := `{"channel":"email","recipient":"test@test.com","content":"Hello","priority":0}`
	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != fiber.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["priority"].(float64) != 0 {
		t.Fatalf("expected priority 0, got %v", result["priority"])
	}
}
