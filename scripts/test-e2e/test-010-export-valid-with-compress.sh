#!/usr/bin/env bash
## Test: Data export with compress flag (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=f, long=format, required=true, type=enum, enum="json,csv,yaml", help=Output format; short=z, long=compress, type=flag, help=Compress output; short=m, long=maxrecords, type=int, default=1000, help=Max records;'

expect_ok "$SCHEMA" -f yaml -z -m 5000 <<'EOF'
SHOPTS_FORMAT=yaml
SHOPTS_COMPRESS=true
SHOPTS_MAXRECORDS=5000
EOF
