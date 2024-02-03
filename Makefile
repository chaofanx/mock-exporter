BINARY_BASE_NAME := mock-exporter
VERSION := 0.2.0
REVISION := $(shell git rev-parse --short HEAD)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
BUILD_USER := $(shell whoami)@$(shell hostname)
BUILD_DATE := $(shell date +%Y%m%d-%H:%M:%S)

OS_LIST = windows linux darwin
ARCH_LIST = amd64

define build_for_os
	GOOS=$(1) GOARCH=$(2) go build \
	-ldflags "-X github.com/prometheus/common/version.Version=$(VERSION) -X github.com/prometheus/common/version.Revision=$(REVISION) -X github.com/prometheus/common/version.Branch=$(BRANCH) -X github.com/prometheus/common/version.BuildUser=$(BUILD_USER) -X github.com/prometheus/common/version.BuildDate=$(BUILD_DATE)" \
	-o $(BINARY_BASE_NAME)_$(1)_$(2) -v
endef

.PHONY: all build-cross

all: build

build-cross:
	$(foreach os,$(OS_LIST),\
		$(foreach arch,$(ARCH_LIST),\
			$(call build_for_os,$(os),$(arch))))

build:
	go build -ldflags "-X github.com/prometheus/common/version.Version=$(VERSION) -X github.com/prometheus/common/version.Revision=$(REVISION) -X github.com/prometheus/common/version.Branch=$(BRANCH) -X github.com/prometheus/common/version.BuildUser=$(BUILD_USER) -X github.com/prometheus/common/version.BuildDate=$(BUILD_DATE)" -o $(BINARY_BASE_NAME) -v

clean:
	rm -f $(BINARY_NAME)

run:
	go run main.go

test:
	go test -v ./...
