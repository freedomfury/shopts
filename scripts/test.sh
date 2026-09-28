#!/usr/bin/env bash
## Basic success path: parse with the README read loop and check every value.
## NOTE: Run this script from the project root (../scripts/test.sh)
set -euo pipefail

# Settings from the caller's environment must not change the output.
unset "${!GO_SHOPTS_@}"

SCHEMA='
short=u, long=username, required=true, type=string, help=Username for login, minLength=3;
short=p, long=pass, required=true, type=string, help=Password for login, minLength=6;
short=v, long=verbose, required=false, type=flag, help=Enable verbose output;
short=m, long=mode, required=false, type=enum, enum="dev,prod", default=dev, help=Execution mode;
short=c, long=config, required=false, type=string, help=Path to configuration file, default=/etc/app/config.yaml;
'

binary=bin/shopts
if [[ ! -x "${binary}" ]]; then
  go build -o "${binary}" ./cmd/shopts
fi

while IFS=$'\t' read -r k v; do
  printf -v "${k}" '%s' "${v}"
done < <("${binary}" "${SCHEMA}" -u alice -p s3cret -v)
rc=0
wait $! || rc=$?
if [[ ${rc} -ne 0 ]]; then
  echo "FAIL: shopts exited ${rc}" >&2
  exit 1
fi

check() {
  if [[ "${!1-}" != "$2" ]]; then
    printf 'FAIL: %s=%q, want %q\n' "$1" "${!1-}" "$2" >&2
    exit 1
  fi
  printf 'PASS: %s=%s\n' "$1" "$2"
}

check SHOPTS_USERNAME alice
check SHOPTS_PASS s3cret
check SHOPTS_VERBOSE true
check SHOPTS_MODE dev
check SHOPTS_CONFIG /etc/app/config.yaml
