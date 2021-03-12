.PHONY: lint test static install uninstall cross
VERSION := $(shell git describe --tags --dirty --always)
BIN_DIR := $(GOPATH)/bin

lint:
	test -z $$(gofmt -s -l .)
	go vet ./...

test:
	go test -v ./...

# Compilation
LDFLAGS := '-s -w -extldflags "-static"'
static:
	CGO_ENABLED=0 go build -ldflags=${LDFLAGS} ./cmd/cuetsy

install:
	CGO_ENABLED=0 go install -ldflags=${LDFLAGS} ./cmd/cuetsy

uninstall:
	go clean -i ./cmd/cuetsy

# CI
drone:
	cue export ./.drone/drone.cue > .drone/drone.yml
	drone fmt --save .drone/drone.yml
