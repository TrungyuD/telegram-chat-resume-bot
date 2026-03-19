.PHONY: build run dev clean tidy test lint fmt fmt-prettier vet check hooks docker-build docker-build-amd64 docker-up docker-down docker-logs help

BINARY := telegram-chat-resume-bot
BUILD_DIR := dist

## Build & Run
build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/bot/

run: build
	./$(BUILD_DIR)/$(BINARY)

dev:
	air

clean:
	rm -rf $(BUILD_DIR)/ tmp/

## Dependencies
tidy:
	go mod tidy

## Quality
fmt:
	gofmt -w .
	goimports -w . 2>/dev/null || true

fmt-prettier:
	pnpm prettier --write .

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test -race ./...

test-cover:
	go test -race -coverprofile=cover.out ./...
	go tool cover -html=cover.out -o cover.html
	@echo "Coverage report: cover.html"

check: fmt fmt-prettier vet lint test
	@echo "All checks passed."

## Git Hooks
hooks:
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed."

## Docker
docker-build:
	docker compose build

docker-build-amd64:
	docker buildx build --platform linux/amd64 -t telegram-bot .

docker-up:
	@test -f .env.docker || (cp .env.docker.example .env.docker && echo "Created .env.docker from example — fill in your tokens before starting.")
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

## Help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build & Run:"
	@echo "  build         Build binary to $(BUILD_DIR)/$(BINARY)"
	@echo "  run           Build and run"
	@echo "  dev           Run with air (hot-reload)"
	@echo "  clean         Remove build artifacts"
	@echo ""
	@echo "Dependencies:"
	@echo "  tidy          Run go mod tidy"
	@echo ""
	@echo "Quality:"
	@echo "  fmt           Format Go files (gofmt + goimports)"
	@echo "  fmt-prettier  Format non-Go files (JSON, YAML, MD)"
	@echo "  vet           Run go vet"
	@echo "  lint          Run golangci-lint"
	@echo "  test          Run tests with race detector"
	@echo "  test-cover    Run tests with coverage report"
	@echo "  check         Run fmt + prettier + vet + lint + test"
	@echo ""
	@echo "Setup:"
	@echo "  hooks         Install git pre-commit hook"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build       Build Docker image"
	@echo "  docker-build-amd64 Build for linux/amd64 (cross-platform)"
	@echo "  docker-up          Start containers (detached)"
	@echo "  docker-down        Stop containers"
	@echo "  docker-logs        Follow container logs"
