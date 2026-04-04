# Changelog

All notable changes to this project will be documented here.
Versioning follows [Semantic Versioning](https://semver.org/).

## [0.0.5] - 2026-04-04

### Changed
- Makefile `release` and `tag` targets are now idempotent. Running them multiple times with the same version will detect previous operations and warn instead of failing or creating duplicates.
- `release` target now checks git log to detect if a release commit already exists, preventing duplicate release commits.
- `tag` target now checks both local and remote tags before creating a new tag, preventing duplicate tags.
---

## [0.0.4] - 2026-04-03

### Breaking Changes
- Default emitted prefix changed from `GO_SHOPTS_` to `SHOPTS_`. Scripts using the default prefix must update variable references (e.g. `$GO_SHOPTS_USER` → `$SHOPTS_USER`).

### Added
- Reserved namespace guard: `GO_SHOPTS_PREFIX` must not start with `GO_SHOPTS_` (reserved for internal controls). Attempting to do so produces an immediate error (exit 1).
- Distinct exit codes: `1` general failure, `2` schema error (invalid schema), `3` parse/validation error (bad arguments). Previously all failures exited with code `1`.

### Changed
- `GO_SHOPTS_UPCASE` now defaults to on (`true`), so emitted variable names are uppercase by default. Setting `GO_SHOPTS_UPCASE=0` retains lowercase output.

### Fixed
- CI release build now includes `-s -w` ldflags to strip symbols and debug info, keeping release binaries at ~1.8 MB instead of ~3 MB.
- Release workflow now verifies CI has passed for the specific tagged commit, not just any successful main run.

---

## [0.0.3] - 2026-04-03

### Changed
- Argument parsing errors (unknown options, missing values, parse-time type errors) are now all collected and reported together in a single error message instead of failing on the first error encountered. This gives users a complete picture of all problems in one run.
- Type error messages no longer expose Go stdlib internals. For example:
  - `int` errors now say `must be a valid integer` (was `int value required: strconv.Atoi: parsing "abc": invalid syntax`)
  - `float` errors now say `must be a valid number` (was `float value required: strconv.ParseFloat: ...`)
  - `bool` errors now say `must be a valid boolean`
- Parse errors and validation errors are merged and reported together at end of a run.
- `dedent` now strips both leading spaces and tabs, so tab-indented heredocs work correctly.

---

## [0.0.2] - 2026-04-03

### Changed
- Optimized binary size: stripped debug symbols and build paths (`-s -w -trimpath`), reducing binary from ~3 MB to ~1.8 MB
- Enhanced `make tag` with validation: verifies branch is main, VERSION is valid semver, and tag doesn't already exist remotely
- `list` type now enforces an implicit `maxItems=100` when no explicit `maxItems` is set, to prevent unbounded input
- `list` type now enforces an implicit `minItems=1` when `required=true` and no explicit `minItems` is set
- Rewrote README schema documentation with a type reference table and field reference table, clarifying `flag` vs `bool`, list item string behavior, and implicit list constraints
- Added "Why?" section to README explaining the problem shopts solves, why existing Bash argument parsing tools fall short, and practical benefits for script authors

---

## [0.0.1] - 2026-04-03

Initial release.

### Added
- Schema-driven CLI argument parsing from an inline text schema
- Supported types: `string`, `int`, `float`, `bool`, `enum`, `list`, `flag`
- Validation: `required`, `default`, `minLength`/`maxLength`, `pattern`/`failure`, `enum`, `minItems`/`maxItems`
- Shell-safe `KEY\0VALUE\n` output format for safe `read -d $'\0'` consumption
- Environment controls: `GO_SHOPTS_PREFIX`, `GO_SHOPTS_UPCASE`, `GO_SHOPTS_LIST_DELIM`
- `-h`/`--help` for schema-derived usage text
- `-V`/`--version` to print the build version
- `--` to terminate option parsing
- GitHub Actions CI: Go tests (with race detector), ShellCheck, golangci-lint, bash integration tests
- Tag-driven release workflow building a Linux amd64 binary with SHA256 checksum
- `Makefile` with `test`, `build`, `clean`, `benchmark`, `compare`, and `tag` targets
- Bash reference parser and benchmark scripts in `bench/`
