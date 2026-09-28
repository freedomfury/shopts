#!/usr/bin/env bash
## Acceptance tests for the shopts contract (one or more checks per spec item).
## NOTE: Run this script from the project root (../scripts/test-contract.sh)
set -euo pipefail

binary=bin/shopts
if [[ ! -x "${binary}" ]]; then
  go build -o "${binary}" ./cmd/shopts
fi

# Settings from the caller's environment must not change the output.
unset "${!GO_SHOPTS_@}"

TAB=$'\t'
passed=0
failed=0
RUN_ENV=() # VAR=value settings for the next run (see envrun)

# run SCHEMA ARGS... — sets OUT (stdout, byte for byte), ERR and RC.
run() {
  local errf
  errf=$(mktemp)
  RC=0
  # The trailing x keeps $(...) from stripping the output's final newline.
  OUT=$(
    rc=0
    env "${RUN_ENV[@]}" "${binary}" "$@" 2>"${errf}" || rc=$?
    printf x
    exit "${rc}"
  ) || RC=$?
  OUT=${OUT%x}
  ERR=$(<"${errf}")
  rm -f "${errf}"
}

pass() { passed=$((passed + 1)); }
fail() {
  failed=$((failed + 1))
  printf 'FAIL: %s\n' "$1"
  printf '  rc=%s\n  stdout=%q\n  stderr=%q\n' "${RC}" "${OUT}" "${ERR}"
}

# ok NAME EXPECTED_STDOUT SCHEMA ARGS... — expects exit 0 and exactly the given
# lines on stdout, each ending in a newline.
ok() {
  local name=$1 want=$2$'\n'
  shift 2
  run "$@"
  if [[ ${RC} -eq 0 && "${OUT}" == "${want}" ]]; then pass; else fail "${name} (want stdout ${want@Q})"; fi
}

# err NAME RC STDERR_SUBSTRING SCHEMA ARGS... — expects the exit code, empty
# stdout, and stderr containing the substring.
err() {
  local name=$1 want_rc=$2 want_err=$3
  shift 3
  run "$@"
  if [[ ${RC} -eq ${want_rc} && -z "${OUT}" && "${ERR}" == *"${want_err}"* ]]; then pass; else fail "${name} (want rc ${want_rc}, stderr ~ ${want_err@Q})"; fi
}

S='long=a, type=string;'

echo "--- Schema format ---"
ok "multi-line indented entries, optional space after comma" "SHOPTS_ENV${TAB}dev" '
    short=e,long=env, type=enum,
      enum="dev,prod";
' -e dev
ok "quoted value holds , and ;" "SHOPTS_A${TAB}x" 'long=a, type=string, help="one, two; three";' --a x
err "missing terminating ;" 2 "entry is missing its terminating ';'" 'long=a, type=string'

echo "--- Schema parser: positions and quoting ---"
err "positioned unknown field" 2 'schema line 3, col 24: unknown field "patern"' '
  long=a, type=string;
  long=b, type=string, patern=x;
'
ok 'quoted regex needs no double escaping' "SHOPTS_A${TAB}123" 'long=a, type=string, pattern="^\d{2,4}$";' --a 123
ok 'quoted \" is a literal quote' "SHOPTS_A${TAB}Say \"hi\", then go" 'long=a, type=string, default="Say \"hi\", then go";'
# shellcheck disable=SC1003 # the trailing backslash is the value under test
ok 'quoted \\ is a literal backslash' "SHOPTS_A${TAB}C:\\" 'long=a, type=string, pattern="^C:\\\\";' '--a=C:\'
ok 'unquoted regex' "SHOPTS_A${TAB}42" 'long=a, type=string, pattern=^\d+$;' --a 42
ok 'Go escapes are not interpreted' "SHOPTS_A${TAB}a\\tb" 'long=a, type=string, default="a\tb";'

