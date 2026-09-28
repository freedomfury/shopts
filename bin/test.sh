#!/usr/bin/env bash
# Example caller (bash 4.4+). Try: bin/test.sh -u alice, bin/test.sh -H
SCHEMA='
short=u, long=user, required=true, type=string, minLength=3, help=Username;
short=p, long=port, type=int, min=1, max=65535, default=8080, help=Port number;
short=v, long=verbose, type=flag, help=Enable verbose output;
'

while IFS=$'\t' read -r key val; do
  printf -v "$key" '%s' "$val"
done < <(GO_SHOPTS_NAME="$0" ./bin/shopts "$SCHEMA" "$@")
rc=0
wait $! || rc=$?
[[ $rc -eq 7 ]] && exit 0
[[ $rc -ne 0 ]] && exit "$rc"

printf "User: %s\n" "$SHOPTS_USER"
printf "Port: %d\n" "$SHOPTS_PORT"
if [ "$SHOPTS_VERBOSE" = "true" ]; then
  echo "Verbose mode is enabled."
fi
