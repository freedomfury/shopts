#!/usr/bin/env bash
## Test: Server with invalid IPv4 address (invalid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=h, long=host, required=true, type=string, pattern={{ IPv4Address }}, help=Server host; short=p, long=port, type=int, default=8080, help=Server port;'

expect_fail 3 'invalid value for --host: must be a valid IPv4 address' "$SCHEMA" -h 999.999.999.999 -p 8080
