# Variables
BINARY_NAME=yaml-checker
CMD_PATH=.
BUILD_DIR=bin
VERSION ?= $(shell git describe --tags --always --dirty)

# Default target
all: build

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_PATH)

clean:
	rm -rf $(BUILD_DIR)

build-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_PATH)

# Updated buildx-based Docker image build
docker-build:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--push \
		--tag wang/yaml-checker:$(VERSION) \
		--tag wang/yaml-checker:latest \
		.
