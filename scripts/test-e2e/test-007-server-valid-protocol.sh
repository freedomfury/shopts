#!/usr/bin/env bash
## Test: Server config with protocol (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=h, long=host, required=true, type=string, pattern={{ IPv4Address }}, help=Server host; short=p, long=port, type=int, default=8080, help=Server port; short=c, long=protocol, type=enum, enum="http,https", default=http, help=Protocol; short=s, long=sslverify, type=bool, default=true, help=Verify SSL;'

expect_ok "$SCHEMA" -h 172.16.0.1 -p 8443 -c https -s false <<'EOF'
SHOPTS_HOST=172.16.0.1
SHOPTS_PORT=8443
SHOPTS_PROTOCOL=https
SHOPTS_SSLVERIFY=false
EOF
