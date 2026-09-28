#!/usr/bin/env bash
## Test: Server config with just host (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=h, long=host, required=true, type=string, pattern={{ IPv4Address }}, help=Server host; short=p, long=port, type=int, default=8080, help=Server port;'

expect_ok "$SCHEMA" -h 192.168.1.1 <<'EOF'
SHOPTS_HOST=192.168.1.1
SHOPTS_PORT=8080
EOF
