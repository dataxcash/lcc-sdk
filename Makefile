.PHONY: all build test clean install

all: build

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
