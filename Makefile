APP = landscaper
FOLDERS = ./cmd/... ./pkg/...

BUILD_DIR ?= build

# get version info from git's tags
GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_TAG := $(shell git describe --tags --long --dirty 2>/dev/null)
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null)

# inject version info into version vars
LD_RELEASE_FLAGS += -X github.com/eneco/landscaper/pkg/landscaper.GitCommit=${GIT_COMMIT}
LD_RELEASE_FLAGS += -X github.com/eneco/landscaper/pkg/landscaper.GitTag=${GIT_TAG}
LD_RELEASE_FLAGS += -X github.com/eneco/landscaper/pkg/landscaper.SemVer=${VERSION}

.PHONY: default bootstrap clean test build

default: build

bootstrap:
	glide install -v

clean:
	rm -rf $(BUILD_DIR)

test:
	go test -cover $(FOLDERS)

build:
	cd cmd && go build -ldflags "$(LD_RELEASE_FLAGS)" -o ../$(BUILD_DIR)/$(APP); cd ..
