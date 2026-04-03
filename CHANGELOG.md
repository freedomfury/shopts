# Changelog

All notable changes to this project will be documented here.
Versioning follows [Semantic Versioning](https://semver.org/).

---

## [0.0.2] - 2026-04-03

### Changed
- Optimized binary size: stripped debug symbols and build paths (`-s -w -trimpath`), reducing binary from ~3 MB to ~1.8 MB
- Enhanced `make tag` with validation: verifies branch is main, VERSION is valid semver, and tag doesn't already exist remotely

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
