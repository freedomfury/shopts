#!/usr/bin/env bash
## NOTE: Run this script from the project root (../scripts/test-negative.sh)
set -euo pipefail

# Settings from the caller's environment must not change the output.
unset "${!GO_SHOPTS_@}"

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

rc=0
GO_SHOPTS_NAME=login.sh "${binary}" "${SCHEMA}" --help >/dev/null 2>"${help_out}" || rc=$?
if [[ ${rc} -ne 7 ]]; then
  echo "expected --help to exit 7, got ${rc}" >&2
  exit 1
fi
cat >"${help_expected}" <<'EOF'
Usage: login.sh [OPTIONS]

Options:
  -u, --username <value>   Username for login; string; required; minimum length: 3
                           The username to authenticate with the system.
  -p, --pass <value>       Password for login; string; required; minimum length: 6
  -v, --verbose            Enable verbose output; flag
  -m, --mode <value>       Execution mode; enum; default: dev; allowed: dev, prod
  -H, --help               Show this help
EOF

diff -u "${help_expected}" "${help_out}"

rc=0
GO_SHOPTS_NAME=login.sh "${binary}" "${SCHEMA}" -u al -p x >/dev/null 2>"${err_out}" || rc=$?
if [[ ${rc} -ne 3 ]]; then
  echo "expected validation failure to exit 3, got ${rc}" >&2
  exit 1
fi

cat >"${err_expected}" <<'EOF'
login.sh: invalid value for --username: must be at least 3 characters long
login.sh: invalid value for --pass: must be at least 6 characters long
Usage: login.sh [OPTIONS]
Try 'login.sh --help' for more information.
EOF

diff -u "${err_expected}" "${err_out}"
printf 'negative-path checks passed\n'
