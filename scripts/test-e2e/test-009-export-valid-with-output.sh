#!/usr/bin/env bash
## Test: Data export with format and output (valid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=f, long=format, required=true, type=enum, enum="json,csv,yaml", help=Output format; short=o, long=output, type=string, pattern={{ RelativePath }}, help=Output file; short=z, long=compress, type=flag, help=Compress output;'

if "$BINARY" "$SCHEMA" --format csv --output ./data/export.csv >/dev/null 2>&1; then
    exit 0
else
    exit 1
fi
