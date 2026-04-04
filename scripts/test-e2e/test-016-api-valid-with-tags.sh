#!/usr/bin/env bash
## Test: API client with tags list (valid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=e, long=endpoint, required=true, type=string, pattern={{ URL }}, help=API endpoint; short=t, long=tags, type=list, minItems=1, maxItems=5, help=Request tags;'

if "$BINARY" "$SCHEMA" -e https://localhost:8080/api -t production -t critical >/dev/null 2>&1; then
    exit 0
else
    exit 1
fi
