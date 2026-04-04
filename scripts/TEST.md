# E2E Testing Documentation

## Overview

The shopts project includes a comprehensive End-to-End (E2E) test suite consisting of 22 deterministic test cases that validate the CLI argument parser across multiple real-world scenarios and error conditions.

## Test Architecture

### Directory Structure
- **scripts/test-e2e/** — 22 individual test files (test-001-*.sh through test-022-*.sh)
- **scripts/run-e2e-tests.sh** — Parallel test runner with background job pool
- **scripts/test.sh** — Go unit tests runner
- **scripts/test-negative.sh** — Negative test cases
- **scripts/test-extensive.sh** — Extensive test cases

### Test Execution

Tests are executed in parallel using a background job pool with configurable parallelism (defaults to CPU count via `nproc`). Each test:
1. Runs independently in the background
2. Captures output to a per-test result file
3. Logs results with status (PASS/FAIL)
4. Results are sorted by test number in final output

Run all E2E tests with:
```bash
make test-e2e
```

## Test Coverage

### Valid Scenarios (Tests 001-016)

#### Authentication Schema (Tests 001-004)
- **test-001**: Short option `-u` with username
- **test-002**: Long option `--username` with username
- **test-003**: Mixed short and long options (username + password)
- **test-004**: Boolean flag `--verbose` enabled

#### Server Configuration Schema (Tests 005-007)
- **test-005**: Host configuration only
- **test-006**: Host and port configuration
- **test-007**: Protocol selection and SSL verification

#### Data Export Schema (Tests 008-010)
- **test-008**: Output format selection (JSON)
- **test-009**: Output file path specification
- **test-010**: Compression option with format selection

#### Database Connection Schema (Tests 011-013)
- **test-011**: Minimal database name
- **test-012**: Host configuration
- **test-013**: Connection pooling and timeout settings

#### API Client Schema (Tests 014-016)
- **test-014**: API endpoint configuration
- **test-015**: HTTP method selection (POST)
- **test-016**: Multiple tag parameters (tags list)

### Invalid Scenarios (Tests 017-022)

#### String Validation Errors (Tests 017-020)
- **test-017**: String fails minLength constraint (too short username)
- **test-018**: Invalid IPv4 address pattern (malformed host)
- **test-019**: Enum validation fails with invalid format (XML not in: json, csv, yaml)
- **test-020**: Invalid URL pattern (malformed output path)

#### Enum Negative Testing (Tests 021-022)
- **test-021**: Export format enum rejects TOML (only json, csv, yaml allowed)
- **test-022**: API method enum rejects PATCH (only GET, POST, PUT, DELETE allowed)

## Enum Validation

The test suite includes comprehensive enum validation:

| Schema | Field | Valid Values | Invalid Test |
|--------|-------|--------------|--------------|
| export | format | json, csv, yaml | test-019 (xml), test-021 (toml) |
| api | method | GET, POST, PUT, DELETE | test-022 (PATCH) |

## Schemas Used

### 1. Authentication
```
long=username, short=u, required=true, type=string, minLength=3
long=password, short=p, required=false, type=string, minLength=6
long=verbose, short=v, required=false, type=bool
```

### 2. Server
```
long=host, required=true, type=string, pattern=IPv4
long=port, required=false, type=int
long=protocol, required=false, type=string, enum=http|https
long=sslverify, required=false, type=bool
```

### 3. Export
```
long=format, required=true, type=string, enum=json|csv|yaml
long=output, required=false, type=string, pattern=path
long=compress, required=false, type=bool
```

### 4. Database
```
long=dbname, required=true, type=string, minLength=1
long=host, required=false, type=string, pattern=IPv4
long=poolsize, required=false, type=int
long=timeout, required=false, type=int
```

### 5. API
```
long=endpoint, required=true, type=string, pattern=url
long=method, required=false, type=string, enum=GET|POST|PUT|DELETE
long=tags, required=false, type=list, listType=string
```

## Test Execution & Linting

### Quality Assurance
All bash test scripts are validated with shellcheck for:
- Proper quoting and variable expansion
- Correct command syntax
- Security best practices
- Shell portability

Run linting with:
```bash
make lint-all
```

This runs:
- Bash linting (shellcheck on all shell scripts)
- Go linting (golangci-lint on source code)

### Test Targets

| Target | Purpose |
|--------|---------|
| `make test` | Run Go unit tests + Bash basic tests |
| `make test-go` | Run Go unit tests only |
| `make test-bash` | Run basic bash tests |
| `make test-e2e` | Run 22 E2E tests in parallel |
| `make test-all` | Run all tests (Go, Bash basic, + E2E) |

### Lint Targets

| Target | Purpose |
|--------|---------|
| `make lint-bash` | Lint bash scripts with shellcheck |
| `make lint` | Lint bash + Go code |
| `make lint-all` | Lint bash + Go code (comprehensive) |

## Test Result Interpretation

Tests output:
```
test-001 [auth-short]: PASS
test-002 [auth-long]: PASS
...
Results: 22 passed, 0 failed, 22 total (ran with N parallel jobs)
```

- **PASS**: Test executed successfully with expected behavior
- **FAIL**: Test did not produce expected output or exited with error

## Implementation Details

### Parallel Execution
The test runner implements a background job pool:
```bash
# Jobs are queued and executed in parallel (limited by CPU cores)
for test_file in scripts/test-e2e/test-*.sh; do
    # Wait for available job slot
    while [ $(jobs -r -p | wc -l) -ge "$MAX_JOBS" ]; do
        sleep 0.1
    done
    # Launch test in background
    bash "$test_file" &
done
wait
```

### Cleanup & Error Handling
- Temporary directories created by tests are cleaned up automatically
- Process cleanup uses proper trap handlers
- Exit codes are captured and reported

## Contributing Tests

When adding new E2E tests:
1. Create `scripts/test-e2e/test-NNN-description.sh` following the naming convention
2. Start with `#!/bin/bash` and `set -euo pipefail`
3. Use `SCHEMA` variable for CLI definition
4. Execute binary with `./bin/shopts "$SCHEMA" "${@}"`
5. Validate output with appropriate assertions
6. Make file executable: `chmod +x scripts/test-e2e/test-NNN-*.sh`
7. Run `make test-e2e` to verify
8. Run `make lint-bash` to ensure shellcheck compliance
