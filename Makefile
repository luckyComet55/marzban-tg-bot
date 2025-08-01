BINARY_NAME=marzban-tg-bot
BUILD_DIR=build

.PHONY: build clean setup help

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/main.go

setup:
	go mod download

clean:
	rm -rf $(BUILD_DIR)

help:
	@echo "Available commands:"
	@echo "    build        - builds Marzban client binary"
	@echo "    setup        - creates environment required for build"
	@echo "    clean        - cleans build artifacts"
