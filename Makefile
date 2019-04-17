APP = landscaper
FOLDERS = ./cmd/... ./pkg/...

BUILD_DIR ?= build

# get version info from git's tags
GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_TAG := $(shell git describe --tags --dirty 2>/dev/null)
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null)

# inject version info into version vars
LD_RELEASE_FLAGS += -X github.com/eneco/landscaper/pkg/landscaper.GitCommit=${GIT_COMMIT}
LD_RELEASE_FLAGS += -X github.com/eneco/landscaper/pkg/landscaper.GitTag=${GIT_TAG}
LD_RELEASE_FLAGS += -X github.com/eneco/landscaper/pkg/landscaper.SemVer=${VERSION}

.PHONY: default bootstrap clean test build static docker

default: build

bootstrap:
	dep ensure -v -vendor-only

clean:
	rm -rf $(BUILD_DIR)
	rm -rf vendor

clean-vendor:
	rm -rf ./vendor > /dev/null

test:
	go test -cover $(FOLDERS)

build:
	cd cmd && go build -ldflags "$(LD_RELEASE_FLAGS)" -o ../$(BUILD_DIR)/$(APP); cd ..

# builds a statically linked binary for linux-amd64
dockerbinary:
	cd cmd && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LD_RELEASE_FLAGS)" -o ../$(BUILD_DIR)/$(APP); cd ..

docker: dockerbinary
	cp docker/* ./$(BUILD_DIR)/
	docker build -t eneco/landscaper ./$(BUILD_DIR)/
	docker run eneco/landscaper landscaper version
	docker tag eneco/landscaper eneco/landscaper:latest
	docker tag eneco/landscaper eneco/landscaper:$(GIT_TAG)

publish_docker: docker
	docker push eneco/landscaper:latest
	docker push eneco/landscaper:$(GIT_TAG)

publish_github:
	go get github.com/goreleaser/goreleaser
	./scripts/goreleaser.yaml.sh "$(LD_RELEASE_FLAGS)" >/tmp/gorel.yaml
	goreleaser --config /tmp/gorel.yaml
