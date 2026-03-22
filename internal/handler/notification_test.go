package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
	"github.com/beyzacanbay/notification-service/internal/repository"
	"github.com/beyzacanbay/notification-service/internal/service"
)

// --- mocks ---

type mockProducer struct {
	lastID uuid.UUID
	err    error
}

func (m *mockProducer) Enqueue(_ context.Context, id uuid.UUID, _ model.Priority, _ model.Channel) error {
	m.lastID = id
	return m.err
}

func (m *mockProducer) EnqueueAt(_ context.Context, id uuid.UUID, _ model.Priority, _ model.Channel, _ time.Time) error {
	m.lastID = id
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

func (m *mockRepo) CreateBatch(_ context.Context, notifications []*model.Notification) error {
	if m.err != nil {
		return m.err
	}
	for _, n := range notifications {
		m.notifications[n.ID] = n
	}
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Notification, error) {
	if n, ok := m.notifications[id]; ok {
		return n, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockRepo) GetByBatchID(_ context.Context, batchID uuid.UUID) ([]*model.Notification, error) {
	var result []*model.Notification
	for _, n := range m.notifications {
		if n.BatchID != nil && *n.BatchID == batchID {
			result = append(result, n)
		}
	}
	return result, nil
}

func (m *mockRepo) List(_ context.Context, _ *dto.ListNotificationsRequest) ([]*model.Notification, int64, error) {
	var result []*model.Notification
	for _, n := range m.notifications {
		result = append(result, n)
	}
	return result, int64(len(result)), nil
}

func (m *mockRepo) UpdateStatus(_ context.Context, id uuid.UUID, status model.Status, _ *string) error {
	if n, ok := m.notifications[id]; ok {
		n.Status = status
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockRepo) IncrementAttempt(_ context.Context, id uuid.UUID, _ string) error {
	if n, ok := m.notifications[id]; ok {
		n.AttemptCount++
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockRepo) MarkSent(_ context.Context, id uuid.UUID) error {
	if n, ok := m.notifications[id]; ok {
		n.Status = model.StatusSent
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockRepo) GetMetrics(_ context.Context) (*dto.MetricsResponse, error) {
	return &dto.MetricsResponse{}, nil
}

func (m *mockRepo) seedNotification(status model.Status) *model.Notification {
	n := &model.Notification{
		ID:          uuid.New(),
		Channel:     model.ChannelSMS,
		Recipient:   "+905551234567",
		Content:     "test",
		Priority:    model.PriorityNormal,
		Status:      status,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.notifications[n.ID] = n
	return n
}

func setupApp(repo *mockRepo, producer *mockProducer) *fiber.App {
	svc := service.NewNotificationService(repo, producer, nil) // nil redis = skip idempotency
	app := fiber.New()
	h := NewNotificationHandler(svc, nil)
	h.RegisterRoutes(app.Group("/api/v1/notifications"))
	return app
}

// --- Create ---

func TestCreate_Success(t *testing.T) {
	repo := newMockRepo()
	mock := &mockProducer{}
	app := setupApp(repo, mock)

	body := `{"channel":"sms","recipient":"+905551234567","content":"Hello"}`
	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}
	if len(repo.notifications) != 1 {
		t.Fatalf("expected 1 notification in repo, got %d", len(repo.notifications))
	}
	if mock.lastID == uuid.Nil {
		t.Fatal("expected notification to be enqueued")
	}
}

func TestCreate_MissingFields(t *testing.T) {
	app := setupApp(newMockRepo(), &mockProducer{})

	body := `{"channel":"sms"}`
	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreate_InvalidChannel(t *testing.T) {
	app := setupApp(newMockRepo(), &mockProducer{})

	body := `{"channel":"telegram","recipient":"+905551234567","content":"Hello"}`
	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// --- CreateBatch ---

func TestCreateBatch_Success(t *testing.T) {
	repo := newMockRepo()
	app := setupApp(repo, &mockProducer{})

	body := `{"notifications":[
		{"channel":"sms","recipient":"+905551234567","content":"Hello 1"},
		{"channel":"email","recipient":"test@test.com","content":"Hello 2"}
	]}`
	req := httptest.NewRequest("POST", "/api/v1/notifications/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}
	if len(repo.notifications) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(repo.notifications))
	}
}

func TestCreateBatch_Empty(t *testing.T) {
	app := setupApp(newMockRepo(), &mockProducer{})

	body := `{"notifications":[]}`
	req := httptest.NewRequest("POST", "/api/v1/notifications/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// --- GetByID ---

func TestGetByID_Success(t *testing.T) {
	repo := newMockRepo()
	n := repo.seedNotification(model.StatusPending)
	app := setupApp(repo, &mockProducer{})

	req := httptest.NewRequest("GET", "/api/v1/notifications/"+n.ID.String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	app := setupApp(newMockRepo(), &mockProducer{})

	req := httptest.NewRequest("GET", "/api/v1/notifications/"+uuid.New().String(), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// --- GetStatus ---

func TestGetStatus_Success(t *testing.T) {
	repo := newMockRepo()
	n := repo.seedNotification(model.StatusSent)
	app := setupApp(repo, &mockProducer{})

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/notifications/%s/status", n.ID), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "sent" {
		t.Fatalf("expected status sent, got %v", result["status"])
	}
}

func TestGetStatus_NotFound(t *testing.T) {
	app := setupApp(newMockRepo(), &mockProducer{})

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/notifications/%s/status", uuid.New()), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// --- Cancel ---

func TestCancel_Success(t *testing.T) {
	repo := newMockRepo()
	n := repo.seedNotification(model.StatusPending)
	app := setupApp(repo, &mockProducer{})

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/notifications/%s/cancel", n.ID), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if repo.notifications[n.ID].Status != model.StatusCancelled {
		t.Fatalf("expected status cancelled, got %s", repo.notifications[n.ID].Status)
	}
}

func TestCancel_AlreadySent(t *testing.T) {
	repo := newMockRepo()
	n := repo.seedNotification(model.StatusSent)
	app := setupApp(repo, &mockProducer{})

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/notifications/%s/cancel", n.ID), nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// --- List ---

func TestList_Success(t *testing.T) {
	repo := newMockRepo()
	repo.seedNotification(model.StatusPending)
	repo.seedNotification(model.StatusSent)
	app := setupApp(repo, &mockProducer{})

	req := httptest.NewRequest("GET", "/api/v1/notifications", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result dto.PaginatedResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result.TotalItems != 2 {
		t.Fatalf("expected 2 total items, got %d", result.TotalItems)
	}
}

func TestList_WithFilters(t *testing.T) {
	repo := newMockRepo()
	repo.seedNotification(model.StatusPending)
	repo.seedNotification(model.StatusSent)
	app := setupApp(repo, &mockProducer{})

	req := httptest.NewRequest("GET", "/api/v1/notifications?status=sent&channel=sms&page=1&per_page=5", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result dto.PaginatedResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)

	if result.Page != 1 {
		t.Fatalf("expected page 1, got %d", result.Page)
	}
	if result.PerPage != 5 {
		t.Fatalf("expected per_page 5, got %d", result.PerPage)
	}
}

func TestList_Empty(t *testing.T) {
	app := setupApp(newMockRepo(), &mockProducer{})

	req := httptest.NewRequest("GET", "/api/v1/notifications", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result dto.PaginatedResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result.TotalItems != 0 {
		t.Fatalf("expected 0 total items, got %d", result.TotalItems)
	}
}
