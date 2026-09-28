#!/usr/bin/env bash
## Test: Authentication with mixed options (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=u, long=username, required=true, type=string, minLength=3, help=Username; short=p, long=pass, required=true, type=string, minLength=6, help=Password;'

expect_ok "$SCHEMA" -u bob --pass secretpass <<'EOF'
SHOPTS_USERNAME=bob
SHOPTS_PASS=secretpass
EOF
