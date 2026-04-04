#!/usr/bin/env bash
## Test: API with invalid URL (no scheme) (invalid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=e, long=endpoint, required=true, type=string, pattern={{ URL }}, help=API endpoint; short=m, long=method, type=enum, enum="GET,POST,PUT,DELETE", default=GET, help=HTTP method;'

# Should fail: URL must have scheme (http://, https://, etc)
if "$BINARY" "$SCHEMA" -e api.example.com/data -m POST >/dev/null 2>&1; then
    exit 1 # Test FAILS if command succeeds
else
    exit 0 # Test PASSES if command fails
fi
