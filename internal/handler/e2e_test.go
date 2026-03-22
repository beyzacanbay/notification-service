package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
	"github.com/beyzacanbay/notification-service/internal/service"
)

func setupFullApp(repo *mockRepo, producer *mockProducer) *fiber.App {
	notifSvc := service.NewNotificationService(repo, producer, nil)
	templateSvc := service.NewTemplateService(newMockTemplateRepo())
	app := fiber.New()
	h := NewNotificationHandler(notifSvc, templateSvc)
	h.RegisterRoutes(app.Group("/api/v1/notifications"))
	return app
}

// --- E2E: Full Create → GetByID → GetStatus flow ---

func TestE2E_CreateAndQuery(t *testing.T) {
	repo := newMockRepo()
	producer := &mockProducer{}
	app := setupFullApp(repo, producer)

	// 1. Create
	body := `{"channel":"sms","recipient":"+905551234567","content":"Hello E2E"}`
	createReq := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")

	createResp, _ := app.Test(createReq)
	if createResp.StatusCode != fiber.StatusAccepted {
		t.Fatalf("create: expected 202, got %d", createResp.StatusCode)
	}

	var created dto.NotificationResponse
	_ = json.NewDecoder(createResp.Body).Decode(&created)

	if created.ID == uuid.Nil {
		t.Fatal("create: expected valid ID")
	}
	if created.Status != model.StatusPending {
		t.Fatalf("create: expected status pending, got %s", created.Status)
	}

	// 2. GetByID
	getReq := httptest.NewRequest("GET", "/api/v1/notifications/"+created.ID.String(), nil)
	getResp, _ := app.Test(getReq)

	if getResp.StatusCode != fiber.StatusOK {
		t.Fatalf("get: expected 200, got %d", getResp.StatusCode)
	}

	var fetched dto.NotificationResponse
	_ = json.NewDecoder(getResp.Body).Decode(&fetched)
	if fetched.ID != created.ID {
		t.Fatalf("get: expected ID %s, got %s", created.ID, fetched.ID)
	}

	// 3. GetStatus
	statusReq := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/notifications/%s/status", created.ID), nil)
	statusResp, _ := app.Test(statusReq)

	if statusResp.StatusCode != fiber.StatusOK {
		t.Fatalf("status: expected 200, got %d", statusResp.StatusCode)
	}

	var status map[string]interface{}
	_ = json.NewDecoder(statusResp.Body).Decode(&status)
	if status["status"] != "pending" {
		t.Fatalf("status: expected pending, got %v", status["status"])
	}
}

// --- E2E: Create → Cancel → Verify cancelled ---

func TestE2E_CreateAndCancel(t *testing.T) {
	repo := newMockRepo()
	app := setupFullApp(repo, &mockProducer{})

	// 1. Create
	body := `{"channel":"email","recipient":"test@test.com","content":"Cancel me"}`
	createReq := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")

	createResp, _ := app.Test(createReq)
	var created dto.NotificationResponse
	_ = json.NewDecoder(createResp.Body).Decode(&created)

	// 2. Cancel
	cancelReq := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/notifications/%s/cancel", created.ID), nil)
	cancelResp, _ := app.Test(cancelReq)

	if cancelResp.StatusCode != fiber.StatusOK {
		t.Fatalf("cancel: expected 200, got %d", cancelResp.StatusCode)
	}

	// 3. Verify status is cancelled
	statusReq := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/notifications/%s/status", created.ID), nil)
	statusResp, _ := app.Test(statusReq)

	var status map[string]interface{}
	_ = json.NewDecoder(statusResp.Body).Decode(&status)
	if status["status"] != "cancelled" {
		t.Fatalf("expected cancelled, got %v", status["status"])
	}

	// 4. Cancel again should fail
	cancelReq2 := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/notifications/%s/cancel", created.ID), nil)
	cancelResp2, _ := app.Test(cancelReq2)

	if cancelResp2.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("double cancel: expected 400, got %d", cancelResp2.StatusCode)
	}
}

// --- E2E: Batch create with partial validation failure ---

