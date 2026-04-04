VERSION := $(shell cat VERSION)
BINARY  := bin/shopts
N       := 100

.PHONY: all clean test test-go test-bash benchmark compare lint tag clean-tag release

all: test

$(BINARY): $(shell find ./cmd ./pkg -name '*.go') go.mod go.sum
	go build -ldflags "-X main.version=v$(VERSION) -s -w" -trimpath -o $(BINARY) ./cmd/shopts

build: $(BINARY)

clean:
	rm -f $(BINARY)

test: $(BINARY) test-go test-bash

test-go:
	go test -race ./...

test-bash:
	./scripts/test.sh
	./scripts/test-negative.sh
	./scripts/test-extensive.sh

lint:
	golangci-lint run ./...

benchmark: $(BINARY)
	./bench/benchmark.sh $(N) \
	  "long=user, short=u, required=true, type=string, minLength=3, help=Username;" \
	  -u alice

compare: $(BINARY)
	./bench/compare.sh $(N) -u alice -p s3cr3tpass

tag: TAG_VERSION ?= $(VERSION)
tag:
	@bin/tag.sh $(TAG_VERSION)

clean-tag: TAG_VERSION ?= $(VERSION)
clean-tag:
	@bin/clean-tag.sh $(TAG_VERSION)

release:
	@bin/release.sh $(VERSION)
