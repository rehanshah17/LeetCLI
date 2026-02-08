GOCACHE ?= $(CURDIR)/.cache/go-build
GOMODCACHE ?= $(CURDIR)/.cache/gomod
GO ?= go

.PHONY: build test run tidy

build:
	@mkdir -p bin .cache/go-build .cache/gomod
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) build -o bin/leet .

test:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) test ./...

run:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) run .

tidy:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) mod tidy
