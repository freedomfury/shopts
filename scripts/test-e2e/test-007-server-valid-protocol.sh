#!/usr/bin/env bash
## Test: Server config with protocol (valid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=h, long=host, required=true, type=string, pattern={{ IPv4Address }}, help=Server host; short=p, long=port, type=int, default=8080, help=Server port; short=c, long=protocol, type=enum, enum="http,https", default=http, help=Protocol; short=s, long=sslverify, type=bool, default=true, help=Verify SSL;'

if "$BINARY" "$SCHEMA" -h 172.16.0.1 -p 8443 -c https -s false >/dev/null 2>&1; then
    exit 0
else
    exit 1
fi
