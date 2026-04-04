#!/usr/bin/env bash
## Test: Authentication with long options (valid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=u, long=username, required=true, type=string, minLength=3, help=Username; short=p, long=pass, required=true, type=string, minLength=6, help=Password; short=v, long=verbose, type=flag, help=Verbose mode;'

if "$BINARY" "$SCHEMA" --username alice --pass password123 --verbose >/dev/null 2>&1; then
    exit 0
else
    exit 1
fi
