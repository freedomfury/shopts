#!/usr/bin/env bash
## Test: Server config with host and port (valid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=h, long=host, required=true, type=string, pattern={{ IPv4Address }}, help=Server host; short=p, long=port, type=int, default=8080, help=Server port; short=c, long=protocol, type=enum, enum="http,https", default=http, help=Protocol;'

if "$BINARY" "$SCHEMA" --host 10.0.0.1 --port 9000 >/dev/null 2>&1; then
    exit 0
else
    exit 1
fi
