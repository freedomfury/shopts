#!/usr/bin/env bash
## Test: Data export with compress flag (valid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=f, long=format, required=true, type=enum, enum="json,csv,yaml", help=Output format; short=z, long=compress, type=flag, help=Compress output; short=m, long=maxrecords, type=int, default=1000, help=Max records;'

if "$BINARY" "$SCHEMA" -f yaml -z -m 5000 >/dev/null 2>&1; then
    exit 0
else
    exit 1
fi
