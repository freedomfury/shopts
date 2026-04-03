# shopts

[![CI](https://github.com/freedomfury/shopts/actions/workflows/ci.yml/badge.svg)](https://github.com/freedomfury/shopts/actions/workflows/ci.yml)

`shopts` is a schema-driven Go CLI parser that reads an inline schema definition,
validates and pairs provided arguments, and emits shell-safe output as `KEY\0VALUE\n` records.

This project also includes Bash scripts for test automation and benchmarking support, plus a Bash reference parser for comparison.

See [CHANGELOG.md](CHANGELOG.md) for release history.

## Features

- Schema-driven CLI argument parsing from a text-based schema string.
- Built-in validation for required args, types, enum, min/max length, patterns,
  and list item counts.
- `flag` type support (boolean switch that is true when present).
- `list` type support (repeatable option values joined with `GO_SHOPTS_LIST_DELIM`).
- Environment-controlled output naming: `GO_SHOPTS_PREFIX`, `GO_SHOPTS_UPCASE`.
- `-h`/`--help` for schema-derived usage text.
- No shell eval; output is intended for safe `read -d $'\0'` consumer patterns.

## Usage

```
shopts "$SCHEMA" [OPTIONS...]
```

- `args[0]`: binary path
- `args[1]`: schema string (required)
- `args[2:]`: command arguments to parse

### Schema format

The schema is a text block containing one or more non-empty lines. Each line is
semicolon-separated `key=value` pairs terminated by `;`.

- each line defines one option
- fields are `short`, `long`, `required`, `type`, `help`, etc.
- optional quoting with Go-style string literals (for semicolons, commas, etc.)


Example:

```bash
SCHEMA='
short=u;long=username;required=true;type=string;help=Username;minLength=3;
short=p;long=password;required=true;type=string;help=Password;minLength=6;
short=v;long=verbose;type=flag;help=Verbose mode;
short=t;long=tags;type=list;minItems=1;maxItems=5;
'

./shopts "$SCHEMA" -u alice -p s3cret -v -t a -t b
```

### Notes

- `flag` options do not accept a value (`-v` sets true, absence is false in output).
- `--` terminates options and disallows trailing positional args.
- `-abc` bundles are not supported; only single-letter short options.
- Unknown options and invalid schemas produce stderr errors and exit code 1.

## Project Structure

- `cmd/shopts/` — Go CLI entrypoint.
- `pkg/shopts/` — Go parser implementation and logic.
- `scripts/` — Bash test scripts used to exercise the CLI end to end.
- `bench/` — Bash reference parser plus benchmark scripts for comparing Go and Bash behavior.
- `bin/` — Built Go binaries, kept out of version control.

There is no separate `benchmark/` folder; the benchmark helpers live in `bench/`.

## Schema fields

- `long` (required): long option name, allowed `[A-Za-z0-9_-]+`
- `short` (optional): single alphanumeric short option
- `required`: `true|false`
- `type`: `string`, `int`, `float`, `bool`, `enum`, `list`, `flag`
- `help`, `description`, `default`, `pattern`, `failure`
- `enum`: comma-separated enum values (required for `enum`, forbidden for others)
- `minLength`, `maxLength` (string types)
- `minItems`, `maxItems` (`list` type only)

## Validation rules

- `required` and `default` are mutually exclusive.
- `enum` requires `type=enum`.
- `flag` rejects `minLength/maxLength/pattern/enum`.
- `int`, `float`, `bool` reject `minLength/maxLength/pattern`.
- `minLength <= maxLength`, `minItems <= maxItems`.
- Validate defaults at schema parse time.

## Output behavior

- Successful parse prints `KEY\0VALUE\n` for each emitted option.
- `KEY` is generated as `GO_SHOPTS_<sanitized-long>` by default.
- `GO_SHOPTS_UPCASE=1` uppercases the key name.
- `GO_SHOPTS_PREFIX` overrides the prefix (must be a valid shell identifier prefix, or empty).
- `list` values are joined with `GO_SHOPTS_LIST_DELIM` (`,`, default).

## Help

`-h` or `--help` prints schema-derived usage and exits 0.

`-V` or `--version` prints the version and exits 0.

## Quick test snippet

```bash
export GO_SHOPTS_UPCASE=1
SCHEMA='\
short=u;long=user;required=true;type=string;help=User;\
short=v;long=verbose;type=flag;help=Verbose;\
'

while IFS= read -r -d $'\0' k && IFS= read -r v; do
  printf -v "$k" '%s' "$v"
  declare -xr "${k#GO_SHOPTS_}"="$v"
done < <(./shopts "$SCHEMA" -u alice -v)

printf 'USER=%s\n' "$USER"
printf 'VERBOSE=%s\n' "$VERBOSE"
```

## Testing

### Make Targets

All testing is automated via the `Makefile`. Common targets:

| Target | Description |
|---|---|
| `make` / `make all` | Run full test suite (tests + bash integration). Only rebuilds binary if source changed. |
| `make build` | Build the binary to `bin/shopts`. Skip if binary is up-to-date. |
| `make clean` | Remove the binary (`bin/shopts`). |
| `make test` | Run all tests (Go + bash). |
| `make test-go` | Run Go unit tests only (`go test -race ./...`). |
| `make test-bash` | Run bash integration tests only. |
| `make benchmark` | Run Go parser benchmark (default 100 iterations). |
| `make benchmark N=1000` | Run benchmark with custom iteration count. |
| `make compare` | Compare Go parser vs Bash reference parser (default 100 iterations). |
| `make compare N=1000` | Run comparison with custom iteration count. |
| `make tag` | Create and push release tag (`v{VERSION}`). Validates: VERSION is semver, on main branch, tag doesn't exist remotely. Triggers release workflow. |

### GitHub Actions CI

- Runs on: `pull_request` and `push` to `main`
- Jobs: `test` (Go tests, ShellCheck, bash scripts), `lint` (golangci-lint), `build` (builds binary on main only)

### Bash Test Scripts

Run these from the project root.

- `./scripts/test.sh` builds `bin/shopts` if needed and checks a basic successful parse path, including exported environment variables.
- `./scripts/test-negative.sh` verifies help output and a representative validation failure path.
- `./scripts/test-extensive.sh` exercises the wider type matrix, defaults, repeated list values, flags, and delimiter handling.

All test scripts will build the Go binary if missing. Ensure `bin/shopts` is up to date.

## Releases

Releases are tag-driven through GitHub Actions using a build-once, promote pattern.

### Release Workflow

1. **Merge PR to main** -- CI runs tests, builds binary, uploads as artifact (retained 90 days)
2. **Verify main is green** -- check GitHub Actions CI passed
3. **Update `VERSION`** and commit with message: `Release v{VERSION}` (must be valid semver: `major.minor.patch`)
4. **Run `make tag`** -- validates VERSION format and branch, then creates and pushes `v{VERSION}` tag. Fails if: not on main branch, VERSION is not semver, or tag already exists on remote
5. **Release workflow** -- validates VERSION, checks main is stable, downloads artifact, publishes GitHub release

### Artifact Details

- Built on: Linux amd64, `CGO_ENABLED=0`, trimmed paths
- Versioned via: `-X main.version=v{VERSION}` at build time
- Published to: GitHub Releases with SHA256 checksum

See `spec-releaseworkflow.md` for detailed architecture.

## Bash Reference Parser & Benchmarks

- `bench/bash-parser.sh` is a Bash implementation of the parser behavior used as a local reference when comparing performance or behavior.
- `bench/benchmark.sh` runs the Go parser repeatedly for a single schema and argument set, then prints total and per-call timing.
- `bench/compare.sh` runs the Go parser and the Bash reference parser with the same arguments, then prints a side-by-side timing comparison.

These benchmark scripts are intended for local measurement. They expect the parser binaries they reference to be built and available in the repository root.

Example:

```bash
./bench/benchmark.sh 1000 "$SCHEMA" -u alice -p s3cret
./bench/compare.sh 1000 -u alice -p s3cret
```

## Code Quality

- All Bash scripts are checked with `shellcheck` (run: `shellcheck scripts/*.sh bench/*.sh`)
- Go code is linted with `golangci-lint` (run: `golangci-lint run ./...`)

## Versioning

- Follows [Semantic Versioning](https://semver.org/). Currently pre-1.0 (`0.x.x`) — breaking changes may occur in minor versions.
- `VERSION` at the repository root is the source of truth for release numbers (no `v` prefix).
- Tags use the `v` prefix (e.g. `v0.0.1`). The release workflow validates the tag matches `VERSION`.
- To cut a release: update `VERSION`, commit, then run `make tag`.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE).

## Go Module

- Module: `github.com/freedomfury/shopts`
- Go version: 1.24.4