echo "--- Fields ---"
err "long must start with a letter" 2 "must start with a letter" 'long=1a, type=string;'
err "long has no hyphens" 2 "must start with a letter" 'long=dry-run, type=flag;'
err "short H reserved" 2 "reserved" 'long=a, short=H, type=flag;'
err "short V reserved" 2 "reserved" 'long=a, short=V, type=flag;'
err "required is true or false only" 2 "must be true or false" 'long=a, type=string, required=yes;'
err "required and default exclusive" 2 "required and default cannot both be set" 'long=a, type=string, required=true, default=x;'
err "flag cannot have a default" 2 "so it cannot have a default" 'long=cache, type=flag, default=true;'
err "flag cannot be required" 2 "cannot be required" 'long=cache, type=flag, required=true;'
ok "on-by-default behavior is a flag named for turning it off" "SHOPTS_NO_CACHE${TAB}true" 'long=no_cache, type=flag;' --no-cache
err "no --no- form is implied" 3 "unknown option --no-dry-run" 'long=dry_run, type=flag;' --no-dry-run
ok "both spellings take two entries" "SHOPTS_DRY_RUN${TAB}false
SHOPTS_NO_DRY_RUN${TAB}true" 'long=dry_run, type=flag; long=no_dry_run, type=flag;' --no-dry-run
ok "no_ has no special meaning" "SHOPTS_NO_LIMIT${TAB}5" 'long=no_limit, type=int;' --no-limit 5
err "defaults validated at parse time" 2 'default "x": must be a valid integer' 'long=a, type=int, default=x;'
ok "min/max inclusive (low)" "SHOPTS_PORT${TAB}1" 'long=port, type=int, min=1, max=65535;' --port 1
ok "min/max inclusive (high)" "SHOPTS_PORT${TAB}65535" 'long=port, type=int, min=1, max=65535;' --port 65535
err "below min" 3 "must be at least 1" 'long=port, type=int, min=1, max=65535;' --port 0
err "above max" 3 "must be at most 65535" 'long=port, type=int, min=1, max=65535;' --port 65536
err "float max" 3 "must be at most 1.5" 'long=r, type=float, max=1.5;' --r 1.6
err "min on string rejected" 2 'field "min" does not apply to type string' 'long=a, type=string, min=1;'
ok "maxLength counts code points" "SHOPTS_A${TAB}héé" 'long=a, type=string, maxLength=3;' --a héé
err "maxLength exceeded" 3 "no more than 3 characters" 'long=a, type=string, maxLength=3;' --a héée
items=()
for _ in {1..100}; do items+=(-t x); done
ok "maxItems defaults to 100 (100 ok)" "SHOPTS_T${TAB}$(printf 'x,%.0s' {1..99})x" 'long=t, short=t, type=list;' "${items[@]}"
err "maxItems defaults to 100 (101 fails)" 3 "at most 100 items" 'long=t, short=t, type=list;' "${items[@]}" -t x
err "pattern applies to list items" 3 "must match the pattern" 'long=t, short=t, type=list, pattern=[a-z]+;' -t ok -t NO

