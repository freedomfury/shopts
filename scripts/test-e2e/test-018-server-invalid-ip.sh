#!/usr/bin/env bash
## Test: Server with invalid IPv4 address (invalid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=h, long=host, required=true, type=string, pattern={{ IPv4Address }}, help=Server host; short=p, long=port, type=int, default=8080, help=Server port;'

# Should fail: invalid IPv4 address
if "$BINARY" "$SCHEMA" -h 999.999.999.999 -p 8080 >/dev/null 2>&1; then
    exit 1 # Test FAILS if command succeeds
else
    exit 0 # Test PASSES if command fails
fi
