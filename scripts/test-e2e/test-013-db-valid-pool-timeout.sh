#!/usr/bin/env bash
## Test: Database with pool and timeout settings (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=d, long=dbname, required=true, type=string, minLength=2, help=Database name; short=u, long=user, required=true, type=string, minLength=1, help=Database user; short=s, long=poolsize, type=int, default=10, help=Connection pool size; short=t, long=timeout, type=int, default=30, help=Connection timeout;'

expect_ok "$SCHEMA" -d testdb -u appuser -s 50 -t 60 <<'EOF'
SHOPTS_DBNAME=testdb
SHOPTS_USER=appuser
SHOPTS_POOLSIZE=50
SHOPTS_TIMEOUT=60
EOF