func TestE2E_BatchPartialFailure(t *testing.T) {
	repo := newMockRepo()
	app := setupFullApp(repo, &mockProducer{})

	body := `{"notifications":[
		{"channel":"sms","recipient":"+905551234567","content":"Valid"},
		{"channel":"sms","recipient":"invalid-phone","content":"Bad recipient"},
		{"channel":"push","recipient":"device-token-12345","content":"Also valid"}
	]}`
	req := httptest.NewRequest("POST", "/api/v1/notifications/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusAccepted {
		t.Fatalf("batch: expected 202, got %d", resp.StatusCode)
	}

	var result dto.BatchCreateResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)

	if result.TotalCreated != 2 {
		t.Fatalf("batch: expected 2 created, got %d", result.TotalCreated)
	}
	if result.TotalFailed != 1 {
		t.Fatalf("batch: expected 1 failed, got %d", result.TotalFailed)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("batch: expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Index != 1 {
		t.Fatalf("batch: expected error at index 1, got %d", result.Errors[0].Index)
	}
}

// --- E2E: Scheduled notification ---

func TestE2E_ScheduledNotification(t *testing.T) {
	repo := newMockRepo()
	producer := &mockProducer{}
	app := setupFullApp(repo, producer)

	body := `{"channel":"sms","recipient":"+905551234567","content":"Future","scheduled_at":"2026-12-25T10:00:00Z"}`
	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusAccepted {
		t.Fatalf("scheduled: expected 202, got %d", resp.StatusCode)
	}

	var created dto.NotificationResponse
	_ = json.NewDecoder(resp.Body).Decode(&created)

	if created.ScheduledAt == nil {
		t.Fatal("scheduled: expected scheduled_at to be set")
	}
	if producer.lastID == uuid.Nil {
		t.Fatal("scheduled: expected notification to be enqueued")
	}
}

// --- E2E: Idempotency with header ---

func TestE2E_IdempotencyHeader(t *testing.T) {
	repo := newMockRepo()
	app := setupFullApp(repo, &mockProducer{})

	// Redis is nil in test, so idempotency won't deduplicate
	// But we verify the flow doesn't break with the header
	body := `{"channel":"sms","recipient":"+905551234567","content":"Idempotent"}`

	req1 := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBufferString(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", "test-key-123")

	resp1, _ := app.Test(req1)
	if resp1.StatusCode != fiber.StatusAccepted {
		t.Fatalf("first: expected 202, got %d", resp1.StatusCode)
	}

	var first dto.NotificationResponse
	_ = json.NewDecoder(resp1.Body).Decode(&first)

	if first.ID == uuid.Nil {
		t.Fatal("first: expected valid ID")
	}
}

// --- E2E: List with pagination ---

func TestE2E_ListPagination(t *testing.T) {
	repo := newMockRepo()
	app := setupFullApp(repo, &mockProducer{})

	// Create 3 notifications
	for i := 0; i < 3; i++ {
		body := fmt.Sprintf(`{"channel":"sms","recipient":"+90555123456%d","content":"Msg %d"}`, i, i)
		req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		_, _ = app.Test(req)
	}

	// List page 1, per_page 2
	listReq := httptest.NewRequest("GET", "/api/v1/notifications?page=1&per_page=2", nil)
	listResp, _ := app.Test(listReq)

	if listResp.StatusCode != fiber.StatusOK {
		t.Fatalf("list: expected 200, got %d", listResp.StatusCode)
	}

	var result dto.PaginatedResponse
	_ = json.NewDecoder(listResp.Body).Decode(&result)

	if result.TotalItems != 3 {
		t.Fatalf("list: expected 3 total items, got %d", result.TotalItems)
	}
	if result.Page != 1 {
		t.Fatalf("list: expected page 1, got %d", result.Page)
	}
	if result.PerPage != 2 {
		t.Fatalf("list: expected per_page 2, got %d", result.PerPage)
	}
	if result.TotalPages != 2 {
		t.Fatalf("list: expected 2 total pages, got %d", result.TotalPages)
	}
}
