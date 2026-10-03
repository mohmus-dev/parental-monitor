# ============================================================
# Parental Monitor CLI - Makefile
# ============================================================

# Go settings
GO        := go
BINARY    := parental-monitor
BINARY_EXE := parental-monitor.exe
BUILD_DIR := build
VERSION   := 1.0.0
LDFLAGS   := -ldflags "-s -w -X main.version=$(VERSION)"

# Colors for pretty output (use printf, not echo!)
GREEN  := \033[0;32m
YELLOW := \033[0;33m
RED    := \033[0;31m
CYAN   := \033[0;36m
BOLD   := \033[1m
NC     := \033[0m

# ============================================================
# Default target
# ============================================================
.PHONY: all
all: build

# ============================================================
# Build for current OS
# ============================================================
.PHONY: build
build:
	@mkdir -p $(BUILD_DIR)
	@printf "$(GREEN)$(BOLD)[BUILD]$(NC) Building for current OS...\n"
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) .
	@printf "$(GREEN)[OK]$(NC) Build complete: $(BUILD_DIR)/$(BINARY)\n"

# ============================================================
# Build for Windows
# ============================================================
.PHONY: build-windows
build-windows:
	@mkdir -p $(BUILD_DIR)
	@printf "$(GREEN)$(BOLD)[BUILD]$(NC) Building for Windows (amd64)...\n"
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_EXE) .
	@printf "$(GREEN)[OK]$(NC) Windows build complete: $(BUILD_DIR)/$(BINARY_EXE)\n"

# ============================================================
# Build for Linux
# ============================================================
.PHONY: build-linux
build-linux:
	@mkdir -p $(BUILD_DIR)
	@printf "$(GREEN)$(BOLD)[BUILD]$(NC) Building for Linux (amd64)...\n"
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux .
	@printf "$(GREEN)[OK]$(NC) Linux build complete: $(BUILD_DIR)/$(BINARY)-linux\n"

# ============================================================
# Build for macOS (Intel & Apple Silicon)
# ============================================================
.PHONY: build-mac
build-mac:
	@mkdir -p $(BUILD_DIR)
	@printf "$(GREEN)$(BOLD)[BUILD]$(NC) Building for macOS (amd64)...\n"
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-mac-amd64 .
	@printf "$(GREEN)$(BOLD)[BUILD]$(NC) Building for macOS (arm64)...\n"
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-mac-arm64 .
	@printf "$(GREEN)[OK]$(NC) macOS builds complete\n"

# ============================================================
# Build for ALL platforms
# ============================================================
.PHONY: build-all
build-all: build-windows build-linux build-mac
	@printf "$(GREEN)[OK]$(NC) All platform builds complete!\n"

# ============================================================
# Run in foreground mode
# ============================================================
.PHONY: run
run: build
	@printf "$(CYAN)$(BOLD)[RUN]$(NC) Running in foreground mode...\n"
	$(BUILD_DIR)/$(BINARY) -env .env

# ============================================================
# Run as background daemon
# ============================================================
.PHONY: run-daemon
run-daemon: build
	@printf "$(CYAN)$(BOLD)[RUN]$(NC) Starting as background daemon...\n"
	$(BUILD_DIR)/$(BINARY) -env .env -daemon

# ============================================================
# Stop the daemon
# ============================================================
.PHONY: stop
stop:
	@printf "$(YELLOW)$(BOLD)[STOP]$(NC) Stopping daemon...\n"
	$(BUILD_DIR)/$(BINARY) -stop

# ============================================================
# Run tests
# ============================================================
.PHONY: test
test:
	@printf "$(GREEN)$(BOLD)[TEST]$(NC) Running tests...\n"
	$(GO) test ./... -v -count=1

# ============================================================
# Run tests with coverage
# ============================================================
.PHONY: coverage
coverage:
	@printf "$(GREEN)$(BOLD)[TEST]$(NC) Running tests with coverage...\n"
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@printf "$(GREEN)[OK]$(NC) Coverage report: coverage.html\n"

# ============================================================
# Run linter
# ============================================================
.PHONY: lint
lint:
	@printf "$(GREEN)$(BOLD)[LINT]$(NC) Running linter...\n"
	@command -v golangci-lint >/dev/null 2>&1 || (printf "$(RED)[ERROR]$(NC) golangci-lint not installed. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest\n" && exit 1)
	golangci-lint run ./...

# ============================================================
# Format code
# ============================================================
.PHONY: fmt
fmt:
	@printf "$(GREEN)$(BOLD)[FMT]$(NC) Formatting code...\n"
	$(GO) fmt ./...

# ============================================================
# Vet code
# ============================================================
.PHONY: vet
vet:
	@printf "$(GREEN)$(BOLD)[VET]$(NC) Running go vet...\n"
	$(GO) vet ./...

# ============================================================
# Tidy dependencies
# ============================================================
.PHONY: tidy
tidy:
	@printf "$(GREEN)$(BOLD)[TIDY]$(NC) Tidying go.mod...\n"
	$(GO) mod tidy

# ============================================================
# Clean build artifacts
# ============================================================
.PHONY: clean
clean:
	@printf "$(YELLOW)$(BOLD)[CLEAN]$(NC) Cleaning build artifacts...\n"
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	rm -f monitor.pid monitor.db daemon.log
	@printf "$(GREEN)[OK]$(NC) Clean complete\n"

# ============================================================
# Install dependencies
# ============================================================
.PHONY: deps
deps:
	@printf "$(GREEN)$(BOLD)[DEPS]$(NC) Downloading dependencies...\n"
	$(GO) mod download

# ============================================================
# Show help
# ============================================================
.PHONY: help
help:
	@printf "$(GREEN)Parental Monitor CLI - Make Commands:$(NC)\n"
	@printf "  $(YELLOW)make build$(NC)          Build for current OS\n"
	@printf "  $(YELLOW)make build-windows$(NC)  Build Windows binary\n"
	@printf "  $(YELLOW)make build-linux$(NC)    Build Linux binary\n"
	@printf "  $(YELLOW)make build-mac$(NC)      Build macOS binaries\n"
	@printf "  $(YELLOW)make build-all$(NC)      Build for all platforms\n"
	@printf "  $(YELLOW)make run$(NC)            Run in foreground\n"
	@printf "  $(YELLOW)make run-daemon$(NC)     Run as background daemon\n"
	@printf "  $(YELLOW)make stop$(NC)           Stop the daemon\n"
	@printf "  $(YELLOW)make test$(NC)           Run tests\n"
	@printf "  $(YELLOW)make coverage$(NC)       Run tests with coverage report\n"
	@printf "  $(YELLOW)make lint$(NC)           Run linter\n"
	@printf "  $(YELLOW)make fmt$(NC)            Format code\n"
	@printf "  $(YELLOW)make vet$(NC)            Run go vet\n"
	@printf "  $(YELLOW)make tidy$(NC)           Tidy go.mod\n"
	@printf "  $(YELLOW)make deps$(NC)           Download dependencies\n"
	@printf "  $(YELLOW)make clean$(NC)          Clean build artifacts\n"