SHELL := /bin/sh

APP_NAME := geo-acq
SIM_NAME := simul-gps
SOUNDER_SIM_NAME := simul-echosounder
EXPORT_NAME := geo-export
MAIN_PKG := ./cmd/geo-acq
SIM_PKG := ./cmd/simul/gps
SOUNDER_SIM_PKG := ./cmd/simul/echosounder
EXPORT_PKG := ./cmd/export
BIN_DIR := bin
DIST_DIR := dist
GO ?= go

PLATFORMS ?= windows/amd64 linux/amd64 linux/arm64 darwin/amd64

GOCACHE ?= $(CURDIR)/.gocache
GOMODCACHE ?= $(CURDIR)/.gomodcache
GOENV = GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE)
LDFLAGS ?=

ifeq ($(OS),Windows_NT)
APP_BIN := $(APP_NAME).exe
SIM_BIN := $(SIM_NAME).exe
SOUNDER_SIM_BIN := $(SOUNDER_SIM_NAME).exe
EXPORT_BIN := $(EXPORT_NAME).exe
else
APP_BIN := $(APP_NAME)
SIM_BIN := $(SIM_NAME)
SOUNDER_SIM_BIN := $(SOUNDER_SIM_NAME)
EXPORT_BIN := $(EXPORT_NAME)
endif

.DEFAULT_GOAL := help

.PHONY: help fmt test build build-all build-sim build-sim-sounder build-export run cross-build clean copy

help:
	@printf "%s\n" \
		"Targets:" \
		"  make build         Build $(APP_NAME) in $(BIN_DIR)/" \
		"  make build-all     Build all binaries in $(BIN_DIR)/" \
		"  make build-sim     Build the GPS simulator in $(BIN_DIR)/" \
		"  make build-sim-sounder Build the echosounder simulator in $(BIN_DIR)/" \
		"  make build-export  Build the export tool in $(BIN_DIR)/" \
		"  make test          Run go test ./..." \
		"  make fmt           Run gofmt on the repository" \
		"  make cross-build   Build release binaries in $(DIST_DIR)/" \
		"  make run           Build and run $(APP_NAME)" \
		"  make copy DEST=/path/to/dir   Copy release artifacts and TOML files" \
		"  make clean         Remove build outputs and local Go caches"

fmt:
	$(GOENV) $(GO) fmt ./...

test:
	$(GOENV) $(GO) test ./...

build:
	mkdir -p $(BIN_DIR)
	$(GOENV) $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_BIN) $(MAIN_PKG)

build-all: build build-sim build-sim-sounder build-export

build-sim:
	mkdir -p $(BIN_DIR)
	$(GOENV) $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(SIM_BIN) $(SIM_PKG)

build-sim-sounder:
	mkdir -p $(BIN_DIR)
	$(GOENV) $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(SOUNDER_SIM_BIN) $(SOUNDER_SIM_PKG)

build-export:
	mkdir -p $(BIN_DIR)
	$(GOENV) $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(EXPORT_BIN) $(EXPORT_PKG)

run: build
	./$(BIN_DIR)/$(APP_BIN)

cross-build:
	mkdir -p $(DIST_DIR)
	@for target in $(PLATFORMS); do \
		os=$${target%/*}; \
		arch=$${target#*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		echo ">> building $(APP_NAME) for $$os/$$arch"; \
		GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -ldflags "$(LDFLAGS)" \
			-o $(DIST_DIR)/$(APP_NAME)-$$os-$$arch$$ext $(MAIN_PKG) || exit 1; \
	done

copy: cross-build
ifndef DEST
	$(error DEST is required, for example: make copy DEST=/tmp/release)
endif
	mkdir -p $(DEST)
	cp $(DIST_DIR)/$(APP_NAME)-* $(DEST)/
	cp ./*.toml $(DEST)/

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR) $(GOCACHE) $(GOMODCACHE)
