#!/usr/bin/env bash
## NOTE: Run this script from the project root (../scripts/test-negative.sh)
set -euo pipefail

SCHEMA='
short=u, long=username, required=true, type=string, help=Username for login, description=The username to authenticate with the system., minLength=3;
short=p, long=pass, required=true, type=string, help=Password for login, minLength=6;
short=v, long=verbose, required=false, type=flag, help=Enable verbose output;
short=m, long=mode, required=false, type=enum, enum="dev,prod", default=dev, help=Execution mode;
'

help_out=$(mktemp)
help_expected=$(mktemp)
err_out=$(mktemp)
err_expected=$(mktemp)
cleanup() {
  rm -f "${help_out}" "${help_expected}" "${err_out}" "${err_expected}"
}
trap cleanup EXIT

binary=bin/shopts
if [[ ! -x "${binary}" ]]; then
  go build -o "${binary}" ./cmd/shopts
fi

"${binary}" "${SCHEMA}" --help 2>"${help_out}"
cat >"${help_expected}" <<'EOF'
Usage: shopts SCHEMA [OPTIONS]

Options:
  -u, --username <value>   Username for login; string; required; minimum length: 3
                           The username to authenticate with the system.
  -p, --pass <value>       Password for login; string; required; minimum length: 6
  -v, --verbose            Enable verbose output; flag (boolean switch)
  -m, --mode <value>       Execution mode; enum; default: dev; allowed: dev, prod
  -H, --help               Show schema-derived usage and exit
  -V, --version            Print version and exit

Environment variables:
  GO_SHOPTS_UPCASE=1           Output variable names in uppercase
  GO_SHOPTS_LIST_DELIM=,       Delimiter for list-type options (default: ',')
  GO_SHOPTS_OUT_DELIM=\t    Field delimiter between key and value in output (default: tab)
  GO_SHOPTS_PREFIX=X_          Override output variable prefix (default: 'SHOPTS_')

Type notes:
  int, float, bool: parsed and validated as native Go types
  list: option may be repeated, values joined by delimiter
EOF

diff -u "${help_expected}" "${help_out}"

if "${binary}" "${SCHEMA}" -u al -p x >/dev/null 2>"${err_out}"; then
  echo "expected validation failure" >&2
  exit 1
fi

cat >"${err_expected}" <<'EOF'
Usage: shopts SCHEMA [OPTIONS]

Options:
  -u, --username <value>   Username for login; string; required; minimum length: 3
                           The username to authenticate with the system.
  -p, --pass <value>       Password for login; string; required; minimum length: 6
  -v, --verbose            Enable verbose output; flag (boolean switch)
  -m, --mode <value>       Execution mode; enum; default: dev; allowed: dev, prod
  -H, --help               Show schema-derived usage and exit
  -V, --version            Print version and exit

Environment variables:
  GO_SHOPTS_UPCASE=1           Output variable names in uppercase
  GO_SHOPTS_LIST_DELIM=,       Delimiter for list-type options (default: ',')
  GO_SHOPTS_OUT_DELIM=\t    Field delimiter between key and value in output (default: tab)
  GO_SHOPTS_PREFIX=X_          Override output variable prefix (default: 'SHOPTS_')

Type notes:
  int, float, bool: parsed and validated as native Go types
  list: option may be repeated, values joined by delimiter

option "--username" invalid: must be at least 3 characters long; option "--pass" invalid: must be at least 6 characters long
EOF

diff -u "${err_expected}" "${err_out}"
printf 'negative-path checks passed\n'
