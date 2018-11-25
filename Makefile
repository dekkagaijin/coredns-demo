GOPATH?=$(shell go env GOPATH)
export PATH := $(GOPATH)/bin:$(PATH)

NOW := $(shell date -u +%Y-%m-%dT%H:%MZ)
GITCOMMIT?=$(shell git describe --always)
VERSION?=$(NOW)-$(GITCOMMIT)-dev

PKG_LIST = $(shell go list ./... | grep -v /vendor/)

all: bin

.PHONY: deps
deps:
	go get -u -t ./...

.PHONY: bin
bin: deps
	go build lookup.go

.PHONY: vet
vet:
	go vet $(PKG_LIST)

.PHONY: lint
lint: ensure-golint
	golint -set_exit_status -min_confidence=0.4 $(PKG_LIST)

.PHONY: ensure-golint
ensure-golint:
	go get -u github.com/golang/lint/golint
