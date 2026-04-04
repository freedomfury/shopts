#!/usr/bin/env bash
## Test: Database with host override (valid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=d, long=dbname, required=true, type=string, minLength=2, help=Database name; short=u, long=user, required=true, type=string, minLength=1, help=Database user; short=h, long=host, type=string, default=localhost, help=Database host; short=p, long=port, type=int, default=5432, help=Database port;'

if "$BINARY" "$SCHEMA" --dbname proddb --user admin -h db.example.com -p 5433 >/dev/null 2>&1; then
    exit 0
else
    exit 1
fi
