#!/usr/bin/env bash
## Test: Database with required fields only (valid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=d, long=dbname, required=true, type=string, minLength=2, help=Database name; short=u, long=user, required=true, type=string, minLength=1, help=Database user;'

if "$BINARY" "$SCHEMA" -d mydb -u dbuser >/dev/null 2>&1; then
    exit 0
else
    exit 1
fi
