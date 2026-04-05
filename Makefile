BINARY_NAME=pprof-analyzer
CMD_PATH=./cmd/pprof-analyzer
BUILD_DIR=./build

.PHONY: build test lint run clean tidy

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

test:
	@echo "Running tests..."
	go test ./... -v -race -timeout 120s

lint:
	@echo "Running linter..."
	golangci-lint run ./...

run:
	go run $(CMD_PATH)

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

tidy:
	go mod tidy
