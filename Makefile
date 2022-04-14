SHELL=bash

BUILD=build
BUILD_ARCH=$(BUILD)/$(GOOS)-$(GOARCH)
BIN_DIR?=.

export GOOS?=$(shell go env GOOS)
export GOARCH?=$(shell go env GOARCH)

BUILD_TIME=$(shell date +%s)
GIT_COMMIT=$(shell git rev-parse HEAD)
VERSION ?= $(shell git tag --points-at HEAD | grep ^v | head -n 1)
LDFLAGS=-ldflags "-w -s -X 'main.Version=${VERSION}' -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.GitCommit=$(GIT_COMMIT)'"

all: audit test build

audit:
	go list -m all | nancy sleuth --exclude-vulnerability-file ./.nancy-ignore

build:
	@mkdir -p $(BUILD_ARCH)/$(BIN_DIR)
	go build $(LDFLAGS) -o $(BUILD_ARCH)/$(BIN_DIR)/dp-deployer cmd/dp-deployer/main.go

generate:
	go generate ./...

debug: build
	HUMAN_LOG=1 go run $(LDFLAGS) -race cmd/dp-deployer/main.go

test:
	go test -cover -race ./...

.PHONY: all audit build debug test generate
