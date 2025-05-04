# Variables
APP_NAME := axicalendar-api
# Default values for environment variables, can be overridden: make run DYNAMODB_TABLE_NAME=my-table
DYNAMODB_TABLE_NAME ?= AxiCalendarTable-dev
DUMMY_USER_ID ?= 11111111-1111-1111-1111-111111111111
AWS_PROFILE := axicalendar-dev
AWS_ENDPOINT_URL := http://localhost:8000
DOCKER_CONTAINER_NAME := dynamodb-local-axicalendar
OAPI_CODEGEN_CMD := oapi-codegen

# Targets
.PHONY: all build run clean setup setup-db start-db stop-db create-table delete-table gen lint fmt test test-cover help

all: build

help:
	@echo "Usage: make [target] [VAR=value]..."
	@echo ""
	@echo "Targets:"
	@echo "  help          Show this help message"
	@echo "  setup         Download Go module dependencies"
	@echo "  build         Build the application"
	@echo "  run           Run the application (requires local DynamoDB running and table created)"
	@echo "  clean         Remove build artifacts"
	@echo "  setup-db      Start DynamoDB Local (Docker) and create the table"
	@echo "  start-db      Start DynamoDB Local (Docker) in the background (pulls image if needed)"
	@echo "  stop-db       Stop and remove the DynamoDB Local Docker container"
	@echo "  create-table  Create the DynamoDB table locally (requires DynamoDB running)"
	@echo "  delete-table  Delete the DynamoDB table locally (requires DynamoDB running)"
	@echo "  gen           Generate Go code from OpenAPI specification"
	@echo "  lint          Lint the code"
	@echo "  fmt           Format the code"
	@echo "  test          Run tests"
	@echo "  test-cover    Run tests with coverage"
	@echo ""
	@echo "Variables (can be overridden):"
	@echo "  DYNAMODB_TABLE_NAME (default: $(DYNAMODB_TABLE_NAME))"
	@echo "  DUMMY_USER_ID       (default: $(DUMMY_USER_ID))"

# Setup Go Modules
setup:
	@echo "Downloading Go module dependencies..."
	go mod tidy
	go mod download

# Build
build:
	@echo "Building application..."
	go build -o $(APP_NAME) ./cmd/api/main.go

# Run
run: build
	@echo "Running application on localhost:8080..."
	@echo "Using DYNAMODB_TABLE_NAME=$(DYNAMODB_TABLE_NAME)"
	@echo "Using DUMMY_USER_ID=$(DUMMY_USER_ID)"
	@echo "Using AWS_PROFILE=$(AWS_PROFILE)"
	@echo "Make sure DynamoDB is running and the table is created (make setup-db)."
	@echo "Press Ctrl+C to stop."
	@export DYNAMODB_TABLE_NAME=$(DYNAMODB_TABLE_NAME) && \
	export DUMMY_USER_ID=$(DUMMY_USER_ID) && \
	export AWS_PROFILE=$(AWS_PROFILE) && \
	./$(APP_NAME)

# Clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(APP_NAME)

# Database Setup
setup-db: start-db create-table

start-db:
	@echo "Checking for amazon/dynamodb-local Docker image..."
	@docker image inspect amazon/dynamodb-local > /dev/null 2>&1 || (echo "Image not found, pulling..." && docker pull amazon/dynamodb-local)
	@echo "Starting DynamoDB Local in Docker container '$(DOCKER_CONTAINER_NAME)'..."
	@if [ -z "$(docker ps -q -f name=$(DOCKER_CONTAINER_NAME))" ]; then \
		if docker ps -aq -f status=exited -f name=$(DOCKER_CONTAINER_NAME); then \
			echo "Removing existing stopped container..."; \
			docker rm $(DOCKER_CONTAINER_NAME) > /dev/null; \
		fi; \
		echo "Starting new container..."; \
		docker run -d --name $(DOCKER_CONTAINER_NAME) -p 8000:8000 amazon/dynamodb-local; \
		echo "Waiting for DynamoDB Local to be ready..."; \
		sleep 5; \
	else \
		echo "Container '$(DOCKER_CONTAINER_NAME)' is already running."; \
	fi

stop-db:
	@echo "Stopping and removing DynamoDB Local Docker container '$(DOCKER_CONTAINER_NAME)'..."
	@docker stop $(DOCKER_CONTAINER_NAME) > /dev/null 2>&1 || echo "Container not running or already stopped."
	@docker rm $(DOCKER_CONTAINER_NAME) > /dev/null 2>&1 || echo "Container not found or already removed."

create-table:
	@echo "Creating DynamoDB table '$(DYNAMODB_TABLE_NAME)' locally..."
	@echo "Using AWS_PROFILE=$(AWS_PROFILE) and AWS_ENDPOINT_URL=$(AWS_ENDPOINT_URL)"
	@aws dynamodb create-table \
		--table-name $(DYNAMODB_TABLE_NAME) \
		--attribute-definitions \
			AttributeName=PK,AttributeType=S \
			AttributeName=SK,AttributeType=S \
			AttributeName=GSI1PK,AttributeType=S \
			AttributeName=GSI1SK,AttributeType=S \
		--key-schema \
			AttributeName=PK,KeyType=HASH \
			AttributeName=SK,KeyType=RANGE \
		--global-secondary-indexes \
			'[{"IndexName": "GSI1", \
			  "KeySchema": [{"AttributeName":"GSI1PK","KeyType":"HASH"}, \
							{"AttributeName":"GSI1SK","KeyType":"RANGE"}], \
			  "Projection":{"ProjectionType":"ALL"}, \
			  "ProvisionedThroughput": {"ReadCapacityUnits": 5, "WriteCapacityUnits": 5}}]' \
		--provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
		--profile $(AWS_PROFILE) \
		--endpoint-url $(AWS_ENDPOINT_URL) > /dev/null 2>&1 || echo "Table '$(DYNAMODB_TABLE_NAME)' already exists or failed to create."
	@echo "Table creation command executed."

delete-table:
	@echo "Deleting DynamoDB table '$(DYNAMODB_TABLE_NAME)' locally..."
	@echo "Using AWS_PROFILE=$(AWS_PROFILE) and AWS_ENDPOINT_URL=$(AWS_ENDPOINT_URL)"
	@aws dynamodb delete-table \
		--table-name $(DYNAMODB_TABLE_NAME) \
		--profile $(AWS_PROFILE) \
		--endpoint-url $(AWS_ENDPOINT_URL) > /dev/null 2>&1 || echo "Table '$(DYNAMODB_TABLE_NAME)' not found or failed to delete."
	@echo "Table deletion command executed."

# Code Generation
gen:
	@echo "Generating Go code from OpenAPI spec (api/openapi.yaml)..."
	@if ! command -v $(OAPI_CODEGEN_CMD) > /dev/null 2>&1; then \
		echo "Error: $(OAPI_CODEGEN_CMD) command not found."; \
		echo "Please install it: go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest"; \
		exit 1; \
	fi
	$(OAPI_CODEGEN_CMD) -generate types,server -package api -o internal/api/api.gen.go api/openapi.yaml
	@echo "Code generation complete: internal/api/api.gen.go"

# Lint the code
lint:
	@echo "Linting code..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "Warning: golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run ./...

# Format the code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	@echo "Calculating coverage..."
	go tool cover -func=coverage.out
