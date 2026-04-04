#!/usr/bin/env bash
## NOTE: Run this script from the project root (../scripts/test-extensive.sh)
set -euo pipefail

# Extensive test for shopts: covers all supported types, variables, and edge cases

SCHEMA='
short=s;long=stringval;required=true;type=string;help=A required string value;
short=i;long=intval;required=false;type=int;help=Optional integer value;default=42;
short=f;long=floatval;required=false;type=float;help=Optional float value;default=3.14;
short=b;long=boolval;required=false;type=bool;help=Optional boolean value;default=false;
short=B;long=booltrue;required=false;type=bool;help=Bool that defaults to true;default=true;
short=e;long=enumval;required=false;type=enum;enum=red,green,blue;default=green;help=Enum value;
short=l;long=listval;required=false;type=list;help=Optional list value;
short=t;long=taglist;required=false;type=list;minItems=1;maxItems=5;help=Tags list with item constraints;
short=F;long=flagval;required=false;type=flag;help=Optional flag;
short=T;long=trueflag;required=false;type=flag;default=true;help=Flag that defaults to true;
short=d;long=defval;required=false;type=string;help=Has a default;default=defaultval;
short=n;long=nameval;required=false;type=string;minLength=2;maxLength=10;help=Name with length constraints;default=hi;
short=p;long=patternval;required=false;type=string;pattern=^[a-z]+$;failure=must be lowercase letters only;help=Pattern validated string;default=abc;
short=w;long=wordonly;required=false;type=string;pattern=^\D+$;failure=this is not a string;help=Pattern-validated word (rejects numbers);default=hello;
long=longonly;required=false;type=string;help=Long-only option no short flag;description=This option has no short flag.;default=longonly;
'

binary=bin/shopts
if [[ ! -x "${binary}" ]]; then
  go build -o "${binary}" ./cmd/shopts
fi

# Test: all options provided
while IFS= read -r -d $'\0' k && IFS= read -r v; do
  printf -v "${k}" '%s' "${v}"
  declare -x "${k#SHOPTS_}"="${v}"
done < <("${binary}" "${SCHEMA}" \
  -s "hello" -i 99 -f 2.71 -b true -B false -e blue -l a,b,c \
  -t tag1 -t tag2 -F -d "customdef" --nameval=valid --patternval=abc \
  -w "test word" --longonly=custom)

echo "--- All options provided ---"
printf 'STRINGVAL=%s\n' "${STRINGVAL}"
printf 'INTVAL=%s\n' "${INTVAL}"
printf 'FLOATVAL=%s\n' "${FLOATVAL}"
printf 'BOOLVAL=%s\n' "${BOOLVAL}"
printf 'BOOLTRUE=%s\n' "${BOOLTRUE}"
printf 'ENUMVAL=%s\n' "${ENUMVAL}"
printf 'LISTVAL=%s\n' "${LISTVAL}"
printf 'TAGLIST=%s\n' "${TAGLIST}"
printf 'FLAGVAL=%s\n' "${FLAGVAL}"
printf 'TRUEFLAG=%s\n' "${TRUEFLAG}"
printf 'DEFVAL=%s\n' "${DEFVAL}"
printf 'NAMEVAL=%s\n' "${NAMEVAL}"
printf 'PATTERNVAL=%s\n' "${PATTERNVAL}"
printf 'WORDONLY=%s\n' "${WORDONLY}"
printf 'LONGONLY=%s\n' "${LONGONLY}"

echo "--- Defaults and missing values ---"
# Test: only required and some optional
while IFS= read -r -d $'\0' k && IFS= read -r v; do
  printf -v "${k}" '%s' "${v}"
  declare -x "${k#SHOPTS_}"="${v}"
done < <("${binary}" "${SCHEMA}" -s "world" -t only -F)
printf 'STRINGVAL=%s\n' "${STRINGVAL}"
printf 'INTVAL=%s\n' "${INTVAL}"
printf 'FLOATVAL=%s\n' "${FLOATVAL}"
printf 'BOOLVAL=%s\n' "${BOOLVAL}"
printf 'BOOLTRUE=%s\n' "${BOOLTRUE}"
printf 'ENUMVAL=%s\n' "${ENUMVAL}"
printf 'LISTVAL=%s\n' "${LISTVAL}"
printf 'TAGLIST=%s\n' "${TAGLIST}"
printf 'FLAGVAL=%s\n' "${FLAGVAL}"
printf 'TRUEFLAG=%s\n' "${TRUEFLAG}"
printf 'DEFVAL=%s\n' "${DEFVAL}"
printf 'NAMEVAL=%s\n' "${NAMEVAL}"
printf 'PATTERNVAL=%s\n' "${PATTERNVAL}"
printf 'WORDONLY=%s\n' "${WORDONLY}"
printf 'LONGONLY=%s\n' "${LONGONLY}"

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
  if [[ "${k}" == "SHOPTS_LISTVAL" ]]; then
    list_delim_val="${v}"
    break
  fi
done < <(GO_SHOPTS_LIST_DELIM=":" "${binary}" "${SCHEMA}" -s test -l a -l b -l c -t one)

if [[ "${list_delim_val}" == "a:b:c" ]]; then
  echo "PASS: List delimiter works"
else
  echo "FAIL: List delimiter failed"
  echo "Expected: a:b:c"
  echo "Actual: ${list_delim_val}"
  exit 1
fi

echo "--- String type accepts numeric input (strings are not type-narrowed) ---"
if "${binary}" "${SCHEMA}" -s "123" -t one >/dev/null 2>&1; then
  echo "PASS: Numeric string accepted as type=string."
else
  echo "FAIL: Numeric string rejected for type=string."
  exit 1
fi

