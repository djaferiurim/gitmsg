BINARY := gitmsg
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.Version=$(VERSION)

.PHONY: build test lint install clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

test:
	go test ./...

lint:
	go vet ./...

install:
	go install -ldflags "$(LDFLAGS)" .

clean:
	rm -rf bin
