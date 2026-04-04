#!/usr/bin/env bash
## Test: API with invalid HTTP method enum (invalid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=e, long=endpoint, required=true, type=string, pattern={{ URL }}, help=API endpoint; short=m, long=method, type=enum, enum="GET,POST,PUT,DELETE", default=GET, help=HTTP method;'

# Should fail: method must be GET, POST, PUT, or DELETE, not PATCH
if "$BINARY" "$SCHEMA" -e https://api.example.com -m PATCH >/dev/null 2>&1; then
    exit 1 # Test FAILS if command succeeds
else
    exit 0 # Test PASSES if command fails
fi
