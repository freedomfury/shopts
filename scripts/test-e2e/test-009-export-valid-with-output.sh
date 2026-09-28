#!/usr/bin/env bash
## Test: Data export with format and output (valid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=f, long=format, required=true, type=enum, enum="json,csv,yaml", help=Output format; short=o, long=output, type=string, pattern={{ RelativePath }}, help=Output file; short=z, long=compress, type=flag, help=Compress output;'

expect_ok "$SCHEMA" --format csv --output ./data/export.csv <<'EOF'
SHOPTS_FORMAT=csv
SHOPTS_OUTPUT=./data/export.csv
SHOPTS_COMPRESS=false
EOF
