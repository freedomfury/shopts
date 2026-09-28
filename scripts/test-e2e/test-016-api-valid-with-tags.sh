#!/usr/bin/env bash
## Test: API client with tags list (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=e, long=endpoint, required=true, type=string, pattern={{ URL }}, help=API endpoint; short=t, long=tags, type=list, minItems=1, maxItems=5, help=Request tags;'

expect_ok "$SCHEMA" -e https://localhost:8080/api -t production -t critical <<'EOF'
SHOPTS_ENDPOINT=https://localhost:8080/api
SHOPTS_TAGS=production,critical
EOF
