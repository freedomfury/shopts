VERSION := $(shell cat VERSION)
BINARY  := bin/shopts
N       := 100

.PHONY: all clean test test-go test-bash benchmark compare lint tag release

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
	  "long=user;short=u;required=true;type=string;minLength=3;help=Username;" \
	  -u alice

compare: $(BINARY)
	./bench/compare.sh $(N) -u alice -p s3cr3tpass

tag:
	@echo "Validating version: v$(VERSION)"
	@if ! echo "$(VERSION)" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		echo "Error: VERSION is not in semver format (major.minor.patch)"; \
		exit 1; \
	fi
	@BRANCH=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$BRANCH" != "main" ]; then \
		echo "Error: Not on main branch (current: $$BRANCH)"; \
		exit 1; \
	fi
	@git fetch origin --prune-tags --prune
	@if git ls-remote origin refs/tags/v$(VERSION) | grep -q .; then \
		echo "Error: Tag v$(VERSION) already exists on remote"; \
		exit 1; \
	fi
	@echo "Creating and pushing tag v$(VERSION)"
	git tag v$(VERSION)
	git push origin v$(VERSION)

release: lint
	@echo "Validating version: v$(VERSION)"
	@if ! echo "$(VERSION)" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		echo "Error: VERSION is not in semver format (major.minor.patch)"; \
		exit 1; \
	fi
	@if ! grep -q "## \[$(VERSION)\]" CHANGELOG.md; then \
		echo "Error: No entry for [$(VERSION)] found in CHANGELOG.md"; \
		exit 1; \
	fi
	@git fetch origin --prune-tags --prune
	@if git ls-remote origin refs/tags/v$(VERSION) | grep -q .; then \
		echo "Error: Tag v$(VERSION) already exists on remote"; \
		exit 1; \
	fi
	@echo "Staging all changes"
	git add -A
	@echo "Committing release $(VERSION)"
	git commit -m "Release: $(VERSION)"
	@echo "Pushing"
	git push
