#!/usr/bin/env bash
## Test: Authentication with long options (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=u, long=username, required=true, type=string, minLength=3, help=Username; short=p, long=pass, required=true, type=string, minLength=6, help=Password; short=v, long=verbose, type=flag, help=Verbose mode;'

expect_ok "$SCHEMA" --username alice --pass password123 --verbose <<'EOF'
SHOPTS_USERNAME=alice
SHOPTS_PASS=password123
SHOPTS_VERBOSE=true
EOF
