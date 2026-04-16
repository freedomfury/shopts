# shopts

[![CI](https://github.com/freedomfury/shopts/actions/workflows/ci.yml/badge.svg)](https://github.com/freedomfury/shopts/actions/workflows/ci.yml)

`shopts` is a schema-driven CLI argument parser for Bash scripts. Define your options once in a simple text schema, and `shopts` handles parsing, validation, help text, and shell-safe output — so you don't have to.

See [CHANGELOG.md](CHANGELOG.md) for release history.

## Why?

Parsing command-line arguments in Bash is tedious and error-prone. The typical approach is a hand-rolled `while/case/shift` loop that grows with every new option, has no validation, and must be rewritten for every script. `getopts` helps with simple flags, but falls short the moment you need required options, type checking, enums, pattern validation, or repeatable arguments.

`shopts` replaces all of that with a single binary and a schema string:

```bash
# Instead of 40+ lines of case/shift boilerplate:
SCHEMA='
short=u, long=user, required=true, type=string, minLength=3, help=Username;
short=p, long=port, type=int, default=8080, help=Port number;
short=v, long=verbose, type=flag, help=Enable verbose output;
'
# Emits SHOPTS_<LONG> for each option (uppercase by default).
# GO_SHOPTS_ is reserved for internal controls — your variables land in SHOPTS_.

while IFS=$'\t' read -r key val; do
  # Variables are assigned here as shopts streams them out.
  printf -v "$key" '%s' "$val"
done < <(shopts "$SCHEMA" "$@")
wait $! || exit $?  # propagate exit code: 2=schema error, 3=bad args

# Variables are now set and ready to use.
printf "User: %s\n" "$SHOPTS_USER"
printf "Port: %d\n" "$SHOPTS_PORT"
```

That's it. Arguments are parsed, validated, type-checked, and exported as shell variables. If anything is wrong — missing required option, bad type, value too short — `shopts` prints a clear error and exits non-zero. Pass `-H` and your users get auto-generated help text derived from the schema.

**No `eval`. No subshells. No dependencies.** Output uses tab-delimited records (`KEY\tVALUE\n`), which are safe to consume in Bash without worrying about spaces, quotes, or injection.

For comparison, see [bench/bash-parser.sh](bench/bash-parser.sh) — a hand-written Bash implementation of the same logic. Most of that file is the boilerplate argument-parsing code that `shopts` eliminates.

## Features

- Schema-driven CLI argument parsing from a text-based schema string.
- Built-in validation for required args, types, enum, min/max length, patterns,
  and list item counts.
- `flag` type support (boolean switch that is true when present).
- `list` type support (repeatable option values joined with `GO_SHOPTS_LIST_DELIM`).
- Environment-controlled output naming: `GO_SHOPTS_PREFIX`, `GO_SHOPTS_UPCASE`.
- Reserved namespace: `GO_SHOPTS_` prefix is owned by the binary; `GO_SHOPTS_PREFIX` must not start with `GO_SHOPTS_`.
- `-H`/`--help` for schema-derived usage text.
- Reserved built-ins: `-H`/`--help` and `-V`/`--version` are always available; schemas declaring `short=H`, `short=V`, `long=help`, or `long=version` are rejected at parse time (exit 2).
- No shell eval; output is intended for safe `IFS=$'\t' read -r` consumer patterns.

## Usage

```
shopts "$SCHEMA" [OPTIONS...]
```

- `args[0]`: binary path
- `args[1]`: schema string (required)
- `args[2:]`: command arguments to parse

### Schema format

The schema is a text block containing one or more entries. Each entry is
terminated by `;` and its fields are separated by `, ` (comma-space).

- each entry defines one option
- `long` is **required**; `short` is optional (primarily a convenience alias)
- fields are `short`, `long`, `required`, `type`, `help`, etc.
- optional quoting with Go-style string literals (for commas or semicolons inside values)


Example:

```bash
SCHEMA='
short=u, long=username, required=true, type=string, help=Username, minLength=3;
short=p, long=password, required=true, type=string, help=Password, minLength=6;
short=v, long=verbose, type=flag, help=Verbose mode;
short=t, long=tags, type=list, minItems=1, maxItems=5;
'

./shopts "$SCHEMA" -u alice -p s3cret -v -t a -t b
```

### Notes

- `flag` options do not accept a value (`-v` sets true, absence is false in output).
- `--` terminates options and disallows trailing positional args.
- `-abc` bundles are not supported; only single-letter short options.
- Option values that look like flags are consumed as values, not parsed as options. For example, `-u --pass` treats `--pass` as the value of `-u`. Use `=` syntax to avoid ambiguity: `-u=--pass`. This follows standard POSIX CLI conventions.
- Unknown options and invalid schemas produce stderr errors and exit code 1.
- `GO_SHOPTS_UPCASE` defaults to `1` (default behavior is uppercase output variable names; set `GO_SHOPTS_UPCASE=0` to preserve legacy lowercase names).
- `GO_SHOPTS_PREFIX` must not start with `GO_SHOPTS_` — that prefix is reserved for internal controls. Attempting to set it will produce an error (exit 1).
- Exit codes: `1` general failure, `2` schema error (invalid schema), `3` parse/validation error (bad arguments).
- Multiple unknown options and validation errors are all reported together in a single error message rather than failing on the first one.
- Type error messages are human-readable (e.g. `must be a valid integer`) with no Go stdlib internals exposed.

## Project Structure

- `cmd/shopts/` — Go CLI entrypoint.
- `pkg/shopts/` — Go parser implementation and logic.
- `scripts/` — Bash test scripts used to exercise the CLI end to end.
- `bench/` — Bash reference parser plus benchmark scripts for comparing Go and Bash behavior.
- `bin/` — Built Go binaries, kept out of version control.

There is no separate `benchmark/` folder; the benchmark helpers live in `bench/`.

## Schema types

| Type | Accepts value? | Repeatable? | Notes |
|---|---|---|---|
| `string` | Yes | No | Raw string; supports `minLength`, `maxLength`, `pattern` |
| `int` | Yes | No | Must be a valid integer |
| `float` | Yes | No | Must be a valid floating-point number |
| `bool` | Yes | No | Must be explicit `true` or `false` (or `1`/`0`/`yes`/`no`) |
| `enum` | Yes | No | Must be one of the values declared in the `enum` field |
| `flag` | No | No | Boolean switch; presence sets true, absence sets false |
| `list` | Yes | **Yes** | Pass the option multiple times; all values collected as strings and joined with `GO_SHOPTS_LIST_DELIM`. Default maxItems=100, no per-item type validation |

**`flag` vs `bool`**: `flag` requires no value (`-v` sets true). `bool` requires an explicit value (`-b true`).

**`list` item types**: All list values are stored as strings regardless of content. There is no per-item type validation.

**`list` implicit constraints**: If `maxItems` is not set, defaults to 100. If `required=true` and `minItems` is not set, defaults to 1.

## Schema fields

| Field | Applies to | Description |
|---|---|---|
| `long` | All | Long option name, required. Allowed chars: `[A-Za-z0-9_-]+` |
| `short` | All | Single alphanumeric short option (optional) |
| `required` | All | `true\|false` — option must be provided |
| `type` | All | One of: `string`, `int`, `float`, `bool`, `enum`, `list`, `flag` |
| `help` | All | Short help text shown in usage output |
| `description` | All | Extended description shown in usage output |
| `default` | All | Default value if option is not provided (mutually exclusive with `required`) |
| `pattern` | `string` | Regex the value must match |
| `failure` | `string` | Human-readable message shown when `pattern` fails |
| `enum` | `enum` | Comma-separated list of allowed values (required for `enum` type) |
| `minLength` | `string` | Minimum character length |
| `maxLength` | `string` | Maximum character length |
| `minItems` | `list` | Minimum number of values (default: 1 when `required=true`, else 0) |
| `maxItems` | `list` | Maximum number of values (default: 100) |

## Built-in pattern validators

The `pattern` field accepts either a raw Go regex or a named built-in validator using `{{ Name }}` syntax:

```bash
long=email,   type=string, pattern={{ EmailAddress }}, failure=must be a valid email;
long=version, type=string, pattern={{ SemVer }}, default=1.0.0;
long=host,    type=string, pattern={{ IPv4Address }}, required=true;
long=port,    type=string, pattern={{ PortNumber }}, default=8080;
```

Spacing inside the braces is flexible: `{{Name}}`, `{{ Name }}`, `{{ Name}}` all work.

An unknown name is a schema error (exit 2). The optional `failure=` field overrides the default error message for any pattern, built-in or raw regex.

| Template | Validates | Backed by |
|---|---|---|
| `EmailAddress` | `user@domain.tld` | regex |
| `URL` | Full URL with scheme (`https://...`) | regex |
| `URLScheme` | Protocol label (`https`, `ftp`, `git+ssh`) | regex |
| `DomainName` | Dot-separated labels (`github.com`) | regex |
| `Subdomain` | Single DNS label (`docs`, `api`) | regex |
| `URLPath` | Starts with `/` (`/users/123`) | regex |
| `QueryString` | Starts with `?` (`?key=val`) | regex |
| `Fragment` | Starts with `#` (`#section-2`) | regex |
| `IPv4Address` | Four-octet address (`192.168.1.1`) | `net.ParseIP` |
| `IPv6Address` | Colon-hex address (`2001:db8::1`) | `net.ParseIP` |
| `CIDRBlock` | IP + prefix mask (`10.0.0.0/24`) | `net.ParseCIDR` |
| `AbsolutePath` | Unix absolute path (`/usr/local/bin`) | regex |
| `RelativePath` | Relative path (`./config/file.yaml`) | regex |
| `GitRef` | Branch, tag, `HEAD`, or `refs/` path | regex |
| `GitSHA` | 7–40 lowercase hex chars | regex |
| `SemVer` | `MAJOR.MINOR.PATCH[-pre][+build]` | string parsing |
| `PortNumber` | Integer 1–65535 | `strconv.Atoi` + bounds |
| `EnvVar` | `SCREAMING_SNAKE_CASE` | regex |

## Validation rules

- `required` and `default` are mutually exclusive.
- `enum` requires `type=enum`.
- `flag` rejects `minLength`/`maxLength`/`pattern`/`enum`.
- `int`, `float`, `bool` reject `minLength`/`maxLength`/`pattern`.
- `minLength <= maxLength`, `minItems <= maxItems`.
- Defaults are validated at schema parse time.

## Output behavior

- Successful parse prints `KEY\tVALUE\n` for each emitted option.
- `KEY` is generated as `SHOPTS_<sanitized-long>` by default (uppercase).
- `GO_SHOPTS_UPCASE=1` uppercases the key name (default on).
- `GO_SHOPTS_PREFIX` overrides the prefix (must be a valid shell identifier prefix, or empty; must not start with `GO_SHOPTS_`).
- `GO_SHOPTS_OUT_DELIM` overrides the field delimiter between key and value (default: tab).
- `list` values are joined with `GO_SHOPTS_LIST_DELIM` (`,`, default).

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | General failure (bad prefix env var, write error, etc.) |
| `2` | Schema error — invalid schema (developer bug) |
| `3` | Parse/validation error — bad arguments (user input error) |

## Help

`-H` or `--help` prints schema-derived usage and exits 0.

`-V` or `--version` prints the version and exits 0.

Both are reserved: schemas that declare `long=help`, `long=version`, `short=H`, or `short=V` are rejected at parse time with exit code 2. This frees `-h` for your own use (e.g. `short=h, long=host`).

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
