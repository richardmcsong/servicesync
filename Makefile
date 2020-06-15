CGO_ENABLED=0
GOOS=linux
GOARCH=amd64
PACKAGE=github.com/richardmcsong/servicesync
GCR_LOCATION=richardmcsong/servicesync
VERSION=$(shell git describe --tags $(shell git rev-list --tags --max-count=1))
BUILD_FLAGS=-ldflags "-X github.com/richardmcsong/servicesync/pkg/config.Version=$(VERSION)"
WHOAMI=$(shell whoami)
SHA=$(shell git --no-pager log --pretty=format:'%h' -n 1)
.PHONY=build
build:
	@mkdir -p bin/
	go build -o bin/ ./cmd/servicesync

.PHONY=test
test:
	@mkdir -p coverage
	go test -coverprofile coverage/cover.html $(PACKAGE)/pkg/...

.PHONY=container
container:
	docker build -f ./Dockerfile -t $(GCR_LOCATION):$(VERSION).$(WHOAMI)-dev.$(SHA) \
	--build-arg CGO_ENABLED=$(CGO_ENABLED) \
	--build-arg GOOS=$(GOOS) \
	--build-arg GOARCH=$(GOARCH) \
	--build-arg VERSION=$(VERSION) .

.PHONY=container-push
container-push:
	docker push $(GCR_LOCATION):$(VERSION).$(WHOAMI)-dev.$(SHA)
