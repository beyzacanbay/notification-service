package main

import (
	"context"
	"fmt"
	"log"

	"github.com/beyzacanbay/notification-service/internal/bootstrap"
	"github.com/beyzacanbay/notification-service/internal/queue"
	"github.com/beyzacanbay/notification-service/internal/repository"
	"github.com/beyzacanbay/notification-service/internal/server"
)

// @title Notification Service API
// @version 1.0
// @description Event-driven notification system for processing and delivering messages through multiple channels (SMS, Email, Push).
// @host localhost:8081
// @BasePath /
func main() {
	ctx := context.Background()
	deps := bootstrap.Init(ctx)
	defer deps.Close()

	notificationRepo := repository.NewPostgresNotificationRepo(deps.DB)
	templateRepo := repository.NewPostgresTemplateRepo(deps.DB)
	producer := queue.NewProducer(deps.Redis)

	app := server.NewServer(deps.DB, deps.Redis, notificationRepo, templateRepo, producer, deps.Logger)

	addr := fmt.Sprintf(":%d", deps.Config.Server.Port)
	deps.Logger.Info("API server starting", "port", deps.Config.Server.Port)
	log.Fatal(app.Listen(addr))
}
