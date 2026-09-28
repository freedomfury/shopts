#!/usr/bin/env bash
## Test: Authentication with short options (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=u, long=username, required=true, type=string, minLength=3, help=Username; short=p, long=pass, required=true, type=string, minLength=6, help=Password; short=v, long=verbose, type=flag, help=Verbose mode; short=r, long=remember, type=bool, default=false, help=Remember me;'

expect_ok "$SCHEMA" -u alice -p password123 -v <<'EOF'
SHOPTS_USERNAME=alice
SHOPTS_PASS=password123
SHOPTS_VERBOSE=true
SHOPTS_REMEMBER=false
EOF
