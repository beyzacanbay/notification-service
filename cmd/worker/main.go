package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/beyzacanbay/notification-service/internal/bootstrap"
	"github.com/beyzacanbay/notification-service/internal/delivery"
	"github.com/beyzacanbay/notification-service/internal/queue"
	"github.com/beyzacanbay/notification-service/internal/ratelimiter"
	"github.com/beyzacanbay/notification-service/internal/repository"
	ws "github.com/beyzacanbay/notification-service/internal/websocket"
	"github.com/beyzacanbay/notification-service/internal/worker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deps := bootstrap.Init(ctx, "notification-worker")
	defer deps.Close()

	cfg := deps.Config

	providers := delivery.NewProviderRegistry(
		delivery.NewSMSProvider(cfg.Webhook.URL, cfg.Webhook.Timeout),
		delivery.NewEmailProvider(cfg.Webhook.URL, cfg.Webhook.Timeout),
		delivery.NewPushProvider(cfg.Webhook.URL, cfg.Webhook.Timeout),
	)

	notificationRepo := repository.NewPostgresNotificationRepo(deps.DB)
	producer := queue.NewProducer(deps.Redis)
	consumer := queue.NewConsumer(deps.Redis, nil)
	dlq := queue.NewDLQ(deps.Redis)
	rl := ratelimiter.New(deps.Redis, cfg.Worker.RateLimit)

	retryCfg := delivery.RetryConfig{
		BaseDelay: cfg.Worker.RetryBaseDelay,
		MaxDelay:  cfg.Worker.RetryMaxDelay,
	}

	// WebSocket hub — publishes status updates to Redis Pub/Sub (API subscribes)
	wsHub := ws.NewHub(deps.Redis, deps.Logger)

	processor := worker.NewProcessor(notificationRepo, providers, rl, producer, dlq, retryCfg, wsHub, deps.Redis, deps.Logger)
	dispatcher := worker.NewDispatcher(consumer, processor, cfg.Worker.Concurrency, deps.Logger)

	dispatcher.Start(ctx)

	deps.Logger.Info("worker started",
		"concurrency", cfg.Worker.Concurrency,
		"rate_limit", cfg.Worker.RateLimit,
		"webhook_url", cfg.Webhook.URL,
	)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	deps.Logger.Info("received shutdown signal", "signal", sig)

	dispatcher.Stop()
	cancel()

	deps.Logger.Info("worker stopped")
}
