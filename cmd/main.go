package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/beyzacanbay/notification-service/internal/config"
	"github.com/beyzacanbay/notification-service/internal/queue"
	"github.com/beyzacanbay/notification-service/internal/repository"
	"github.com/beyzacanbay/notification-service/internal/server"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// PostgreSQL
	dbPool, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to PostgreSQL")

	// Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to Redis")

	// Dependencies
	notificationRepo := repository.NewPostgresNotificationRepo(dbPool)
	producer := queue.NewProducer(redisClient)

	app := server.NewServer(dbPool, redisClient, notificationRepo, producer)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	slog.Info("API server starting", "port", cfg.Server.Port)
	log.Fatal(app.Listen(addr))
}
