#!/usr/bin/make -f

COMMIT := $(shell git log -1 --format='%H')
VERSION := 0.2 # $(shell echo $(shell git describe --tags) | sed 's/^v//')

ldflags = -X main.AppName=trustless-api \
		  -X main.Version=$(VERSION) \
		  -X main.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(ldflags)' -trimpath -mod=readonly

.PHONY: build format lint release

all: format lint build

###############################################################################
###                                  Build                                  ###
###############################################################################

build:
	@echo "🤖 Building Trustless-API ..."
	@go build $(BUILD_FLAGS) -o "$(PWD)/build/" ./cmd/trustless-api
	@echo "✅ Completed build!"

###############################################################################
###                          Formatting & Linting                           ###
###############################################################################

format:
	@echo "🤖 Running formatter..."
	@gofmt -l -w .
	@echo "✅ Completed formatting!"

lint:
	@echo "🤖 Running linter..."
	@golangci-lint run --timeout=10m
	@echo "✅ Completed linting!"

release:
	@echo "🤖 Creating Trustless-API releases..."
	@rm -rf release
	@mkdir -p release

	@GOOS=darwin CGO_ENABLED=1 GOARCH=arm64 go build $(BUILD_FLAGS) ./cmd/trustless-api
	@tar -czf release/trustless-api_darwin_arm64.tar.gz trustless-api
	@shasum -a 256 release/trustless-api_darwin_arm64.tar.gz >> release/release_checksum

	@GOOS=linux CGO_ENABLED=1 GOARCH=amd64 go build $(BUILD_FLAGS) ./cmd/trustless-api
	@tar -czf release/trustless-api_linux_amd64.tar.gz trustless-api
	@shasum -a 256 release/trustless-api_linux_amd64.tar.gz >> release/release_checksum

	@rm trustless-api
	@echo "✅ Completed release creation!"