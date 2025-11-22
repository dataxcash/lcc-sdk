.PHONY: all build test clean install demo run-demo test-demo help

all: build

help:
	@echo "LCC SDK Makefile Commands:"
	@echo ""
	@echo "  make build       - Build lcc-codegen and lcc-sdk binaries"
	@echo "  make test        - Run all unit tests"
	@echo "  make demo        - Build zero-intrusion demo"
	@echo "  make run-demo    - Build and run demo (requires LCC server)"
	@echo "  make test-demo   - Run standalone demo test (no LCC server)"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make install     - Install binaries"
	@echo "  make fmt         - Format code"
	@echo "  make lint        - Run linter"
	@echo "  make deps        - Download and tidy dependencies"
	@echo ""

build:
	@echo "Building lcc-sdk..."
	@go build -o bin/lcc-codegen ./cmd/lcc-codegen
	@go build -o bin/lcc-sdk ./cmd/lcc-sdk

test:
	@echo "Running tests..."
	@go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

install:
	@echo "Installing..."
	@go install ./cmd/lcc-codegen
	@go install ./cmd/lcc-sdk

fmt:
	@echo "Formatting code..."
	@go fmt ./...

lint:
	@echo "Linting..."
	@golangci-lint run ./...

deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

demo:
	@echo "Building zero-intrusion demo..."
	@cd examples/zero-intrusion && go build -o demo main.go
	@echo "Demo built: examples/zero-intrusion/demo"
	@echo "Run with: cd examples/zero-intrusion && ./demo"

run-demo: demo
	@echo "Running zero-intrusion demo..."
	@cd examples/zero-intrusion && ./demo

test-demo:
	@echo "Building standalone test..."
	@cd examples/zero-intrusion && go build -o test_standalone test_standalone.go
	@echo "Running standalone test (no LCC server required)..."
	@cd examples/zero-intrusion && ./test_standalone
