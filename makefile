.PHONY: all help setup build test test-coverage lint clean run docker-build docker-push
.PHONY: compose-up compose-down migrate-up migrate-down generate-proto

# Binary output
BINARY_NAME=app
BUILD_DIR=build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOLINT=golangci-lint
GOMOD=$(GOCMD) mod
GOGEN=$(GOCMD) generate

# Docker parameters
DOCKER_REGISTRY?=your-registry
DOCKER_IMAGE=user-service
DOCKER_TAG?=latest

# Tool installation paths
GOBIN=$(shell go env GOPATH)/bin
PROTOC_GEN_GO=$(GOBIN)/protoc-gen-go
PROTOC_GEN_GO_GRPC=$(GOBIN)/protoc-gen-go-grpc
MIGRATE=$(GOBIN)/migrate

all: clean generate-proto setup test lint build compose-up ## Build the project

help: ## Display available commands
	@echo "Available commands:"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

setup: ## Install required tools and dependencies
	GOBIN=$(GOBIN) go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	GOBIN=$(GOBIN) go install github.com/swaggo/swag/cmd/swag@latest
	GOBIN=$(GOBIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	GOBIN=$(GOBIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	GOBIN=$(GOBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOMOD) download
	$(GOMOD) tidy

build: ## Build the application
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/server/main.go

test: ## Run tests with coverage percentage
	$(GOTEST) -race -cover ./...

lint: ## Run linters
	$(GOBIN)/golangci-lint run

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -rf internal/api/grpc/gen

run: build ## Run the application (loads .env via godotenv in main.go)
	./$(BUILD_DIR)/$(BINARY_NAME)

docker-build: ## Build Docker image
	docker build -t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-push: ## Push Docker image
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)

compose-up: ## Start all services via Docker Compose and run migrations
	docker-compose up -d
	@echo "Waiting for database to be ready..."
	@sleep 5
	@$(MAKE) migrate-up

compose-down: ## Stop all services and remove containers
	docker-compose down -v

# Load .env for migrate commands
migrate-up: ## Apply database migrations
	@echo "Loading .env and applying migrations..."
	@export $$(grep -v '^#' .env | xargs) && $(MIGRATE) -path migrations -database "$${DATABASE_URL}" up

migrate-down: ## Rollback last database migration
	@echo "Loading .env and rolling back migration..."
	@export $$(grep -v '^#' .env | xargs) && $(MIGRATE) -path migrations -database "$${DATABASE_URL}" down 1

generate-proto: ## Generate Go code from Protocol Buffers
	@mkdir -p internal/api/grpc/gen
	protoc \
		--proto_path=proto \
		--go_out=internal/api/grpc/gen \
		--go_opt=paths=source_relative \
		--go-grpc_out=internal/api/grpc/gen \
		--go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-go=$(PROTOC_GEN_GO) \
		--plugin=protoc-gen-go-grpc=$(PROTOC_GEN_GO_GRPC) \
		user/user.proto
	@rm -f proto/user/*.pb.go
