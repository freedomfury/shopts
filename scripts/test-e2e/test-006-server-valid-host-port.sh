#!/usr/bin/env bash
## Test: Server config with host and port (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=h, long=host, required=true, type=string, pattern={{ IPv4Address }}, help=Server host; short=p, long=port, type=int, default=8080, help=Server port; short=c, long=protocol, type=enum, enum="http,https", default=http, help=Protocol;'

expect_ok "$SCHEMA" --host 10.0.0.1 --port 9000 <<'EOF'
SHOPTS_HOST=10.0.0.1
SHOPTS_PORT=9000
SHOPTS_PROTOCOL=http
EOF