echo "--- Int type accepts numeric strings but rejects words (strconv asymmetry) ---"
if "${binary}" "${SCHEMA}" -s test -t one -i 456 >/dev/null 2>&1; then
  echo "PASS: Numeric string accepted by type=int."
else
  echo "FAIL: Numeric string rejected by type=int."
  exit 1
fi

echo "--- Int type rejects non-numeric words (cannot convert arbitrary strings to numbers) ---"
if "${binary}" "${SCHEMA}" -s test -t one -i notanumber 2>/dev/null; then
  echo "ERROR: Word accepted as type=int!"
  exit 1
else
  echo "PASS: Word rejected by type=int."
fi

echo "--- Pattern validation pass ---"
if "${binary}" "${SCHEMA}" -s test -t one -p "validlower" >/dev/null 2>&1; then
  echo "PASS: Valid pattern accepted."
else
  echo "FAIL: Valid pattern rejected."
  exit 1
fi

echo "--- Pattern validation fail (custom failure message) ---"
if "${binary}" "${SCHEMA}" -s test -t one -p "INVALID123" 2>/dev/null; then
  echo "ERROR: Invalid pattern value accepted!"
  exit 1
else
  echo "PASS: Invalid pattern value rejected."
fi

echo "--- minLength pass ---"
if "${binary}" "${SCHEMA}" -s test -t one -n "ok" >/dev/null 2>&1; then
  echo "PASS: Value at minLength accepted."
else
  echo "FAIL: Value at minLength rejected."
  exit 1
fi

echo "--- minLength fail (too short) ---"
if "${binary}" "${SCHEMA}" -s test -t one -n "x" 2>/dev/null; then
  echo "ERROR: Too-short value accepted!"
  exit 1
else
  echo "PASS: Too-short value rejected."
fi

echo "--- maxLength fail (too long) ---"
if "${binary}" "${SCHEMA}" -s test -t one -n "toolongvalue" 2>/dev/null; then
  echo "ERROR: Too-long value accepted!"
  exit 1
else
  echo "PASS: Too-long value rejected."
fi

echo "--- maxItems fail (too many list items) ---"
if "${binary}" "${SCHEMA}" -s test -t one -t two -t three -t four -t five -t six 2>/dev/null; then
  echo "ERROR: Too many list items accepted!"
  exit 1
else
  echo "PASS: Too many list items rejected."
fi

echo "--- minItems boundary (exactly minItems=1 item passes) ---"
if "${binary}" "${SCHEMA}" -s test -t exactly-one >/dev/null 2>&1; then
  echo "PASS: Exactly minItems items accepted."
else
  echo "FAIL: Exactly minItems items rejected."
  exit 1
fi

echo "--- flag default=true (not passed) ---"
trueflag_val=""
while IFS= read -r -d $'\0' k && IFS= read -r v; do
  if [[ "${k}" == "SHOPTS_TRUEFLAG" ]]; then
    trueflag_val="${v}"
    break
  fi
done < <("${binary}" "${SCHEMA}" -s test -t one)

if [[ "${trueflag_val}" == "true" ]]; then
  echo "PASS: flag default=true emits true when not passed."
else
  echo "FAIL: flag default=true did not emit true (got: ${trueflag_val})"
  exit 1
fi

echo "--- inline --long=value syntax ---"
inline_val=""
while IFS= read -r -d $'\0' k && IFS= read -r v; do
  if [[ "${k}" == "SHOPTS_STRINGVAL" ]]; then
    inline_val="${v}"
    break
  fi
done < <("${binary}" "${SCHEMA}" --stringval=inlinetest -t one)

if [[ "${inline_val}" == "inlinetest" ]]; then
  echo "PASS: --long=value inline syntax works."
else
  echo "FAIL: --long=value inline syntax failed (got: ${inline_val})"
  exit 1
fi

echo "--- GO_SHOPTS_PREFIX override ---"
prefixed_val=""
while IFS= read -r -d $'\0' k && IFS= read -r v; do
  if [[ "${k}" == "MYAPP_STRINGVAL" ]]; then
    prefixed_val="${v}"
    break
  fi
done < <(GO_SHOPTS_PREFIX="MYAPP_" "${binary}" "${SCHEMA}" -s prefixtest -t one)

if [[ "${prefixed_val}" == "prefixtest" ]]; then
  echo "PASS: GO_SHOPTS_PREFIX works."
else
  echo "FAIL: GO_SHOPTS_PREFIX failed (got: ${prefixed_val})"
  exit 1
fi

echo "--- Pattern rejects numbers with custom failure message ---"
if "${binary}" "${SCHEMA}" -s test -t one -w "hello world" >/dev/null 2>&1; then
  echo "PASS: Text accepted by word-only pattern."
else
  echo "FAIL: Text rejected by word-only pattern."
  exit 1
fi

echo "--- Pattern with custom failure message (rejects number input) ---"
error_output=$("${binary}" "${SCHEMA}" -s test -t one -w "hello123" 2>&1 || true)
if echo "$error_output" | grep -q "this is not a string"; then
  echo "PASS: Custom failure message shown for rejected input."
else
  echo "FAIL: Custom failure message not shown."
  echo "Got: $error_output" | tail -1
  exit 1
fi

echo "--- long-only option emitted correctly ---"
longonly_val=""
while IFS= read -r -d $'\0' k && IFS= read -r v; do
  if [[ "${k}" == "SHOPTS_LONGONLY" ]]; then
    longonly_val="${v}"
    break
  fi
done < <("${binary}" "${SCHEMA}" -s test -t one --longonly=fromtest)

if [[ "${longonly_val}" == "fromtest" ]]; then
  echo "PASS: Long-only option works."
else
  echo "FAIL: Long-only option failed (got: ${longonly_val})"
  exit 1
fi

echo "--- All tests passed ---"
