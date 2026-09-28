#!/usr/bin/env bash
## Test: Authentication with bool option (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=u, long=username, required=true, type=string, minLength=3, help=Username; short=p, long=pass, required=true, type=string, minLength=6, help=Password; short=r, long=remember, type=bool, default=false, help=Remember me;'

expect_ok "$SCHEMA" -u charlie -p pass1234 -r true <<'EOF'
SHOPTS_USERNAME=charlie
SHOPTS_PASS=pass1234
SHOPTS_REMEMBER=true
EOF
