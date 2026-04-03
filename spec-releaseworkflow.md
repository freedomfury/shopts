# Release Workflow Spec

## Problem

The current CI/CD pipeline rebuilds the binary at every stage -- on PR, on merge to main, and again on tag. This means:

- The artifact shipped in a release is not the same binary that was tested on main
- Build time is wasted on redundant rebuilds
- For slow builds (e.g. OS images) this compounds into a serious bottleneck

## Desired Behavior

**Build once, promote the artifact.**

- CI runs once on PR (tests + lint)
- Main builds the binary, tests it, and uploads it as an artifact
- A tag downloads that artifact, verifies main was stable, and publishes the release
- The tag pipeline should be fast -- no rebuilding

---

## Trigger Changes

### Current
```yaml
on:
  push:
    branches:
      - main
  pull_request:
```

### Proposed
- **PR** (`pull_request`) -- run tests and lint only, no build artifact
- **Main** (`push: main`) -- run tests, build binary, upload artifact
- **Tag** (`push: tags: v*`) -- download artifact from latest main, publish release

Remove the redundant CI re-run on main that currently mirrors the PR run. Main now has a distinct purpose: produce the artifact.

---

## Main Workflow (`ci.yml`)

### Jobs

**test**
- Runs on: `pull_request` and `push: main`
- Steps: checkout, setup Go, ShellCheck, Go tests (`-race`), bash integration tests

**build** (runs on `push: main` only)
- Depends on: `test`
- Steps:
  - Build binary: `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-X main.version=v$(VERSION)" -o dist/shopts-linux-amd64 ./cmd/shopts`
  - Generate SHA256 checksum: `sha256sum dist/shopts-linux-amd64 > dist/SHA256SUMS`
  - Upload artifact: `actions/upload-artifact` with name `shopts-linux-amd64`, include both files, retain for 90 days

**lint** (runs on `pull_request` and `push: main`)
- Steps: checkout, setup Go, golangci-lint

---

## Release Workflow (`release.yml`)

Triggered by: `push: tags: v*`

### Steps

1. **Validate VERSION matches tag**
   - Read `VERSION` file
   - Compare to `GITHUB_REF_NAME` (strip `v` prefix)
   - Fail if mismatch

2. **Check main is stable**
   - Find the latest completed workflow run on `main` branch for `ci.yml`
   - Check its conclusion is `success`
   - Fail with clear message if not: `"Latest main build is not stable -- cannot release"`

3. **Download artifact**
   - Get the run ID from step 2
   - Download `shopts-linux-amd64` artifact from that run

4. **Publish GitHub release**
   - `gh release create` with the tag
   - Attach binary and SHA256SUMS
   - Auto-generate release notes

---

## Edge Cases

| Scenario | Behavior |
|---|---|
| Tag pushed before main run finishes | Step 2 finds no successful run, fails with clear message |
| Tag pushed when main is red | Step 2 detects failure, blocks release |
| Two PRs merge close together | Main build captures the combined state, release always ships latest green main |
| Artifact expired (>90 days) | Download step fails, re-merge or re-push to main to produce a fresh artifact |

---

## Release Process (Operator Workflow)

1. Merge PR to main -- CI runs tests, builds artifact, uploads it
2. Verify main is green
3. Update `VERSION` file and commit
4. Run `make tag` -- pushes `v{VERSION}`, triggers release workflow
5. Release workflow validates, downloads artifact, publishes

---

## Files to Modify

- `.github/workflows/ci.yml` -- add `build` job gated to `push: main`, split test/build concerns
- `.github/workflows/release.yml` -- replace build steps with artifact download + main stability check