echo "--- Invocation and configuration ---"
# envrun NAME RC WANT VAR=VAL... -- SCHEMA ARGS... — like ok (RC 0, WANT is the
# stdout) or err (WANT is a stderr substring), with the settings in the
# environment of that one run.
envrun() {
  local name=$1 want_rc=$2 want=$3
  shift 3
  while [[ $1 != -- ]]; do RUN_ENV+=("$1"); shift; done
  shift
  if [[ ${want_rc} -eq 0 ]]; then ok "${name}" "${want}" "$@"; else err "${name}" "${want_rc}" "${want}" "$@"; fi
  RUN_ENV=()
}
envrun "GO_SHOPTS_PREFIX" 0 "OPT_A${TAB}x" GO_SHOPTS_PREFIX=OPT_ -- "${S}" --a x
envrun "GO_SHOPTS_PREFIX reserved" 1 "must not start with GO_SHOPTS_" GO_SHOPTS_PREFIX=GO_SHOPTS_X -- "${S}"
envrun "GO_SHOPTS_UPCASE=0 keeps names as written" 0 "SHOPTS_myOpt${TAB}x" GO_SHOPTS_UPCASE=0 -- 'long=myOpt, type=string;' --myOpt x
envrun "bad GO_SHOPTS_UPCASE" 1 "GO_SHOPTS_UPCASE" GO_SHOPTS_UPCASE=maybe -- "${S}"
envrun "GO_SHOPTS_OUT_DELIM" 0 "SHOPTS_A=x" GO_SHOPTS_OUT_DELIM== -- "${S}" --a x
envrun "GO_SHOPTS_LIST_DELIM" 0 "SHOPTS_T${TAB}a:b" GO_SHOPTS_LIST_DELIM=: -- 'long=t, short=t, type=list;' -t a -t b
envrun "delimiter may be any string" 0 "SHOPTS_AAAAx" GO_SHOPTS_OUT_DELIM=AAA -- "${S}" --a x
envrun "named delimiter {{ tab }}" 0 "SHOPTS_A${TAB}x" 'GO_SHOPTS_OUT_DELIM={{tab}}' -- "${S}" --a x
envrun "unknown named delimiter" 1 'unknown delimiter "{{ nope }}"' 'GO_SHOPTS_OUT_DELIM={{ nope }}' -- "${S}"
envrun "delimiter cannot contain a newline" 1 "must not contain a newline" "GO_SHOPTS_OUT_DELIM=a
b" -- "${S}"
# stdout with NUL bytes cannot go through $(...); translate them first.
got=$(GO_SHOPTS_OUT_DELIM='{{ null }}' "${binary}" "${S}" --a "x${TAB}y" | tr '\0' '|')
if [[ "${got}" == "SHOPTS_A|x${TAB}y" ]]; then pass; else RC=0 OUT=${got} ERR=""; fail "named delimiter {{ null }} writes a NUL byte"; fi
envrun "GO_SHOPTS_DASH=0 keeps underscores" 0 "SHOPTS_DRY_RUN${TAB}true
SHOPTS_NO_CACHE${TAB}true" GO_SHOPTS_DASH=0 -- 'long=dry_run, type=flag; long=no_cache, type=flag;' --dry_run --no_cache
envrun "bad GO_SHOPTS_DASH" 1 "GO_SHOPTS_DASH" GO_SHOPTS_DASH=maybe -- "${S}"
envrun "GO_SHOPTS_NAME in errors" 3 "deploy.sh: missing required option --a" GO_SHOPTS_NAME=deploy.sh -- 'long=a, type=string, required=true;'
envrun "GO_SHOPTS_NAME in help" 7 "Usage: deploy.sh [OPTIONS]" GO_SHOPTS_NAME=deploy.sh -- "${S}" -H

echo "--- Argument parsing ---"
for form in "--a v" "--a=v" "-x v" "-x=v"; do
  # shellcheck disable=SC2086 # split the form into words on purpose
  ok "option form ${form}" "SHOPTS_A${TAB}v" 'long=a, short=x, type=string;' ${form}
done
ok "underscores become dashes by default" "SHOPTS_DRY_RUN${TAB}true
SHOPTS_MAX_DEPTH${TAB}3" 'long=dry_run, type=flag; long=max_depth, type=int;' --dry-run --max-depth=3
err "only one spelling is accepted" 3 "unknown option --dry_run" 'long=dry_run, type=flag;' --dry_run
ok "value that looks like an option is a value" "SHOPTS_A${TAB}--b
SHOPTS_B${TAB}false" 'long=a, type=string; long=b, type=flag;' --a --b
err "flags take no value" 3 "--b does not take a value" 'long=b, type=flag;' --b=true
err "repeated non-list option" 3 "--a is given more than once" "${S}" --a x --a y
err "repeated flag" 3 "--b is given more than once" 'long=b, type=flag;' --b --b
ok "list collects every occurrence" "SHOPTS_T${TAB}a,b,c" 'long=t, short=t, type=list;' -t a --t b -t=c
err "-H anywhere" 7 "Options:" "${S}" --a x --bogus -H
err "-V anywhere" 7 "shopts " "${S}" --a x -V
err "-V after other args" 7 "shopts " "${S}" --bogus --version
ok "-h and -v are free for the schema" "SHOPTS_HOST${TAB}db
SHOPTS_VERBOSE${TAB}true" 'long=host, short=h, type=string; long=verbose, short=v, type=flag;' -h db -v
err "short bundles are not supported" 3 "short options cannot be combined: -ab" 'long=a, short=a, type=flag; long=b, short=b, type=flag;' -ab

