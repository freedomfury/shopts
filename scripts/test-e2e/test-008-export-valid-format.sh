#!/usr/bin/env bash
## Test: Data export with just format (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=f, long=format, required=true, type=enum, enum="json,csv,yaml", help=Output format; short=o, long=output, type=string, pattern={{ RelativePath }}, help=Output file;'

expect_ok "$SCHEMA" -f json <<'EOF'
SHOPTS_FORMAT=json
EOF
