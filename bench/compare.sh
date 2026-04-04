#!/usr/bin/env bash
set -euo pipefail

# Compare performance of the Go parser (`shopts`) vs the Bash parser
# Usage: bench/compare.sh N [ARGS...]
# Runs each parser N times with the same args and reports timings.

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 N [ARGS...]"
    exit 2
fi

N="$1"
shift

SHOPTS_BIN="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null && pwd)/bin/shopts"
BASH_PARSER="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null && pwd)/bash-parser.sh"

if [[ ! -x "${SHOPTS_BIN}" ]]; then
    printf 'FATAL: shopts not found or not executable at %s\n' "${SHOPTS_BIN}" >&2
    exit 2
fi

if [[ ! -x "${BASH_PARSER}" ]]; then
    printf 'FATAL: bash-parser not found or not executable at %s\n' "${BASH_PARSER}" >&2
    exit 2
fi

# Schema to use (comma-separated fields, semicolon-terminated entries)
SCHEMA=$(
    cat <<'SCHEMA'
long=user, short=u, required=true, type=string, minLength=3, help=Username for login;
long=pass, short=p, required=true, type=string, minLength=8, help=Password for login;
long=mode, short=m, required=false, type=enum, default=dev, enum="dev,prod,test", help=Mode of operation;
long=tags, short=t, required=false, type=list, minItems=1, maxItems=5, help=Tags (repeatable);
long=verbose, short=v, required=false, type=flag, help=Verbose;
SCHEMA
)

ARGS=("$@")

run_n() {
    local name="$1"
    shift
    local cmd=("$@")
    local n="${N}"
    local start_ns end_ns elapsed_ns
    start_ns=$(date +%s%N)
    for i in $(seq 1 "${n}"); do
        if ! "${cmd[@]}" >/dev/null 2>&1; then
            echo "${name}: iteration ${i} failed" >&2
            return 1
        fi
    done
    end_ns=$(date +%s%N)
    elapsed_ns=$((end_ns - start_ns))
    awk_total=$(awk "BEGIN {print ${elapsed_ns}/1000000000}")
    awk_per_call=$(awk "BEGIN {print ${elapsed_ns}/1000000/${n}}")
    printf '%s: Total: %.3fs Per-call: %.3fms\n' "${name}" "${awk_total}" "${awk_per_call}"
}

echo "Running benchmarks with N=${N}"

echo "-> Go parser"
run_n "shopts" "${SHOPTS_BIN}" "${SCHEMA}" "${ARGS[@]}"

echo "-> Bash parser"
run_n "bash-parser" "${BASH_PARSER}" "${ARGS[@]}"

echo "Done."
