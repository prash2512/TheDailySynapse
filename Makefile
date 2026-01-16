.PHONY: build run test clean docker-build

BINARY_NAME=synapse
BACKEND_DIR=./backend
BIN_DIR=./bin

build:
	@echo "Building backend..."
	@cd $(BACKEND_DIR) && go build -o ../$(BIN_DIR)/${BINARY_NAME} ./cmd/synapse

run: build
	@echo "Starting backend..."
	@$(BIN_DIR)/${BINARY_NAME}

test:
	@echo "Testing backend..."
	@cd $(BACKEND_DIR) && go test ./...

clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)

docker-build:
	@echo "Building Docker image..."
	@docker build -t dailysynapse:latest .