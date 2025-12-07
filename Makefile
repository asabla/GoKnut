APP_NAME ?= goknut
BIN_DIR ?= bin
DIST_DIR ?= dist
BINARY := $(BIN_DIR)/$(APP_NAME)
GO_MAIN := ./cmd/server
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

.PHONY: build run test publish clean

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BINARY) $(GO_MAIN)

run:
	go run $(GO_MAIN)

test:
	go test ./...

publish:
	mkdir -p $(DIST_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go build -o $(DIST_DIR)/$(APP_NAME)-$(GOOS)-$(GOARCH) $(GO_MAIN)

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)
