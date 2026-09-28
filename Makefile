VERSION := $(shell cat VERSION)
BINARY  := bin/shopts
N       := 100

.PHONY: all clean test test-go test-bash test-e2e test-all benchmark compare lint lint-go lint-bash lint-all readme tag clean-tag release

all: test

$(BINARY): $(shell find ./cmd ./pkg -name '*.go') go.mod go.sum
	go build -ldflags "-X main.version=v$(VERSION) -s -w" -trimpath -o $(BINARY) ./cmd/shopts

build: $(BINARY)

clean:
	rm -f $(BINARY)

test: $(BINARY) test-go test-bash

test-go:
	go test -race ./...

test-bash: $(BINARY)
	./scripts/test.sh
	./scripts/test-negative.sh
	./scripts/test-extensive.sh
	./scripts/test-contract.sh

readme:
	go test ./pkg/shopts -run TestREADMETables -update

test-e2e: $(BINARY)
	@scripts/run-e2e-tests.sh bin/shopts

test-all: $(BINARY)
	$(MAKE) -j2 test-go test-bash
	$(MAKE) test-e2e

benchmark: $(BINARY)
	./bench/benchmark.sh $(N) \
	  "long=user, short=u, required=true, type=string, minLength=3, help=Username;" \
	  -u alice

lint-go:
	golangci-lint run ./...

lint: lint-bash lint-go

lint-all:
	$(MAKE) -j2 lint-bash lint-go

# Every shell script in the repo, committed or not (ignored files excluded).
lint-bash:
	@echo "Linting bash scripts..."
	git ls-files -z --cached --others --exclude-standard -- '*.sh' | xargs -0 shellcheck -x

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
