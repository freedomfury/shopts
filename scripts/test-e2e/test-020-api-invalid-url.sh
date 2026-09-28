#!/usr/bin/env bash
## Test: API with invalid URL (no scheme) (invalid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=e, long=endpoint, required=true, type=string, pattern={{ URL }}, help=API endpoint; short=m, long=method, type=enum, enum="GET,POST,PUT,DELETE", default=GET, help=HTTP method;'

expect_fail 3 'invalid value for --endpoint: must be a URL with a scheme, like https://example.com' "$SCHEMA" -e api.example.com/data -m POST
