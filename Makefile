BINARY_NAME=lcd-demo
LDFLAGS=-ldflags="-s -w"

# Default target - build for current architecture
.DEFAULT_GOAL := build

.PHONY: build build-all clean arm5 arm6 arm7 arm64

# Build for current architecture
build:
	@echo "Building for current architecture..."
	go build -o $(BINARY_NAME) $(LDFLAGS) ./main.go

# Build for ARMv5
arm5:
	@echo "Building for ARMv5..."
	GOOS=linux GOARCH=arm GOARM=5 go build -o $(BINARY_NAME)-armv5 $(LDFLAGS) ./main.go

# Build for ARMv6
arm6:
	@echo "Building for ARMv6..."
	GOOS=linux GOARCH=arm GOARM=6 go build -o $(BINARY_NAME)-armv6 $(LDFLAGS) ./main.go

# Build for ARMv7
arm7:
	@echo "Building for ARMv7..."
	GOOS=linux GOARCH=arm GOARM=7 go build -o $(BINARY_NAME)-armv7 $(LDFLAGS) ./main.go

# Build for ARM64
arm64:
	@echo "Building for ARM64..."
	GOOS=linux GOARCH=arm64 go build -o $(BINARY_NAME)-arm64 $(LDFLAGS) ./main.go

# Build for all architectures
build-all: arm5 arm6 arm7 arm64
	@echo "Build complete for all architectures!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME) $(BINARY_NAME)-armv5 $(BINARY_NAME)-armv6 $(BINARY_NAME)-armv7 $(BINARY_NAME)-arm64