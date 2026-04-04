#!/usr/bin/env bash
## Test: API client with just endpoint (valid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=e, long=endpoint, required=true, type=string, pattern={{ URL }}, help=API endpoint; short=m, long=method, type=enum, enum="GET,POST,PUT,DELETE", default=GET, help=HTTP method;'

if "$BINARY" "$SCHEMA" -e https://api.example.com/v1/users >/dev/null 2>&1; then
    exit 0
else
    exit 1
fi
