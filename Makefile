all: build
.PHONY: all

VERSION := $(shell git describe --tags --dirty --match 'v*' 2> /dev/null || git describe --always --dirty)
IMAGE :=quay.io/sallyom/scribecli:latest

scribe: build
.PHONY: scribe

build:
	go build ./cmd/scribe
.PHONY: build

# Build the image
image:
	podman build --build-arg "VERSION=$(VERSION)" . -t ${IMAGE}
.PHONY: image

# Push the image
.PHONY: podman-push
podman-push:
	podman push ${IMAGE}

# TODO: gofmt, Dockerfile
