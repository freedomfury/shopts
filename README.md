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
short=p, long=port, type=int, min=1, max=65535, default=8080, help=Port number;
short=v, long=verbose, type=flag, help=Enable verbose output;
'
# Emits SHOPTS_<LONG> for each option (uppercase by default).
# GO_SHOPTS_ is reserved for shopts' own settings; your variables land in SHOPTS_.

while IFS=$'\t' read -r key val; do
  printf -v "$key" '%s' "$val"   # the loop sets each variable; nothing is exported
done < <(GO_SHOPTS_NAME="$0" shopts "$SCHEMA" "$@")
rc=0; wait $! || rc=$?           # needs bash 4.4 or later
[[ $rc -eq 7 ]] && exit 0        # help or version was printed
[[ $rc -ne 0 ]] && exit "$rc"    # 2 = schema error, 3 = bad arguments

printf "User: %s\n" "$SHOPTS_USER"
printf "Port: %d\n" "$SHOPTS_PORT"
```

Arguments are parsed, validated and type-checked, and the read loop sets them as shell variables. If anything is wrong (a missing required option, a bad type, a value that is too short), `shopts` prints a clear error, writes nothing to stdout and exits non-zero. Pass `-H` and your users get help text generated from the schema.

**No `eval`. No dependencies.** Output is tab-delimited records (`KEY\tVALUE\n`), which are safe to consume in Bash without worrying about spaces, quotes or injection. (The `< <(...)` process substitution does run `shopts` in a subshell; what the loop avoids is `eval`.)

For comparison, see [bench/bash-parser.sh](bench/bash-parser.sh), a hand-written Bash implementation of similar logic. Most of that file is the boilerplate that `shopts` eliminates.

## The contract

`shopts` takes a schema and a list of arguments, validates the arguments against the schema, and writes one `KEY<TAB>VALUE` line per option to stdout. That output, plus the exit code, is the whole contract. How you consume the lines (globals, locals, exports, arrays) is up to you; the read loop above is an example, not a requirement.

```
shopts SCHEMA [ARGS...]
```

`shopts` has no switches of its own: everything after the schema belongs to your schema, except the reserved `-H`/`--help` and `-V`/`--version`.

## Schema format

Each entry is a set of `key=value` fields separated by commas (the space after the comma is optional) and terminated by `;`. Entries may span lines, and common indentation is removed.

```bash
SCHEMA='
short=e, long=env, type=enum, enum="dev,prod", required=true, help=Target;
short=v, long=verbose, type=flag, help=Verbose output;
long=target, type=string, positional=1, required=true, help=Deploy target;
long=files, type=list, positional=rest, help=Files to process;
'
```

Values may be written bare or double-quoted. A bare value runs to the next `,` or `;` and is trimmed; it cannot continue onto the next line. Quote values that contain `,` or `;`.

Inside double quotes only two sequences are special: `\"` is a literal quote and `\\` is a literal backslash. Every other backslash is kept as written, so regexes need no double escaping. Go escapes such as `\n` and `\t` are not interpreted.

| Schema text | Value |
|---|---|
| `pattern="^\d{2,4}$"` | `^\d{2,4}$` |
| `help="Say \"hi\", then go"` | `Say "hi", then go` |
| `pattern="^C:\\\\"` | `^C:\\` |
| `pattern=^\d+$` (unquoted) | `^\d+$` |

Schema errors give the line and column, counted in the schema text as you wrote it:

```
shopts: schema line 4, col 23: unknown field "patern" (did you mean "pattern"?)
```

### Fields

