#!/usr/bin/env bash
## Run all E2E tests in scripts/test-e2e/ in parallel
## Usage: run-e2e-tests.sh [BINARY] [NUM_PARALLEL]

BINARY=${1:-bin/shopts}
NUM_PARALLEL=${2:-$(nproc 2>/dev/null || echo 4)}

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# Check binary exists
if [[ ! -x "$BINARY" ]]; then
    echo "Error: binary not found or not executable: $BINARY" >&2
    exit 1
fi

# Find all test scripts
TEST_DIR="scripts/test-e2e"
if [[ ! -d "$TEST_DIR" ]]; then
    echo "Error: test directory not found: $TEST_DIR" >&2
    exit 1
fi

# Temp directory for results
results_dir=$(mktemp -d)
trap 'rm -rf "$results_dir"' EXIT

# Run a single test
run_test() {
    local test_file="$1"
    local binary="$2"
    local results_dir="$3"
    local test_name
    test_name=$(basename "$test_file" .sh)

    # Run the test and capture result
    set +e
    "$test_file" "$binary" >/dev/null 2>&1
    exit_code=$?
    set -e

    # Write result
    if [[ $exit_code -eq 0 ]]; then
        echo "PASS|$test_name" >"$results_dir/$test_name.result"
    else
        echo "FAIL|$test_name|$exit_code" >"$results_dir/$test_name.result"
    fi
}

export -f run_test

echo "Running E2E tests in parallel (up to $NUM_PARALLEL at a time)..."
echo "==============================================================================="

# Run all tests in parallel using background job pool
declare -a job_pids
for test_file in "$TEST_DIR"/test-*.sh; do
    # Wait if we hit the parallel limit
    while [[ $(jobs -r | wc -l) -ge $NUM_PARALLEL ]]; do
        sleep 0.001
    done

    # Start test in background
    run_test "$test_file" "$BINARY" "$results_dir" &
    job_pids+=($!)
done

# Wait for all background jobs
for pid in "${job_pids[@]}"; do
    wait "$pid" 2>/dev/null || true
done

# Process results
passed=0
failed=0
failed_tests=""

for result_file in "$results_dir"/*.result; do
    if [[ ! -f "$result_file" ]]; then
        continue
    fi

    IFS='|' read -r status test_name exit_code <"$result_file"

    if [[ "$status" == "PASS" ]]; then
        ((passed++))
        printf "%b✓%b %s\n" "$GREEN" "$NC" "$test_name"
    else
        ((failed++))
        failed_tests="$failed_tests $test_name"
        printf "%b✗%b %s (exit: %s)\n" "$RED" "$NC" "$test_name" "$exit_code"
    fi
done

total=$((passed + failed))
echo "==============================================================================="
printf "\nResults: ${GREEN}%d passed${NC}, ${RED}%d failed${NC}, %d total (ran with %d parallel jobs)\n" "$passed" "$failed" "$total" "$NUM_PARALLEL"

if [[ $failed -eq 0 ]]; then
    echo "All tests passed! ✓"
    exit 0
else
    echo "Failed tests:$failed_tests"
    exit 1
fi
