#!/usr/bin/env bash
## Test: API client with endpoint and method (valid)
set -euo pipefail

BINARY=${1:-bin/shopts}
SCHEMA='short=e, long=endpoint, required=true, type=string, pattern={{ URL }}, help=API endpoint; short=m, long=method, type=enum, enum="GET,POST,PUT,DELETE", default=GET, help=HTTP method; short=r, long=retries, type=int, default=3, help=Retry count;'

if "$BINARY" "$SCHEMA" --endpoint https://api.service.io/data --method POST --retries 5 >/dev/null 2>&1; then
    exit 0
else
    exit 1
fi