err "bare words are rejected" 3 'unrecognized bare word "web"' "${S}" --a x web
err "a lone - is a bare word" 3 'unrecognized bare word "-"' "${S}" -
err "everything after -- is rejected" 3 'unrecognized bare word "-x" after --' "${S}" --a x -- -x
err "positional= no longer exists" 2 'unknown field "positional"' 'long=a, type=string, positional=1;'

echo "--- Output contract ---"
ok "schema order, emission rules" "SHOPTS_C${TAB}3
SHOPTS_B${TAB}false
SHOPTS_A${TAB}x" 'long=c, type=int, default=3; long=b, type=flag; long=n, type=string; long=a, type=string;' --a x
ok "bool normalized" "SHOPTS_B${TAB}true" 'long=b, type=bool;' --b T
ok "bool normalized (1)" "SHOPTS_B${TAB}true" 'long=b, type=bool;' --b 1
ok "bool normalized (FALSE)" "SHOPTS_B${TAB}false" 'long=b, type=bool;' --b FALSE
err "bool rejects yes" 3 "must be true or false" 'long=b, type=bool;' --b yes
ok "int normalized (007)" "SHOPTS_N${TAB}7" 'long=n, type=int;' --n 007
ok "int normalized (+5)" "SHOPTS_N${TAB}5" 'long=n, type=int;' --n +5
ok "flag given" "SHOPTS_F${TAB}true" 'long=f, type=flag;' --f
ok "float as given" "SHOPTS_R${TAB}2.50" 'long=r, type=float;' --r 2.50
err "newline rejected" 3 "must not contain a newline" "${S}" --a "x
y"
ok "value may contain the output delimiter" "SHOPTS_A${TAB}x${TAB}y" "${S}" --a "x${TAB}y"
err "no stdout on error" 3 "missing required" 'long=a, type=string; long=b, type=string, required=true;' --a x

echo "--- Exit codes ---"
envrun "1: bad setting" 1 "GO_SHOPTS_UPCASE" GO_SHOPTS_UPCASE=maybe -- "${S}"
err "2: schema error" 2 "schema line 1" 'long=a, type=nope;'
err "3: argument error" 3 "unknown option --b" "${S}" --b
err "7: help" 7 "Usage:" "${S}" --help
err "7: version" 7 "shopts " "${S}" -V
err "version ignores a broken schema" 7 "shopts " 'long=a, type=nope;' --version
envrun "version ignores bad settings" 7 "shopts " GO_SHOPTS_PREFIX=1BAD -- "${S}" -V
envrun "help ignores output settings" 7 "Usage:" GO_SHOPTS_PREFIX=1BAD -- "${S}" -H
err "help needs a valid schema" 2 "schema line" 'long=a, type=nope;' --help

echo "--- Help ---"
run 'short=e, long=env, type=enum, enum="dev,prod", required=true, help=Target;
short=v, long=verbose, type=flag, help=Verbose output;' --help
if [[ ${RC} -eq 7 && -z "${OUT}" && "${ERR}" == "Usage: [OPTIONS]"$'\n'* &&
  "${ERR}" == *"  -e, --env <value>   Target; enum; required; allowed: dev, prod"* &&
  "${ERR}" == *"  -v, --verbose       Verbose output; flag"* &&
  "${ERR}" == *"  -H, --help          Show this help"* &&
  "${ERR}" != *"GO_SHOPTS_"* && "${ERR}" != *"Usage: shopts"* ]]; then
  pass
else
  fail "help on stderr, exit 7, describes the caller's script only"
fi

