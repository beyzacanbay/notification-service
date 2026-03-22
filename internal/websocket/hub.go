package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

const pubsubChannel = "notifications:status"

type StatusUpdate struct {
	NotificationID string `json:"notification_id"`
	Status         string `json:"status"`
}

type Hub struct {
	redis   *redis.Client
	logger  *slog.Logger
	clients map[*websocket.Conn]bool
	mu      sync.RWMutex
}

func NewHub(redisClient *redis.Client, logger *slog.Logger) *Hub {
	return &Hub{
		redis:   redisClient,
		logger:  logger,
		clients: make(map[*websocket.Conn]bool),
	}
}

// Subscribe listens to Redis Pub/Sub and broadcasts to all connected clients.
func (h *Hub) Subscribe(ctx context.Context) {
	sub := h.redis.Subscribe(ctx, pubsubChannel)
	defer sub.Close()

	ch := sub.Channel()
	h.logger.Info("websocket hub subscribed to Redis Pub/Sub", "channel", pubsubChannel)

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			if msg == nil {
				continue
			}
			h.broadcast([]byte(msg.Payload))
		}
	}
}

// BroadcastStatus publishes a status update to Redis Pub/Sub.
// Worker calls this — API hub picks it up via Subscribe.
func (h *Hub) BroadcastStatus(notificationID, status string) {
	update := StatusUpdate{
		NotificationID: notificationID,
		Status:         status,
	}
	data, err := json.Marshal(update)
	if err != nil {
		return
	}
	h.redis.Publish(context.Background(), pubsubChannel, data)
}

func (h *Hub) broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			h.logger.Error("websocket write error", "error", err)
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

func (h *Hub) register(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
	h.logger.Info("websocket client connected", "remote_addr", conn.RemoteAddr())
}

func (h *Hub) unregister(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
	h.logger.Info("websocket client disconnected", "remote_addr", conn.RemoteAddr())
}

// Handler returns the WebSocket handler for Fiber.
func (h *Hub) Handler() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		h.register(c)
		defer h.unregister(c)

		// Keep connection alive — read messages (client can send pings)
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				break
			}
		}
	})
}

// UpgradeMiddleware returns middleware that checks for WebSocket upgrade.
func UpgradeMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}
