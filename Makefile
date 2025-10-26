# Makefile

# 可执行文件名称
BINARY_NAME := traffic-tester-exec

# 架构和操作系统
GOOS := linux
GOARCH := amd64

# 输出目录
BIN_DIR := ./bin

# Go build flags
LDFLAGS := -s -w

.PHONY: all build clean

all: build

build:
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BIN_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) main.go
	@echo "Build finished: $(BIN_DIR)/$(BINARY_NAME)"

clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)