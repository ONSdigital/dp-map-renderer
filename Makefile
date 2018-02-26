SHELL=bash
MAIN=dp-map-renderer

BUILD_DIR=build
BUILD_ARCH=$(GOOS)-$(GOARCH)

BIN_DIR ?= $(BUILD_DIR)/$(BUILD_ARCH)

export GOOS=$(shell go env GOOS)
export GOARCH=$(shell go env GOARCH)

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/dp-map-renderer cmd/$(MAIN)/main.go

debug: build
	HUMAN_LOG=1 go run -race cmd/$(MAIN)/main.go

test:
	go test -cover $(shell go list ./... | grep -v /vendor/)

.PHONY: build debug test