echo "--- Pattern validators ---"
err "inline regex matches the whole value" 3 "must match the pattern [0-9]+" 'long=a, type=string, pattern=[0-9]+;' --a abc1
ok "define" "SHOPTS_TICKET${TAB}ABC-123" '
define=Ticket, pattern=^[A-Z]+-[0-9]+$, failure=must look like ABC-123;
long=ticket, type=string, pattern={{ Ticket }};' --ticket ABC-123
err "define failure message" 3 "must look like ABC-123" '
define=Ticket, pattern=^[A-Z]+-[0-9]+$, failure=must look like ABC-123;
long=ticket, type=string, pattern={{ Ticket }};' --ticket abc
err "define cannot reuse a built-in name" 2 "is a built-in validator" 'define=SemVer, pattern=x; long=a, type=flag;'
err "option failure= wins" 3 "custom" 'long=a, type=string, pattern={{ PortNumber }}, failure=custom;' --a x
err "built-in failure message" 3 "must be a port number from 1 to 65535" 'long=a, type=string, pattern={{ PortNumber }};' --a x
ok "RelativePath accepts config/file.yaml" "SHOPTS_A${TAB}config/file.yaml" 'long=a, type=string, pattern={{ RelativePath }};' --a config/file.yaml
err "PortNumber rejects +80" 3 "port number" 'long=a, type=string, pattern={{ PortNumber }};' --a +80
err "GitRef rejects feature..x" 3 "git ref" 'long=a, type=string, pattern={{ GitRef }};' --a feature..x
ok "IPv6Address accepts IPv4-mapped" "SHOPTS_A${TAB}::ffff:192.0.2.1" 'long=a, type=string, pattern={{ IPv6Address }};' --a ::ffff:192.0.2.1

echo "--- Code review fixes ---"
err "regex cannot escape whole-value matching" 2 "unexpected )" 'long=a, type=string, pattern=[0-9]+)|(.*;' --a 'hello world'
ok "regex with \\Q and no \\E" "SHOPTS_A${TAB}1.2.3" 'long=a, type=string, pattern=\Q1.2.3;' --a 1.2.3
err "malformed validator name is a schema error" 2 'unknown validator "Port_Number"' 'long=a, type=string, pattern={{ Port_Number }};' --a 80
err "bare value cannot start on the next line" 2 'value of "help" is missing on this line' $'long=a, type=string, help=\n  pattern=[a-z]+;'
ok "quoted value may start on the next line" "SHOPTS_A${TAB}x" $'long=a, type=string, default=\n  "x";'
ok "bare value may contain a quoted comma" "SHOPTS_A${TAB}v" 'long=a, type=string, help=Use "x, y" form;' --a v
err "long names differing only in case" 2 "differ only in case" 'long=Env, type=string; long=env, type=string, default=prod;' --Env dev
ok "quoted list items keep their spaces" "SHOPTS_L${TAB}  padded  ,x" 'long=l, type=list, default="\"  padded  \",x";'
err "unclosed quoted enum item" 2 "quoted item is never closed" 'long=m, type=enum, enum="\"a,b";'
err "define field given twice" 2 'field "pattern" is given twice' 'define=T, pattern=[a-z]+, pattern=[0-9]+; long=a, type=string, pattern={{ T }};' --a 123
err "minItems above the default maxItems" 2 "minItems 150 is greater than maxItems 100" 'long=t, type=list, minItems=150;'
err "required list with maxItems=0" 2 "is greater than maxItems 0" 'long=t, type=list, required=true, maxItems=0;'
err "required list needs a non-empty value" 3 "--t requires a non-empty value" 'long=t, type=list, required=true;' --t=
err "RelativePath rejects option-like values" 3 "not starting with / or -" 'long=p, type=string, pattern={{ RelativePath }};' --p -rf
err "-- as an option's value does not end options" 7 "Options:" 'long=a, type=string;' --a -- -H
ok "README validator example parses" "SHOPTS_RELEASE${TAB}1.0.0
SHOPTS_HOST${TAB}10.0.0.1" '
long=email,   type=string, pattern={{ EmailAddress }};
long=release, type=string, pattern={{ SemVer }}, default=1.0.0;
long=host,    type=string, pattern={{ IPv4Address }}, required=true;
' --host 10.0.0.1

