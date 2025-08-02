BINARY_NAME=marzban-tg-bot
BUILD_DIR=build

.PHONY: me-cum edging gooon help

me-cum:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/main.go

edging:
	go mod download

gooon:
	rm -rf $(BUILD_DIR)

help:
	@echo "Available commands:"
	@echo "    me-cum       - makes source cum"
	@echo "    edging       - creates environment required for cum"
	@echo "    gooon        - cleans up"
