.PHONY: build install clean test lint release

BINARY_NAME=claude-coord
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_FLAGS=-ldflags "-X main.Version=$(VERSION)"

build:
	go mod tidy
	go build $(BUILD_FLAGS) -o bin/$(BINARY_NAME) ./cmd/claude-coord

install: build
	cp bin/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installed to /usr/local/bin/$(BINARY_NAME)"

install-local: build
	@echo "Binary available at: $(CURDIR)/bin/$(BINARY_NAME)"
	@echo "Add to PATH with: export PATH=\"\$$PATH:$(CURDIR)/bin\""

clean:
	rm -rf bin/
	rm -rf dist/

test:
	go test -v ./...

lint:
	golangci-lint run

# Cross-platform builds
release:
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/claude-coord
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/claude-coord
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/claude-coord
	GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/claude-coord
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/claude-coord

# Development helpers
dev: build
	./bin/$(BINARY_NAME) status

demo: build
	@echo "=== Demo: Initialize ==="
	./bin/$(BINARY_NAME) init --force
	@echo ""
	@echo "=== Demo: Status (empty) ==="
	./bin/$(BINARY_NAME) status
	@echo ""
	@echo "=== Demo: Acquire lock ==="
	./bin/$(BINARY_NAME) lock "db/schema/*" --op "Adding email column" --agent demo-agent-1
	@echo ""
	@echo "=== Demo: Status (with lock) ==="
	./bin/$(BINARY_NAME) status
	@echo ""
	@echo "=== Demo: Try to acquire same lock ==="
	-./bin/$(BINARY_NAME) lock "db/schema/*" --op "Adding oauth" --agent demo-agent-2
	@echo ""
	@echo "=== Demo: Release lock ==="
	./bin/$(BINARY_NAME) unlock "db/schema/*" --agent demo-agent-1
	@echo ""
	@echo "=== Demo: Cleanup ==="
	rm -rf .claude-coord
