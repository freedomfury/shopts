#!/usr/bin/env bash
## Test: Authentication with username too short (invalid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=u, long=username, required=true, type=string, minLength=3, help=Username; short=p, long=pass, required=true, type=string, minLength=6, help=Password;'

# Should fail: username has only 2 chars, minLength=3
if "$BINARY" "$SCHEMA" -u ab -p password123 >/dev/null 2>&1; then
    exit 1 # Test FAILS if command succeeds
else
    exit 0 # Test PASSES if command fails (exit 3)
fi
