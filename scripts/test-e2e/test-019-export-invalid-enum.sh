#!/usr/bin/env bash
## Test: Export with invalid format enum (invalid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=f, long=format, required=true, type=enum, enum="json,csv,yaml", help=Output format; short=o, long=output, type=string, pattern={{ RelativePath }}, help=Output file;'

# Should fail: format must be json, csv, or yaml
if "$BINARY" "$SCHEMA" -f xml -o ./export.xml >/dev/null 2>&1; then
    exit 1 # Test FAILS if command succeeds
else
    exit 0 # Test PASSES if command fails
fi
