#!/usr/bin/env bash
## Test: Authentication with bool option (valid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=u, long=username, required=true, type=string, minLength=3, help=Username; short=p, long=pass, required=true, type=string, minLength=6, help=Password; short=r, long=remember, type=bool, default=false, help=Remember me;'

if "$BINARY" "$SCHEMA" -u charlie -p pass1234 -r true >/dev/null 2>&1; then
    exit 0
else
    exit 1
fi
