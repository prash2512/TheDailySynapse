.PHONY: build run test clean fmt docker-build docker-run help

BINARY_NAME=synapse
BACKEND_DIR=./backend
BIN_DIR=./bin
DOCKER_IMAGE=dailysynapse

help:
	@echo "TheDailySynapse - Available targets:"
	@echo ""
	@echo "  make build        Build the backend binary"
	@echo "  make run          Build and run the backend"
	@echo "  make test         Run tests"
	@echo "  make fmt          Format Go code"
	@echo "  make clean        Remove build artifacts"
	@echo "  make docker-build Build Docker image"
	@echo "  make docker-run   Run in Docker container"
	@echo ""

fmt:
	@echo "Formatting code..."
	@gofmt -w $(BACKEND_DIR)

build: fmt
	@echo "Building backend..."
	@mkdir -p $(BIN_DIR)
	@cd $(BACKEND_DIR) && go build -o ../$(BIN_DIR)/$(BINARY_NAME) ./cmd/synapse

run: build
	@echo "Starting backend..."
	@$(BIN_DIR)/$(BINARY_NAME)

test:
	@echo "Running tests..."
	@cd $(BACKEND_DIR) && go test -v ./...

clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -f synapse.db

docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE):latest .

docker-run: docker-build
	@echo "Running in Docker..."
	@docker run -d \
		--name dailysynapse \
		-p 8080:8080 \
		-v dailysynapse-data:/app/data \
		-e GEMINI_API_KEY=$(GEMINI_API_KEY) \
		$(DOCKER_IMAGE):latest
	@echo "Container started. API available at http://localhost:8080"

docker-stop:
	@docker stop dailysynapse 2>/dev/null || true
	@docker rm dailysynapse 2>/dev/null || true
