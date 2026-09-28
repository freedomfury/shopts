#!/usr/bin/env bash
## Test: Authentication with username too short (invalid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=u, long=username, required=true, type=string, minLength=3, help=Username; short=p, long=pass, required=true, type=string, minLength=6, help=Password;'

expect_fail 3 'invalid value for --username: must be at least 3 characters long' "$SCHEMA" -u ab -p password123
