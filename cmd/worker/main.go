package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/beyzacanbay/notification-service/internal/config"
	"github.com/beyzacanbay/notification-service/internal/delivery"
	"github.com/beyzacanbay/notification-service/internal/queue"
	"github.com/beyzacanbay/notification-service/internal/ratelimiter"
	"github.com/beyzacanbay/notification-service/internal/repository"
	"github.com/beyzacanbay/notification-service/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// PostgreSQL
	dbPool, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to PostgreSQL")

	// Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to Redis")

	// Channel-specific providers (all hit the same webhook URL)
	providers := delivery.NewProviderRegistry(
		delivery.NewSMSProvider(cfg.Webhook.URL, cfg.Webhook.Timeout),
		delivery.NewEmailProvider(cfg.Webhook.URL, cfg.Webhook.Timeout),
		delivery.NewPushProvider(cfg.Webhook.URL, cfg.Webhook.Timeout),
	)

	// Dependencies
	notificationRepo := repository.NewPostgresNotificationRepo(dbPool)
	producer := queue.NewProducer(redisClient)
	consumer := queue.NewConsumer(redisClient, nil)
	dlq := queue.NewDLQ(redisClient)
	rl := ratelimiter.New(redisClient, cfg.Worker.RateLimit)

	retryCfg := delivery.RetryConfig{
		BaseDelay: cfg.Worker.RetryBaseDelay,
		MaxDelay:  cfg.Worker.RetryMaxDelay,
	}

	processor := worker.NewProcessor(notificationRepo, providers, rl, producer, dlq, retryCfg, logger)
	dispatcher := worker.NewDispatcher(consumer, processor, cfg.Worker.Concurrency, logger)

	dispatcher.Start(ctx)

	logger.Info("worker started",
		"concurrency", cfg.Worker.Concurrency,
		"rate_limit", cfg.Worker.RateLimit,
		"webhook_url", cfg.Webhook.URL,
	)

	// Block until shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("received shutdown signal", "signal", sig)

	dispatcher.Stop()
	cancel()

	logger.Info("worker stopped")
}
