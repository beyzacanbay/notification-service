package bootstrap

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/beyzacanbay/notification-service/internal/config"
	"github.com/beyzacanbay/notification-service/internal/tracing"
)

type Deps struct {
	Config         *config.Config
	Logger         *slog.Logger
	DB             *pgxpool.Pool
	Redis          *redis.Client
	TracerShutdown func(context.Context) error
}

func Init(ctx context.Context, serviceName string) *Deps {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()

	// Tracing
	var tracerShutdown func(context.Context) error
	if cfg.Tracing.Enabled {
		shutdown, err := tracing.Init(ctx, serviceName, cfg.Tracing.Endpoint)
		if err != nil {
			logger.Warn("failed to init tracing, continuing without it", "error", err)
		} else {
			tracerShutdown = shutdown
			logger.Info("tracing initialized", "endpoint", cfg.Tracing.Endpoint)
		}
	}

	dbPool, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	if err := dbPool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to PostgreSQL")

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to Redis")

	return &Deps{
		Config:         cfg,
		Logger:         logger,
		DB:             dbPool,
		Redis:          redisClient,
		TracerShutdown: tracerShutdown,
	}
}

func (d *Deps) Close() {
	if d.TracerShutdown != nil {
		d.TracerShutdown(context.Background())
	}
	d.DB.Close()
	d.Redis.Close()
}
