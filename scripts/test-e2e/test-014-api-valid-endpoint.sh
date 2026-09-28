#!/usr/bin/env bash
## Test: API client with just endpoint (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=e, long=endpoint, required=true, type=string, pattern={{ URL }}, help=API endpoint; short=m, long=method, type=enum, enum="GET,POST,PUT,DELETE", default=GET, help=HTTP method;'

expect_ok "$SCHEMA" -e https://api.example.com/v1/users <<'EOF'
SHOPTS_ENDPOINT=https://api.example.com/v1/users
SHOPTS_METHOD=GET
EOF