| Field | Applies to | Meaning |
|---|---|---|
| `long` | all | Required. Starts with a letter, then letters, digits, `_` (no hyphens). Typed with dashes: `long=dry_run` is `--dry-run` (see [Long option names](#long-option-names)) |
| `short` | all | Optional single letter or digit. `H` and `V` are reserved |
| `type` | all | Required. `string`, `int`, `float`, `bool`, `enum`, `flag`, `list` |
| `required` | all except `flag` | `true` or `false`; cannot be combined with `default` |
| `default` | all except `flag` | Used when the option is not given. Validated when the schema is parsed. For a `list`, items are comma-separated |
| `help` | all | One-line help text |
| `description` | all | Extra help text, shown under the option |
| `enum` | `enum` | Allowed values, comma-separated |
| `pattern` | `string`, `list` | Regex or `{{ Name }}` validator; must match the whole value (each item, for a list) |
| `failure` | with `pattern` | Message shown when the pattern fails |
| `minLength`, `maxLength` | `string` | Length limits, counted in characters (Unicode code points), not bytes |
| `min`, `max` | `int`, `float` | Inclusive numeric bounds, e.g. `min=1, max=65535` |
| `minItems`, `maxItems` | `list` | Item count limits. `maxItems` defaults to 100; `minItems` defaults to 1 when `required=true` |
| `positional` | `string`, `list` | `1`, `2`, … or `rest`; see [Positional arguments](#positional-arguments) |
| `define` | own entry | Declares a named pattern; see [Pattern validators](#pattern-validators) |

### Flags

A flag is a switch without a value: given means `true`, absent means `false`. That is the whole rule, so a flag has no `default=` and cannot be `required`.

Nothing is implied: a flag answers only to the name written in the schema. For behavior that is on by default, name the flag for turning it off:

```bash
long=no_cache, type=flag, help=Skip the cache;
```

| Arguments | `SHOPTS_NO_CACHE` | Behavior |
|---|---|---|
| (none) | `false` | cache on |
| `--no-cache` | `true` | cache off |

`no_` has no special meaning to shopts. To accept both `--dry-run` and `--no-dry-run`, declare both flags; they are independent and emit `SHOPTS_DRY_RUN` and `SHOPTS_NO_DRY_RUN`, and your script decides what giving both means.

### Types

| Type | Takes a value? | Repeatable? | Emitted as |
|---|---|---|---|
| `string` | Yes | No | As given |
| `int` | Yes | No | Plain decimal: `007` → `7`, `+5` → `5`, so bash never reads it as octal |
| `float` | Yes | No | As given; must be a finite number |
| `bool` | Yes | No | `true` or `false`. Accepts Go's set: `1`, `t`, `T`, `TRUE`, `true`, `True`, and the `0`/`f`/`false` forms. `yes`/`no` are rejected |
| `enum` | Yes | No | As given; must be one of the `enum` values |
| `flag` | No | No | `true` when given, `false` when not |
| `list` | Yes | **Yes** | Items joined by `GO_SHOPTS_LIST_DELIM` |

**`flag` vs `bool`**: `flag` takes no value (`-v` sets true). `bool` takes an explicit value (`-b true`).

## Argument parsing

Options are written as `--name value`, `--name=value`, `-x value` or `-x=value`. (`-x=value` is not POSIX; it is supported as a convenience.)

- A value that looks like an option is still taken as the value: `-u --pass` sets `-u` to `--pass`. Use `-u=value` to make it obvious.
- Flags take no value; `--verbose=true` is an error.
- Repeating a non-list option (flags included) is an error. List options collect every occurrence.
- `-H`/`--help` and `-V`/`--version` are recognised anywhere in the arguments (before `--`). Lowercase `-h` and `-v` are free for your schema.
- Short options cannot be bundled (`-abc`).
- Flags take no value and answer only to their own name; see [Flags](#flags).
- Unknown options and bad values are all reported together, not one at a time.

### Long option names

Schema names use `_`; on the command line underscores become dashes. `long=dry_run` is typed `--dry-run` and `long=no_cache` is typed `--no-cache`. Output names are unaffected: the variables are still `SHOPTS_DRY_RUN` and `SHOPTS_NO_CACHE`.

This is one global setting, never a mix: with dashes on (the default), `--dry_run` is an unknown option and the error suggests `--dry-run`. Set `GO_SHOPTS_DASH=0` to type names exactly as written in the schema (`--dry_run`, `--no_cache`) instead. Help and error messages always show the spelling users must type.

### Positional arguments

An entry with `positional=` binds bare arguments instead of a flag:

```bash
long=target, type=string, positional=1, required=true, help=Deploy target;
long=files, type=list, positional=rest, help=Files to process;
```

- `positional=N` binds the Nth bare argument; numbering starts at 1 with no gaps.
- `positional=rest` collects all remaining bare arguments into a list. At most one entry may use it, and it must be `type=list`.
- Everything after `--` is treated as a bare argument.
- Bare arguments with no matching positional entry are an error.
- Positional entries use the same `required`, `default`, `pattern` and length rules as options. They cannot have a `short` flag, and a required positional cannot follow an optional one.

## Configuration

`shopts` is configured only through environment variables under `GO_SHOPTS_`. The prefix is deliberately distinct from the default output prefix `SHOPTS_`, so tool settings and script variables are easy to tell apart. Set them inline for one call: `GO_SHOPTS_PREFIX=OPT_ shopts "$SCHEMA" "$@"`.

| Variable | Default | Effect |
|---|---|---|
| `GO_SHOPTS_PREFIX` | `SHOPTS_` | Output variable prefix. May be empty; must not start with `GO_SHOPTS_` |
| `GO_SHOPTS_UPCASE` | `1` | Uppercase output names. Set to `0` to keep option names as written |
| `GO_SHOPTS_DASH` | `1` | Type long names with dashes (`--dry-run` for `long=dry_run`). Set to `0` to type them as written (`--dry_run`) |
| `GO_SHOPTS_OUT_DELIM` | `{{ tab }}` | Separator between key and value: a [named delimiter](#delimiters) or any string without a newline |
| `GO_SHOPTS_LIST_DELIM` | `,` | Separator between list items: a [named delimiter](#delimiters) or any string without a newline |
| `GO_SHOPTS_NAME` | unset | Program name shown in help and errors, e.g. `GO_SHOPTS_NAME="$0"` |

An invalid setting exits 1.

## Output

On success, `shopts` writes one line per emitted option to stdout, in schema order:

```
<PREFIX><LONG><OUT_DELIM><VALUE>\n
```

The name is the prefix plus the long name, uppercased by default. An option is emitted when it was given, has a default, or is a flag (absent flags emit `false`). An option with none of these is not emitted. Values are normalized as shown in [Types](#types), so you can compare them directly.

A value may contain anything except a newline, which would end its line; a value with a newline is rejected with exit 3. On any error nothing is written to stdout.

### Delimiters

Keys contain only letters, digits and `_`, so the first delimiter on a line always ends the key, and everything after it is the value, even if the value contains the delimiter again.

Any string without a newline can be a delimiter. Characters that are hard or impossible to put in an environment variable have names, written like validators:

<!-- delimiters:begin (generated from pkg/shopts/run.go; run make readme) -->
| Name | Character |
|---|---|
| `{{ tab }}` | Tab. The default output delimiter; works with `cut`, `awk -F'\t'` and `read` |
| `{{ null }}` | NUL byte. No argument can contain it, so every value reads back exactly |
<!-- delimiters:end -->

With the default tab, read the output like this:

```bash
while IFS=$'\t' read -r key val; do
  printf -v "$key" '%s' "$val"
done < <(shopts "$SCHEMA" "$@")
```

bash's `read` trims leading and trailing delimiter characters from the value, so a value that starts or ends with a tab loses them. For values that must survive exactly, use `{{ null }}`: no argument can contain a NUL byte, and two `read`s split each line at it:

```bash
while IFS= read -r -d '' key && IFS= read -r val; do
  printf -v "$key" '%s' "$val"
done < <(GO_SHOPTS_OUT_DELIM='{{ null }}' shopts "$SCHEMA" "$@")
```

Choosing a delimiter that suits your tools is up to you. For example, `read` treats each character of `IFS` as a separate delimiter, so a multi-character delimiter such as `:::` works with `awk -F':::'` but not with `read`.

## Exit codes

| Code | Meaning | stdout |
|---|---|---|
| `0` | Parsed successfully | `KEY<TAB>VALUE` lines |
| `1` | General failure (bad `GO_SHOPTS_` setting, write error) | empty |
| `2` | Schema error: the script author's mistake | empty |
| `3` | Argument error: the user's mistake | empty |
| `7` | Help or version was printed; the caller should stop | empty |

Checks run in this order, and the first to fail decides the code:

1. `-V`/`--version` (7). It needs neither settings nor the schema, so it works even when they are broken.
2. The settings help shows, `GO_SHOPTS_NAME` and `GO_SHOPTS_DASH` (1).
3. The schema (2).
4. `-H`/`--help` (7). It needs the schema, but ignores the other settings.
5. The other settings (1).
6. The arguments (3).

When both `-H` and `-V` are given, the first one wins. An internal error exits 1, never 2.

Exit 7 lets your script tell "help was shown" apart from "parsed successfully". Map it back to 0 for your own user:

```bash
[[ $rc -eq 7 ]] && exit 0
[[ $rc -ne 0 ]] && exit "$rc"
```

## Help and version

`-H`/`--help` prints usage generated from the schema to stderr and exits 7. It stays on stderr because stdout is the data channel. Help describes your script only: its name (from `GO_SHOPTS_NAME`), its arguments and its options.

```
Usage: deploy.sh [OPTIONS] <target> [files...]

Arguments:
  <target>            Deploy target; string; required
  [files...]          Files to process; list

Options:
  -e, --env <value>   Target; enum; required; allowed: dev, prod
  -v, --verbose       Verbose output; flag
  -H, --help          Show this help
```

`-V`/`--version` prints the shopts version to stderr and exits 7, from any position in the arguments. `shopts --version` with no schema works too.

Both are reserved: schemas that declare `long=help`, `long=version`, `short=H` or `short=V` are rejected (exit 2).

## Pattern validators

Every pattern must match the whole value, whether it is a built-in, a defined name or an inline regex: `pattern=[0-9]+` rejects `abc1`. The regex dialect is Go's [RE2](https://github.com/google/re2/wiki/Syntax).

Use a built-in validator with `{{ Name }}` (spacing inside the braces is flexible):

```bash
long=email,   type=string, pattern={{ EmailAddress }};
long=release, type=string, pattern={{ SemVer }}, default=1.0.0;
long=host,    type=string, pattern={{ IPv4Address }}, required=true;
```

Define your own named patterns with a `define` entry and use them the same way. A `define` entry takes only `define`, `pattern` (a regex) and `failure`. Defined names cannot reuse a built-in name, so `{{ SemVer }}` means the same thing everywhere.

```bash
define=Ticket, pattern=^[A-Z]+-[0-9]+$, failure=must look like ABC-123;
long=ticket, type=string, pattern={{ Ticket }};
```

When a pattern fails, the message is the option's `failure=` if set, then the validator's own message, then a generic "must match the pattern …". An unknown name is a schema error (exit 2).

<!-- validators:begin (generated from pkg/shopts/patterns.go; run make readme) -->
| Template | Accepts | Example |
|---|---|---|
| `EmailAddress` | Email address | `user@example.com` |
| `URL` | Full URL with a scheme | `https://example.com/path?q=1` |
| `URLScheme` | URL scheme | `https` |
| `DomainName` | Dot-separated DNS labels | `github.com` |
| `Subdomain` | Single DNS label | `docs` |
| `URLPath` | URL path starting with `/` | `/users/123` |
| `QueryString` | Query string starting with `?` | `?key=val` |
| `Fragment` | URL fragment starting with `#` | `#section-2` |
| `IPv4Address` | IPv4 address | `192.168.1.1` |
| `IPv6Address` | IPv6 address, including IPv4-mapped | `2001:db8::1` |
| `CIDRBlock` | IP address with a prefix length | `10.0.0.0/24` |
| `AbsolutePath` | Path starting with `/` | `/usr/local/bin` |
| `RelativePath` | Path not starting with `/` or `-` | `config/file.yaml` |
| `GitRef` | Branch, tag, `HEAD` or `refs/` path, by git's naming rules | `main` |
| `GitSHA` | 7–40 lowercase hex characters | `abc1234` |
| `SemVer` | `MAJOR.MINOR.PATCH[-pre][+build]` | `1.0.0` |
| `PortNumber` | Integer 1–65535, digits only | `8080` |
| `EnvVar` | `SCREAMING_SNAKE_CASE` name | `PATH` |
<!-- validators:end -->

## Project Structure

- `cmd/shopts/` — CLI entrypoint: argv in, exit code out.
- `pkg/shopts/` — the parser, one file per concern:
  - `schema.go` — lexer and parser, with positioned errors
  - `fields.go` — field table: how each field is read and which types it applies to
  - `validate.go` — the one `validate` function every value goes through
  - `patterns.go` — validator registry; built-ins are entries
  - `args.go` — scans argv into raw values (options, positionals)
  - `help.go` — help text generated from the schema
  - `run.go` — the pipeline: scan, resolve defaults, validate, emit
- `scripts/` — Bash acceptance tests; `scripts/test-contract.sh` covers the contract item by item.
- `bench/` — Bash reference parser plus benchmark scripts.
- `bin/` — release and example scripts (tracked) and the built `bin/shopts` binary (untracked).

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
| `make readme` | Regenerate the README validator table from the registry. |
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

- `./scripts/test.sh` builds `bin/shopts` if needed and checks a basic successful parse path.
- `./scripts/test-negative.sh` verifies help output and a representative validation failure path.
- `./scripts/test-extensive.sh` exercises the wider type matrix, defaults, repeated list values, flags, and delimiter handling.
- `./scripts/test-contract.sh` is the acceptance suite for the contract: each item of the spec has at least one check.

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
