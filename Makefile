.PHONY: build install test clean

BINARY    = kai
BUILD_DIR = ./cmd/kai
MODULE    = github.com/tinhvqbk/kai

VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS = -X $(MODULE).Version=$(VERSION) \
          -X $(MODULE).Commit=$(COMMIT) \
          -X $(MODULE).BuildDate=$(BUILD_DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(BUILD_DIR)

install:
	go install -ldflags "$(LDFLAGS)" $(BUILD_DIR)

test:
	go test ./... -v

clean:
	rm -f $(BINARY)
