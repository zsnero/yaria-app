BINARY = yaria-app
BUILD_DIR = build/bin
TAGS_FREE = webkit2_41,desktop,production
TAGS_PRO = webkit2_41,desktop,production,pro
LDFLAGS = -s -w

.PHONY: build build-pro run dev dev-pro clean tidy

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

clean:
	rm -rf $(BUILD_DIR)
	go clean

tidy:
	go mod tidy
