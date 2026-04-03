# go-shopts

`go-shopts` is a schema-driven Go CLI parser that reads an inline schema definition,
validates and pairs provided arguments, and emits shell-safe output as `KEY\0VALUE\n` records.

This project also includes Bash scripts for testing and benchmarking, and a Bash reference parser for comparison.

## Features

- Schema-driven CLI argument parsing from a text-based schema string.
- Built-in validation for required args, types, enum, min/max length, patterns,
  and list item counts.
- `flag` type support (boolean switch that is true when present).
- `list` type support (repeatable option values joined with `GO_GETOPT_LIST_DELIM`).
- Environment-controlled output naming: `GO_GETOPT_PREFIX`, `GO_GETOPT_UPCASE`.
- `-h`/`--help` for schema-derived usage text.
- No shell eval; output is intended for safe `read -d $'\0'` consumer patterns.

- Bash reference parser and performance comparison scripts included.

## Usage

```
go-shopts "$SCHEMA" [OPTIONS...]
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
SCHEMA='\
short=u;long=username;required=true;type=string;help=Username;minLength=3;\
short=p;long=password;required=true;type=string;help=Password;minLength=6;\
short=v;long=verbose;type=flag;help=Verbose mode;\
short=t;long=tags;type=list;minItems=1;maxItems=5;\
'

./go-shopts "$SCHEMA" -u alice -p s3cret -v -t a -t b
```

### Notes

- `flag` options do not accept a value (`-v` sets true, absence is false in output).
- `--` terminates options and disallows trailing positional args.
- `-abc` bundles are not supported; only single-letter short options.
- Unknown options and invalid schemas produce stderr errors and exit code 1.

## Project Structure

- `cmd/shops/` — Go CLI entrypoint (main.go)
- `pkg/shopts/` — Go parser implementation and logic
- `scripts/` — Bash test scripts (`test.sh`, `test-negative.sh`, `test-extensive.sh`)
- `bench/` — Bash reference parser and benchmark scripts
- `bin/` — Built Go binaries (git-ignored)

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
- `flag` ignores `minLength/maxLength/pattern/enum`.
- `minLength <= maxLength`, `minItems <= maxItems`.
- Validate defaults at schema parse time.

## Output behavior

- Successful parse prints `KEY\0VALUE\n` for each emitted option.
- `KEY` is generated as `GO_GETOPT_<sanitized-long>` by default.
- `GO_GETOPT_UPCASE=1` uppercases the key name.
- `GO_GETOPT_PREFIX` overrides the prefix (can be empty).
- `list` values are joined with `GO_GETOPT_LIST_DELIM` (`,`, default).

## Help

`-h` or `--help` prints schema-derived usage and exits 0.

## Quick test snippet

```bash
export GO_GETOPT_UPCASE=1
SCHEMA='\
short=u;long=user;required=true;type=string;help=User;\
short=v;long=verbose;type=flag;help=Verbose;\
'

while IFS= read -r -d $'\0' k && IFS= read -r v; do
  printf -v "$k" '%s' "$v"
  declare -xr "${k#GO_GETOPT_}"="$v"
done < <(./go-shopts "$SCHEMA" -u alice -v)

printf 'USER=%s\n' "$USER"
printf 'VERBOSE=%s\n' "$VERBOSE"
```

## Testing

- Go unit tests: `go test ./...`
- Bash test scripts (run from project root):
  - `./scripts/test.sh` — basic positive path
  - `./scripts/test-negative.sh` — help/validation/negative path
  - `./scripts/test-extensive.sh` — all types, edge cases

All test scripts will build the Go binary if missing. Ensure `bin/shops` is up to date.

## Bash Reference Parser & Benchmarks

- `bench/bash-parser.sh` — Bash implementation of the same CLI schema for comparison
- `bench/benchmark.sh` — Run Go parser N times for timing
- `bench/compare.sh` — Compare Go and Bash parser performance

Example:

```bash
./bench/benchmark.sh 1000 "$SCHEMA" -u alice -p s3cret
./bench/compare.sh 1000 -u alice -p s3cret
```

## Code Quality

- All Bash scripts are checked with `shellcheck` (run: `shellcheck scripts/*.sh bench/*.sh`)
- Go code is linted with `golangci-lint` (run: `golangci-lint run ./...`)

## Go Module

- Module: `github.com/freedomfury/shopts`
- Go version: 1.24.4
