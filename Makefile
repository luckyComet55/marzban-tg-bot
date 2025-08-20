BINARY_NAME=marzban-tg-bot
BUILD_DIR=build

.PHONY: build setup clean help

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/main.go

setup:
	go mod download

clean:
	rm -rf $(BUILD_DIR)

help:
	@echo "Available commands:"
	@echo "    build        - builds source"
	@echo "    setup        - creates environment"
	@echo "    clean        - cleans up"
