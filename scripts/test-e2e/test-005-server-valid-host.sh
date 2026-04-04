#!/usr/bin/env bash
## Test: Server config with just host (valid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=h, long=host, required=true, type=string, pattern={{ IPv4Address }}, help=Server host; short=p, long=port, type=int, default=8080, help=Server port;'

if "$BINARY" "$SCHEMA" -h 192.168.1.1 >/dev/null 2>&1; then
    exit 0
else
    exit 1
fi
