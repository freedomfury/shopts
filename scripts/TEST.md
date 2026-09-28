# Testing

shopts has Go unit tests and five bash suites that run the built binary. The bash suites
are the acceptance tests: they exercise the contract (stdout, stderr, exit code) the way a
calling script sees it.

## Suites

| Suite | What it checks |
|---|---|
| `go test ./...` | Unit tests for each file in `pkg/shopts`: lexer and parser, field rules, `validate`, the validator registry, argument scanning, help, and `Run` end to end. Each validator's `Valid`/`Invalid` examples run as tests, and the README's generated tables must match the code. |
| `scripts/test.sh` | The basic success path through the README read loop: shopts exits 0 and every variable has the expected value. |
| `scripts/test-negative.sh` | The exact help text (`--help`, exit 7) and the exact error output for a validation failure (exit 3). |
| `scripts/test-extensive.sh` | Every type, defaults, lists, flags, patterns, length limits, the list delimiter and the prefix setting. Each value is checked, and each run starts with no `SHOPTS_` variables left over. |
| `scripts/test-contract.sh` | The acceptance suite for the contract spec: one or more checks per spec item, plus a check for each code-review fix. Each check asserts the exit code, stdout byte for byte (including the final newline), and a stderr message. |
| `scripts/test-e2e/` | 22 realistic scenarios (auth, server, export, database, API), run in parallel by `scripts/run-e2e-tests.sh`. |

Every suite clears `GO_SHOPTS_*` settings from its environment first, so settings you have
exported in your shell can't change the results.

## Running

| Target | Runs |
|---|---|
| `make test` | Go tests and the bash suites |
| `make test-go` | `go test -race ./...` |
| `make test-bash` | `test.sh`, `test-negative.sh`, `test-extensive.sh`, `test-contract.sh` |
| `make test-e2e` | The e2e scenarios |
| `make test-all` | All of the above |
| `make lint-bash` | shellcheck on every `.sh` file in the repo |
| `make readme` | Regenerates the README tables that `go test` checks |

Run the scripts from the project root. Each one builds `bin/shopts` if it is missing, but
does not rebuild it if it is stale; `make` rebuilds it when the Go sources change.

CI runs the Go tests, all bash suites, the e2e scenarios, golangci-lint and
`make lint-bash`.

## Contract suite (`scripts/test-contract.sh`)

Checks are one line each, using three helpers:

- `ok NAME STDOUT SCHEMA ARGS...`: exit 0, and stdout is exactly the given lines.
- `err NAME RC MESSAGE SCHEMA ARGS...`: exit `RC`, empty stdout, and `MESSAGE` in stderr.
- `envrun NAME RC WANT VAR=VALUE... -- SCHEMA ARGS...`: like `ok` (when `RC` is 0) or
  `err`, with the given settings in the environment of that one run.

Sections follow the spec: schema format; parser positions and quoting; fields; invocation
and configuration; argument parsing; output; exit codes; help; pattern validators; code
review fixes; and the README read loops (tab and `{{ null }}`).

## E2E scenarios (`scripts/test-e2e/`)

Each `test-NNN-*.sh` file holds one scenario: a schema, the arguments, and the exact
expected result, checked with the helpers in `scripts/test-e2e/lib.sh`:

```bash
expect_ok "$SCHEMA" -u alice -p password123 -v <<'EOF'
SHOPTS_USERNAME=alice
SHOPTS_PASS=password123
SHOPTS_VERBOSE=true
SHOPTS_REMEMBER=false
EOF

expect_fail 3 'invalid value for --format: must be one of: json, csv, yaml' "$SCHEMA" -f xml
```

- `expect_ok` requires exit 0, empty stderr, and exactly the listed lines on stdout. In
  each expected line, the first `=` stands for the tab between key and value.
- `expect_fail` requires the exact exit code, empty stdout, and the message in stderr. A
  typo in the test's own schema therefore fails the test (exit 2, want 3) instead of
  passing by accident.

| Test | Scenario | Expects |
|---|---|---|
| test-001 | Authentication with short options | output |
| test-002 | Authentication with long options | output |
| test-003 | Authentication with mixed options | output |
| test-004 | Authentication with a bool option | output |
| test-005 | Server config with just host | output |
| test-006 | Server config with host and port | output |
| test-007 | Server config with protocol | output |
| test-008 | Data export with just format | output |
| test-009 | Data export with format and output | output |
| test-010 | Data export with compress flag | output |
| test-011 | Database with required fields only | output |
| test-012 | Database with host override | output |
| test-013 | Database with pool and timeout settings | output |
| test-014 | API client with just endpoint | output |
| test-015 | API client with endpoint and method | output |
| test-016 | API client with a tags list | output |
| test-017 | Username too short | exit 3 |
| test-018 | Invalid IPv4 address | exit 3 |
| test-019 | Format not in the enum (xml) | exit 3 |
| test-020 | URL without a scheme | exit 3 |
| test-021 | Format not in the enum (toml) | exit 3 |
| test-022 | HTTP method not in the enum | exit 3 |

The runner runs the tests in parallel (one job per CPU by default) and prints a summary.
For a failing test it also prints what the test reported: the expected-vs-actual diff, or
the exit code and the binary's stdout and stderr.

```bash
scripts/run-e2e-tests.sh [BINARY] [NUM_PARALLEL]
```

## Adding tests

- **A contract rule:** add an `ok`, `err` or `envrun` line to the matching section of
  `scripts/test-contract.sh`.
- **An e2e scenario:** copy an existing `scripts/test-e2e/test-NNN-*.sh`, use the next
  number, set the schema and arguments, and write the expected lines or error. The runner
  picks up every `test-*.sh` file; no registration is needed.
- **A built-in validator:** add its entry, with `Valid` and `Invalid` examples, to the
  registry in `pkg/shopts/patterns.go`, then run `make readme`.

Run `make lint-bash` after changing any script.
