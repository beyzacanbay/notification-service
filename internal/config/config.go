package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Worker   WorkerConfig
	Webhook  WebhookConfig
	Tracing  TracingConfig
}

type TracingConfig struct {
	Endpoint string
	Enabled  bool
}

type ServerConfig struct {
	Port int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode)
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type WorkerConfig struct {
	Concurrency    int
	RateLimit      int
	MaxRetries     int
	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration
}

type WebhookConfig struct {
	URL     string
	Timeout time.Duration
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnvInt("SERVER_PORT", 8081),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "notifications"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Worker: WorkerConfig{
			Concurrency:    getEnvInt("WORKER_CONCURRENCY", 5),
			RateLimit:      getEnvInt("WORKER_RATE_LIMIT", 100),
			MaxRetries:     getEnvInt("WORKER_MAX_RETRIES", 3),
			RetryBaseDelay: getEnvDuration("WORKER_RETRY_BASE_DELAY", 1*time.Second),
			RetryMaxDelay:  getEnvDuration("WORKER_RETRY_MAX_DELAY", 5*time.Minute),
		},
		Webhook: WebhookConfig{
			URL:     getEnv("WEBHOOK_URL", "https://webhook.site/test"),
			Timeout: getEnvDuration("WEBHOOK_TIMEOUT", 10*time.Second),
		},
		Tracing: TracingConfig{
			Endpoint: getEnv("OTEL_EXPORTER_ENDPOINT", "localhost:4318"),
			Enabled:  getEnv("OTEL_ENABLED", "true") == "true",
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
