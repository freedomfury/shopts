#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 3 ]]; then
    cat <<USAGE
Usage: $0 N SCHEMA ARGS...
Example:
  $0 1000 '- long: user\n  long: user' -u alice -p secret
USAGE
    exit 2
fi

N="$1"
shift
SCHEMA="$1"
shift
ARGS=("$@")

echo "Running go-getopt ${N} times..."
start_ns=$(date +%s%N)
for i in $(seq 1 "${N}"); do
    go-getopt "${SCHEMA}" "${ARGS[@]}" >/dev/null 2>&1 || {
        echo "iteration ${i}: failure"
        exit 1
    }

end_ns=$(date +%s%N)
elapsed_ns=$((end_ns - start_ns))
awk_total=$(awk "BEGIN {print ${elapsed_ns}/1000000000}")
awk_per_call=$(awk "BEGIN {print ${elapsed_ns}/1000000/${N}}")
printf 'Total: %.3fs\nPer-call: %.3fms\n' "${awk_total}" "${awk_per_call}"
