# Changelog

All notable changes to this project will be documented here.
Versioning follows [Semantic Versioning](https://semver.org/).

## [0.0.15] - 2026-09-28

Test-only release: no change to the binary's behavior.

### Changed
- E2E scenarios assert the exact output, exit code and error message instead of only success or failure, through a shared `scripts/test-e2e/lib.sh`. A failing scenario prints its diff and output, and CI now runs the scenarios.
- `scripts/test.sh` and `scripts/test-extensive.sh` check every parsed value instead of printing it; each `test-extensive.sh` run starts from no `SHOPTS_` variables.
- `scripts/test-contract.sh` checks stdout byte for byte, including the final newline.
- Every bash suite and the Go tests clear `GO_SHOPTS_*` settings from the environment first, so exported settings can't change the results.
- `scripts/TEST.md` rewritten to describe the suites as they are.

### Fixed
- Two checks in `scripts/test-extensive.sh` (invalid enum, missing required option) printed an error on failure but still passed; they now fail unless shopts exits 3.

---

## [0.0.14] - 2026-09-28

Implements the shopts contract spec. Several changes are breaking.

### Added
- Positional arguments: `positional=1`, `2`, … binds bare arguments; `positional=rest` collects the remainder into a list. Everything after `--` is bare.
- `min`/`max` inclusive bounds for `int` and `float`.
- `define` entries declare named patterns usable as `{{ Name }}`.
- `GO_SHOPTS_NAME` sets the program name shown in help and errors.
- Named delimiters for `GO_SHOPTS_OUT_DELIM` and `GO_SHOPTS_LIST_DELIM`, written like validators: `{{ tab }}` and `{{ null }}` (a NUL byte, which no argument can contain, so every value reads back exactly).
- `GO_SHOPTS_DASH` (default `1`): long names are typed with dashes, so `long=dry_run` is `--dry-run`. Set to `0` to type names as written. Output names are unchanged, and a mistyped spelling gets a "did you mean" hint.
- Each built-in validator has its own failure message. The README validator table is generated from the registry (`make readme`).
- `scripts/test-contract.sh`: acceptance suite with checks for each item of the spec.

### Changed
- `-V`/`--version` ignores the schema and settings, so it works even when they are broken. `-H`/`--help` needs a valid schema but ignores settings it doesn't show. An internal error (panic) exits 1 instead of Go's default 2, which would look like a schema error.
- **Breaking:** `-H`/`--help` and `-V`/`--version` exit 7 instead of 0, so callers can tell help apart from a successful parse. `--version` now prints to stderr and works from any position.
- **Breaking:** Help describes the caller's script only: the `Usage: shopts SCHEMA` header, the `GO_SHOPTS_` variable list and the type notes are gone; positionals are listed.
- **Breaking:** Inline regexes must match the whole value (`pattern=[0-9]+` rejects `abc1`).
- **Breaking:** Quoted schema values use a new rule: only `\"` and `\\` are escapes, so `pattern="^\d+$"` works. Go escapes such as `\n` are no longer interpreted.
- **Breaking:** Long names containing `_` are typed with dashes by default (`--dry-run`, not `--dry_run`). Set `GO_SHOPTS_DASH=0` for the old spelling.
- **Breaking:** A flag is `true` when given and `false` when not: it cannot have a `default` or be `required=true`. For on-by-default behavior, name the flag for turning it off (`long=no_cache`, typed `--no-cache`). No `--no-` form is implied; to accept both spellings, declare both flags.
- **Breaking:** `required=` accepts only `true` or `false`. A field given twice in one entry is an error. An unquoted value can no longer continue onto the next line.
- **Breaking:** Repeating a flag is an error, like any other non-list option.
- **Breaking:** `bool` values are emitted as `true`/`false` and `int` values as plain decimal (`007` → `7`).
- **Breaking:** `GO_SHOPTS_UPCASE` and `GO_SHOPTS_DASH` must be booleans, and invalid settings exit 1.
- Values may now contain the output delimiter: the first delimiter on a line always ends the key, since keys never contain it. Only newlines are rejected. `GO_SHOPTS_OUT_DELIM` and `GO_SHOPTS_LIST_DELIM` may be any string without a newline.
- Schema errors report line and column (`schema line 4, col 23: unknown field "patern"`), with a suggestion for misspelled fields.
- Argument errors are printed one per line, followed by the usage line, instead of the full help.
- `minLength`/`maxLength` count characters (Unicode code points), not bytes.
- The schema is parsed by a single-pass lexer built on `text/scanner`; the package is split into one file per concern.
- README corrected where it disagreed with the code (exit codes, `long` characters, `bool` values, "exported", "no subshells", POSIX `-u=value`, `pattern` on lists, `bin/`), and notes that the `wait $!` idiom needs bash 4.4 or later.

### Fixed
- `RelativePath` accepts `config/file.yaml`; `PortNumber` rejects `+80` and leading zeros; `GitRef` follows git's ref naming rules; `IPv6Address` accepts IPv4-mapped addresses.

### Removed
- `docs/review.md` (stale).

---

## [0.0.13] - 2026-04-16

### Fixed
- Long option names must now start with a letter; leading digits or underscores are rejected at schema parse time.
- `maxLength=0` is now rejected with a clear error (`maxLength must be >= 1`).
- `list` type now rejects `minLength`/`maxLength` with a helpful error pointing to `minItems`/`maxItems`.
- `splitEnum` rewritten as a single-pass parser, fixing edge cases where escape sequences inside quoted enum items could be mishandled.

### Changed
- Documented that option values resembling flags (e.g. `--pass`) are consumed as values, not parsed as options; use `=` syntax (`-u=--pass`) to avoid ambiguity.

---

## [0.0.12] - 2026-04-04

### Fixed
- Empty inline values (e.g., `--name=`) now correctly use empty string instead of consuming the next argument
- Flags with inline values (e.g., `--flag=`) are now properly rejected with "does not take a value" error
- Enum type now rejects incompatible minLength, maxLength, and pattern fields at schema parse time
- `required=` field now validates boolean values and rejects non-boolean inputs like "maybe"
- Float validation now rejects NaN and Inf values (only finite numbers accepted)
- Double dash (`--`) with multiple positional arguments now reports all of them, not just the first
- Duplicate enum values are now detected and rejected at schema parse time
- Repeating non-list options now properly reports "already specified" error

---

## [0.0.11] - 2026-04-04

### Changed
- Release process documentation clarified: all development work must be committed before release, lint and test must pass before writing changelog, `make test-all` now used for complete test coverage.
- `bin/release.sh`: removed internal `make lint test` calls; lint and test are now explicit pre-steps in the manual release workflow.

---

## [0.0.10] - 2026-04-04

### Added
- End-to-end test suite: 22 deterministic test cases in `scripts/test-e2e/` covering valid and invalid CLI argument scenarios.
- `scripts/run-e2e-tests.sh`: parallel E2E test runner.
- `scripts/TEST.md`: documentation for the E2E test suite.
- `make test-e2e` target: runs the E2E test suite against the built binary.
- `make test-all` target: runs `test-go` and `test-bash` in parallel, then `test-e2e` sequentially.
- `make lint-go` target: runs `golangci-lint` across all Go packages.
- `make lint-bash` target: runs `shellcheck` across all bash scripts.
- `make lint-all` target: runs `lint-bash` and `lint-go` in parallel.

### Changed
- `make lint` now runs `lint-bash` and `lint-go` as explicit dependencies.
- Release process updated: explicit `make lint-all` and `make test` steps added before `make release`.

---

## [0.0.9] - 2026-04-04

### Breaking Changes
- Help flag changed from `-h` to `-H`. Scripts calling `-h` for help must be updated to `-H`.

### Added
- Reserved built-in flags: `short=H`, `short=V`, `long=help`, and `long=version` are now rejected at schema parse time (exit 2). This prevents silent shadowing of the built-in `-H`/`--help` and `-V`/`--version` flags.
- `-h` is now available for use in schemas (e.g. `short=h, long=host`).

---

## [0.0.8] - 2026-04-04

### Changed
- Long option names now restricted to `[A-Za-z0-9_]` — hyphens are no longer allowed.
- Enum items now support per-item quoting (`"a,b",c`) for values containing commas; the previous `\,` escape is removed.

### Fixed
- Missing trailing `;` in a schema now reports a clear `missing terminating ';'` error instead of the misleading "schema must contain at least one option".
- Empty enum items (e.g. `a,,b`) are now rejected at schema parse time.

---

## [0.0.7] - 2026-04-04

### Changed
- Release scripts extracted from Makefile into `bin/release.sh`, `bin/tag.sh`, and `bin/clean-tag.sh` to reduce Makefile complexity.
- `make release` now validates version format, changelog entry, and remote state before running lint and tests, failing fast on quick checks.
- `make tag` now checks for a passing CI run on the tagged commit before pushing, matching the requirement enforced by the release workflow.

### Added
- `make clean-tag` — removes a tag from both local and remote, useful for re-triggering a failed release workflow.
- `RELEASE.md` — documents the step-by-step release process.

---

## [0.0.6] - 2026-04-04

### Breaking Changes
- Output field delimiter changed from NUL (`\0`) to tab (`\t`). The consumer pattern in Bash scripts must change from `while IFS= read -r -d $'\0' key && IFS= read -r val` to `while IFS=$'\t' read -r key val`.
- Values containing a tab character are now rejected as invalid input (same treatment as newline). Use `GO_SHOPTS_OUT_DELIM` to select a different delimiter if tab appears in your values.

### Added
- Built-in pattern validators: use `pattern={{ Name }}` in a schema field instead of a raw regex. 18 validators available (`EmailAddress`, `URL`, `IPv4Address`, `SemVer`, `PortNumber`, `GitSHA`, etc.). Unknown names are schema errors (exit 2). See README for the full table.
- `GO_SHOPTS_OUT_DELIM` environment variable to override the output field delimiter (default: tab). Values containing the configured delimiter are rejected at parse time.

### Fixed
- `make benchmark` schema string used `;` as field separators instead of `,`, causing every benchmark run to fail with a schema error.

---

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
- Shell-safe `KEY\tVALUE\n` output format for safe `IFS=$'\t' read -r` consumption
- Environment controls: `GO_SHOPTS_PREFIX`, `GO_SHOPTS_UPCASE`, `GO_SHOPTS_LIST_DELIM`, `GO_SHOPTS_OUT_DELIM`
- `-h`/`--help` for schema-derived usage text
- `-V`/`--version` to print the build version
- `--` to terminate option parsing
- GitHub Actions CI: Go tests (with race detector), ShellCheck, golangci-lint, bash integration tests
- Tag-driven release workflow building a Linux amd64 binary with SHA256 checksum
- `Makefile` with `test`, `build`, `clean`, `benchmark`, `compare`, and `tag` targets
- Bash reference parser and benchmark scripts in `bench/`
