#!/usr/bin/env bash
## Test: Database with pool and timeout settings (valid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=d, long=dbname, required=true, type=string, minLength=2, help=Database name; short=u, long=user, required=true, type=string, minLength=1, help=Database user; short=s, long=poolsize, type=int, default=10, help=Connection pool size; short=t, long=timeout, type=int, default=30, help=Connection timeout;'

if "$BINARY" "$SCHEMA" -d testdb -u appuser -s 50 -t 60 >/dev/null 2>&1; then
    exit 0
else
    exit 1
fi
