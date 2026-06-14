# ─── Sunken Sunflower — Makefile ─────────────────────────────────────
# Convenience commands for development and release.

VERSION ?= $(shell date +%Y.%m.%d)
BINARY  ?= sunken-sunflower

# Detect OS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
	EXT :=
endif
ifeq ($(UNAME_S),Darwin)
	EXT :=
endif
ifeq ($(UNAME_S),MINGW32_NT-6.1)
	EXT := .exe
endif

.PHONY: all build run test clean dist release help

all: test build

# ─── Development ────────────────────────────────────────────────────

build:  ## Build for current platform
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY)$(EXT) .

run:    ## Run the game
	go run .

test:   ## Run all tests
	go test ./... -v -count=1

vet:    ## Run static analysis
	go vet ./...

clean:  ## Remove build artifacts
	rm -rf $(BINARY) $(BINARY)$(EXT) dist/ steam-output/

# ─── Steam Deck ─────────────────────────────────────────────────────

steamdeck: build  ## Build optimized for Steam Deck
	@echo "Running on Steam Deck with default settings."

# ─── Release ────────────────────────────────────────────────────────

dist:  ## Build release artifacts for all platforms
	./scripts/build-all.sh $(VERSION)

release: dist  ## Package for Steam upload
	@echo ""
	@echo "✦ Steam Upload Ready!"
	@echo "  Upload artifacts from dist/ to Steamworks."
	@echo "  Or use steamcmd with steam/steambuild.vdf:"
	@echo "    steamcmd +run_app_build steam/steambuild.vdf"
	@echo ""

# ─── Cross-compile helpers ──────────────────────────────────────────

build-linux:    ## Cross-compile for Linux amd64
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY)-linux-$(VERSION) .

build-windows:  ## Cross-compile for Windows amd64
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY)-windows-$(VERSION).exe .

build-macos:    ## Cross-compile for macOS amd64
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY)-macos-$(VERSION) .

build-macos-m1: ## Cross-compile for macOS arm64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BINARY)-macos-m1-$(VERSION) .

# ─── Help ───────────────────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'