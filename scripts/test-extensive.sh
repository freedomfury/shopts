#!/usr/bin/env bash
## NOTE: Run this script from the project root (../scripts/test-extensive.sh)
set -euo pipefail

# Extensive test for shopts: covers all supported types, variables, and edge cases

SCHEMA='
short=s;long=stringval;required=true;type=string;help=A required string value;
short=i;long=intval;required=false;type=int;help=Optional integer value;default=42;
short=f;long=floatval;required=false;type=float;help=Optional float value;default=3.14;
short=b;long=boolval;required=false;type=bool;help=Optional boolean value;default=false;
short=e;long=enumval;required=false;type=enum;enum=red,green,blue;default=green;help=Enum value;
short=l;long=listval;required=false;type=list;help=Optional list value;
short=F;long=flagval;required=false;type=flag;help=Optional flag;
short=d;long=defval;required=false;type=string;help=Has a default;default=defaultval;
'

export GO_SHOPTS_UPCASE=1

binary=bin/shopts
if [[ ! -x "${binary}" ]]; then
  go build -o "${binary}" ./cmd/shopts
fi

# Test: all options provided
while IFS= read -r -d $'\0' k && IFS= read -r v; do
  printf -v "${k}" '%s' "${v}"
  declare -x "${k#GO_SHOPTS_}"="${v}"
done < <("${binary}" "${SCHEMA}" \
  -s "hello" -i 99 -f 2.71 -b true -e blue -l a,b,c -F -d "customdef")

echo "--- All options provided ---"
printf 'STRINGVAL=%s\n' "${STRINGVAL}"
printf 'INTVAL=%s\n' "${INTVAL}"
printf 'FLOATVAL=%s\n' "${FLOATVAL}"
printf 'BOOLVAL=%s\n' "${BOOLVAL}"
printf 'ENUMVAL=%s\n' "${ENUMVAL}"
printf 'LISTVAL=%s\n' "${LISTVAL}"
printf 'FLAGVAL=%s\n' "${FLAGVAL}"
printf 'DEFVAL=%s\n' "${DEFVAL}"

echo "--- Defaults and missing values ---"
# Test: only required and some optional
while IFS= read -r -d $'\0' k && IFS= read -r v; do
  printf -v "${k}" '%s' "${v}"
  declare -x "${k#GO_SHOPTS_}"="${v}"
done < <("${binary}" "${SCHEMA}" -s "world" -F)
printf 'STRINGVAL=%s\n' "${STRINGVAL}"
printf 'INTVAL=%s\n' "${INTVAL}"
printf 'FLOATVAL=%s\n' "${FLOATVAL}"
printf 'BOOLVAL=%s\n' "${BOOLVAL}"
printf 'ENUMVAL=%s\n' "${ENUMVAL}"
printf 'LISTVAL=%s\n' "${LISTVAL}"
printf 'FLAGVAL=%s\n' "${FLAGVAL}"
printf 'DEFVAL=%s\n' "${DEFVAL}"

echo "--- Invalid enum value (should fail) ---"
if "${binary}" "${SCHEMA}" -s "fail" -e yellow 2>/dev/null; then
  echo "ERROR: Invalid enum value accepted!"
else
  echo "PASS: Invalid enum value rejected."
fi

echo "--- Missing required (should fail) ---"
if "${binary}" "${SCHEMA}" 2>/dev/null; then
  echo "ERROR: Missing required value accepted!"
else
  echo "PASS: Missing required value rejected."
fi

echo "--- List delimiter test (colon) ---"
list_delim_val=""
while IFS= read -r -d $'\0' k && IFS= read -r v; do
  if [[ "${k}" == "GO_SHOPTS_LISTVAL" ]]; then
    list_delim_val="${v}"
    break
  fi
done < <(GO_SHOPTS_LIST_DELIM=":" "${binary}" "${SCHEMA}" -s test -l a -l b -l c)

if [[ "${list_delim_val}" == "a:b:c" ]]; then
  echo "PASS: List delimiter works"
else
  echo "FAIL: List delimiter failed"
  echo "Expected: a:b:c"
  echo "Actual: ${list_delim_val}"
  exit 1
fi
