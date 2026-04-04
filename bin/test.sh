#!/usr/bin/env bash
SCHEMA='
short=u, long=user, required=true, type=string, minLength=3, help=Username;
short=p, long=port, type=int, default=8080, help=Port number;
short=v, long=verbose, type=flag, help=Enable verbose output;
'

while IFS=$'\t' read -r key val; do
  printf -v "$key" '%s' "$val"
done < <(./bin/shopts "$SCHEMA" "$@")
wait $! || exit $?

printf "User: %s\n" "$SHOPTS_USER"
printf "Port: %d\n" "$SHOPTS_PORT"
if [ "$SHOPTS_VERBOSE" = "true" ]; then
  echo "Verbose mode is enabled."
fi
