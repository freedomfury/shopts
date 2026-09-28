#!/usr/bin/env bash
## Test: Database with required fields only (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=d, long=dbname, required=true, type=string, minLength=2, help=Database name; short=u, long=user, required=true, type=string, minLength=1, help=Database user;'

expect_ok "$SCHEMA" -d mydb -u dbuser <<'EOF'
SHOPTS_DBNAME=mydb
SHOPTS_USER=dbuser
EOF
