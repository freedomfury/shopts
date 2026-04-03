#!/usr/bin/env bash
## NOTE: Run this script from the project root (../scripts/test.sh)
set -euo pipefail

SCHEMA='
short=u;long=username;required=true;type=string;help=Username for login;minLength=3;
short=p;long=pass;required=true;type=string;help=Password for login;minLength=6;
short=v;long=verbose;required=false;type=flag;help=Enable verbose output;
short=m;long=mode;required=false;type=enum;enum=dev,prod;default=dev;help=Execution mode;
short=c;long=config;required=false;type=string;help=Path to configuration file;default=/etc/app/config.yaml;
'

binary=bin/shopts
if [[ ! -x "${binary}" ]]; then
  go build -o "${binary}" ./cmd/shopts
fi

export GO_SHOPTS_UPCASE=1
while IFS= read -r -d $'\0' k && IFS= read -r v; do
  printf -v "${k}" '%s' "${v}"
  declare -xr "${k#GO_SHOPTS_}"="${v}"
done < <("${binary}" "${SCHEMA}" -u alice -p s3cret -v)

printf 'USERNAME=%s\n' "${USERNAME}"
printf 'PASS=%s\n' "${PASS}"
printf 'VERBOSE=%s\n' "${VERBOSE}"
printf 'MODE=%s\n' "${MODE}"
printf 'CONFIG=%s\n' "${CONFIG}"
