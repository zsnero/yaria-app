BINARY = yaria-app
BUILD_DIR = build/bin
TAGS_FREE = webkit2_41,desktop,production
TAGS_PRO = webkit2_41,desktop,production,pro
TAGS_WIN_FREE = desktop,production
TAGS_WIN_PRO = desktop,production,pro
LDFLAGS = -s -w
LDFLAGS_WIN = -s -w -H windowsgui
MINGW_CC = x86_64-w64-mingw32-gcc

.PHONY: build build-pro run dev dev-pro clean tidy build-windows build-windows-pro

# Free build (Yaria only, opensource)
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

# Cross-compilation (free)
build-linux:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags "$(TAGS_FREE)" -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64

# Cross-compilation (pro)
build-linux-pro:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags "$(TAGS_PRO)" -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64

# Cross-compilation Windows (free) - requires: sudo apt install gcc-mingw-w64-x86-64
build-windows:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 CC=$(MINGW_CC) GOOS=windows GOARCH=amd64 go build -tags "$(TAGS_WIN_FREE)" -ldflags "$(LDFLAGS_WIN)" -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe

# Cross-compilation Windows (pro)
build-windows-pro:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 CC=$(MINGW_CC) GOOS=windows GOARCH=amd64 go build -tags "$(TAGS_WIN_PRO)" -ldflags "$(LDFLAGS_WIN)" -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe

# Build all platforms (free)
build-all: build-linux build-windows

# Build all platforms (pro)
build-all-pro: build-linux-pro build-windows-pro

clean:
	rm -rf $(BUILD_DIR)
	go clean

tidy:
	go mod tidy
