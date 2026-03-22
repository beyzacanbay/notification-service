.PHONY: run-api run-worker build test test-e2e test-unit lint migrate-up migrate-down docker-up docker-down clean

# Run locally
run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

# Build binaries
build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

# Run all tests
test:
	go test ./... -v -race -count=1

# Run only e2e tests
test-e2e:
	go test ./internal/handler/ -v -race -count=1 -run TestE2E

# Run only unit tests (exclude e2e)
test-unit:
	go test ./... -v -race -count=1 -run "^Test[^E]"

# Run linter
lint:
	golangci-lint run ./...

# Database migrations
migrate-up:
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DATABASE_URL)" down

# Docker
docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

# Scale workers
scale-workers:
	docker compose up --scale worker=$(N) -d

# Swagger
swagger:
	docker run --rm -v $(pwd):/code ghcr.io/swaggo/swag:latest init -g cmd/api/main.go -o docs/swagger

# Clean build artifacts
clean:
	rm -rf bin/
