#!/usr/bin/env bash
## Test: Database with host override (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=d, long=dbname, required=true, type=string, minLength=2, help=Database name; short=u, long=user, required=true, type=string, minLength=1, help=Database user; short=h, long=host, type=string, default=localhost, help=Database host; short=p, long=port, type=int, default=5432, help=Database port;'

expect_ok "$SCHEMA" --dbname proddb --user admin -h db.example.com -p 5433 <<'EOF'
SHOPTS_DBNAME=proddb
SHOPTS_USER=admin
SHOPTS_HOST=db.example.com
SHOPTS_PORT=5433
EOF
