#!/usr/bin/env bash
## Test: Export with invalid format enum (invalid)
set -euo pipefail

# shellcheck source=scripts/test-e2e/lib.sh
source "$(dirname "$0")/lib.sh"

SCHEMA='short=f, long=format, required=true, type=enum, enum="json,csv,yaml", help=Output format; short=o, long=output, type=string, pattern={{ RelativePath }}, help=Output file;'

expect_fail 3 'invalid value for --format: must be one of: json, csv, yaml' "$SCHEMA" -f xml -o ./export.xml
