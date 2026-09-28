#!/usr/bin/env bash
## Test: API client with endpoint and method (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=e, long=endpoint, required=true, type=string, pattern={{ URL }}, help=API endpoint; short=m, long=method, type=enum, enum="GET,POST,PUT,DELETE", default=GET, help=HTTP method; short=r, long=retries, type=int, default=3, help=Retry count;'

expect_ok "$SCHEMA" --endpoint https://api.service.io/data --method POST --retries 5 <<'EOF'
SHOPTS_ENDPOINT=https://api.service.io/data
SHOPTS_METHOD=POST
SHOPTS_RETRIES=5
EOF
