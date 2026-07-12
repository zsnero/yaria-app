BINARY = yaria-app
BUILD_DIR = build/bin
# Linux needs webkit2_41; Windows/macOS use WebView2 / WKWebView (no webkit tag)
TAGS_FREE = webkit2_41,desktop,production
TAGS_PRO = webkit2_41,desktop,production,pro
TAGS_CROSS_FREE = desktop,production
TAGS_CROSS_PRO = desktop,production,pro
LDFLAGS = -s -w
LDFLAGS_WIN = -s -w -H windowsgui
MINGW_CC = x86_64-w64-mingw32-gcc

.PHONY: build build-pro run dev dev-pro clean tidy \
	build-linux build-linux-pro build-windows build-windows-pro \
	build-darwin build-darwin-pro build-all build-all-pro

# Free build (Yaria only, opensource) — host platform
build:
	@mkdir -p $(BUILD_DIR)
	go build -tags "$(TAGS_FREE)" -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)

# Pro build (Yaria + Mantorex) - run 'go mod tidy' first if deps are missing
build-pro:
	@mkdir -p $(BUILD_DIR)
	go mod tidy
	go build -tags "$(TAGS_PRO)" -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)

# Development builds
dev:
	@mkdir -p $(BUILD_DIR)
	go build -tags "webkit2_41,dev" -gcflags "all=-N -l" -o $(BUILD_DIR)/$(BINARY)
	$(BUILD_DIR)/$(BINARY)

dev-pro:
	@mkdir -p $(BUILD_DIR)
	go build -tags "webkit2_41,dev,pro" -gcflags "all=-N -l" -o $(BUILD_DIR)/$(BINARY)
	$(BUILD_DIR)/$(BINARY)

# Run existing build
run:
	$(BUILD_DIR)/$(BINARY)

# --- Linux ---
build-linux:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags "$(TAGS_FREE)" -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64

build-linux-pro:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags "$(TAGS_PRO)" -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64

# --- Windows (requires: gcc-mingw-w64-x86-64; WebView2 runtime on target) ---
build-windows:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 CC=$(MINGW_CC) GOOS=windows GOARCH=amd64 go build -tags "$(TAGS_CROSS_FREE)" -ldflags "$(LDFLAGS_WIN)" -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe

build-windows-pro:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 CC=$(MINGW_CC) GOOS=windows GOARCH=amd64 go build -tags "$(TAGS_CROSS_PRO)" -ldflags "$(LDFLAGS_WIN)" -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe

# --- macOS (best built on a Mac; CGO cross-compile needs osxcross) ---
build-darwin:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -tags "$(TAGS_CROSS_FREE)" -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -tags "$(TAGS_CROSS_FREE)" -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64

build-darwin-pro:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -tags "$(TAGS_CROSS_PRO)" -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -tags "$(TAGS_CROSS_PRO)" -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64

# Build all platforms (free / pro)
build-all: build-linux build-windows
build-all-pro: build-linux-pro build-windows-pro

clean:
	rm -rf $(BUILD_DIR)
	go clean

tidy:
	go mod tidy