echo "--- Schema and settings edge cases ---"
run
if [[ ${RC} -eq 2 && -z "${OUT}" && "${ERR}" == *"missing schema"* ]]; then pass; else fail "no schema argument is a schema error"; fi
err "tool-level --help without a schema" 7 "usage: shopts SCHEMA [ARGS...]" --help
err "long=help is reserved" 2 '"help" is reserved' 'long=help, type=flag;'
err "long=version is reserved" 2 '"version" is reserved' 'long=version, type=flag;'
err "a field given twice in an option entry" 2 'field "type" is given twice' 'long=a, type=string, type=int;'
envrun "GO_SHOPTS_PREFIX must be a shell prefix" 1 "not a valid shell variable prefix" GO_SHOPTS_PREFIX=1BAD -- "${S}"
envrun "GO_SHOPTS_LIST_DELIM cannot contain a newline" 1 "must not contain a newline" "GO_SHOPTS_LIST_DELIM=a
b" -- "${S}"

echo "--- Code review leftovers ---"
err "unknown field has no suggestion" 2 'unknown field "patern"' 'long=a, type=string, patern=x;'
run 'long=a, type=string, patern=x;'
if [[ "${ERR}" != *"did you mean"* ]]; then pass; else fail "unknown field has no suggestion (no did-you-mean)"; fi
err "float rejects Go-only syntax" 3 "must be a valid number" 'long=r, type=float;' --r 1_000
ok "float accepts an exponent" "SHOPTS_R${TAB}1e3" 'long=r, type=float;' --r 1e3
err "help must be one line" 2 "help: must be a single line" $'long=a, type=flag, help="one\ntwo";'
err "missing value reported once" 3 "--a requires a value" 'long=a, type=string, required=true;' --a
run 'long=a, type=string, required=true;' --a
if [[ "${ERR}" != *"missing required"* ]]; then pass; else fail "missing value reported once (no second error)"; fi
err "schema must be UTF-8" 2 "schema line 1, col 30: schema is not valid UTF-8" "$(printf 'long=a, type=string, help=Caf\xe9;')"
err "go_shopts names are reserved" 2 "names starting with go_shopts belong to shopts" 'long=go_shopts_args, type=string;'
err "SemVer rejects a leading-zero pre-release" 3 "must be a semantic version" 'long=v, type=string, pattern={{ SemVer }};' --v 1.0.0-01
ok "SemVer accepts a proper pre-release" "SHOPTS_V${TAB}1.0.0-rc.1+build.5" 'long=v, type=string, pattern={{ SemVer }};' --v 1.0.0-rc.1+build.5
err "option names are ASCII" 3 "unknown option -é" "${S}" -é
err "count fields take plain digits" 2 'got "01"' 'long=t, type=list, maxItems=01;'

echo "--- Caller idiom (README read loop, bash 4.4+) ---"
caller() {
  local SHOPTS_A=""
  while IFS=$'\t' read -r key val; do
    printf -v "${key}" '%s' "${val}"
  done < <("${binary}" 'long=a, type=string, required=true;' "$@" 2>/dev/null)
  local rc=0
  wait $! || rc=$?
  [[ ${rc} -eq 7 ]] && { echo "stop"; return 0; }
  [[ ${rc} -ne 0 ]] && { echo "exit ${rc}"; return 0; }
  echo "a=${SHOPTS_A}"
}
if [[ "$(caller --a hi)" == "a=hi" && "$(caller --help)" == "stop" && "$(caller)" == "exit 3" ]]; then
  pass
else
  fail "caller idiom propagates 0, 7 and 3"
fi

echo "--- Caller idiom with {{ null }} (README) ---"
nullcaller() {
  local key val SHOPTS_A=""
  while IFS= read -r -d '' key && IFS= read -r val; do
    printf -v "${key}" '%s' "${val}"
  done < <(GO_SHOPTS_OUT_DELIM='{{ null }}' "${binary}" 'long=a, type=string;' "$@")
  printf '%s' "${SHOPTS_A}"
}
if [[ "$(nullcaller --a "${TAB} padded  ${TAB}")" == "${TAB} padded  ${TAB}" ]]; then
  pass
else
  fail "{{ null }} read loop keeps a value exactly"
fi

echo
printf 'contract checks: %d passed, %d failed\n' "${passed}" "${failed}"
[[ ${failed} -eq 0 ]]
