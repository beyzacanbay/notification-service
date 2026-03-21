package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/redis/go-redis/v9"

	"github.com/beyzacanbay/notification-service/internal/config"
	"github.com/beyzacanbay/notification-service/internal/queue"
	"github.com/beyzacanbay/notification-service/internal/server"
)

func main() {
	cfg := config.Load()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to Redis")

	producer := queue.NewProducer(redisClient)

	app := server.NewServer(producer)

	log.Fatal(app.Listen(":8081"))
}
