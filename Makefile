.PHONY: all build run run-tui run-interactive test clean

APP_NAME := llm_adventure
CMD_DIR := ./cmd/llm_adventure

all: build test

build:
	@echo "Building $(APP_NAME)..."
	@go build -o $(APP_NAME) $(CMD_DIR)

run: run-tui

run-tui:
	@echo "Running TUI with LLM..."
	@go run $(CMD_DIR)/main.go tui-llm

run-interactive:
	@echo "Running Interactive mode..."
	@go run $(CMD_DIR)/main.go interactive

test:
	@echo "Running tests..."
	@go test -v ./...

clean:
	@echo "Cleaning up..."
	@rm -f $(APP_NAME)
	@rm -f savegame.json
