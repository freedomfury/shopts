#!/usr/bin/env bash
## Test: Export with toml enum value (invalid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=f, long=format, required=true, type=enum, enum="json,csv,yaml", help=Output format;'

# Should fail: format must be json, csv, or yaml, not toml
if "$BINARY" "$SCHEMA" -f toml >/dev/null 2>&1; then
    exit 1 # Test FAILS if command succeeds
else
    exit 0 # Test PASSES if command fails
fi
