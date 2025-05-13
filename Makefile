.PHONY: build clean test fmt vet install

# Build the application
build:
	go build -o eolas

# Clean build artifacts
clean:
	rm -f eolas

# Run tests
test:
	go test ./...

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Install the application
install:
	go install

# Run all quality checks
check: fmt vet test

# Run the application
run: build
	./eolas

# Help message
help:
	@echo "Available targets:"
	@echo "  build    - Build the application"
	@echo "  clean    - Remove build artifacts"
	@echo "  test     - Run tests"
	@echo "  fmt      - Format code"
	@echo "  vet      - Run go vet"
	@echo "  install  - Install the application"
	@echo "  check    - Run all quality checks (fmt, vet, test)"
	@echo "  run      - Build and run the application"
	@echo "  help     - Show this help message"