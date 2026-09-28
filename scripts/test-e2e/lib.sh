#!/usr/bin/env bash
## Shared helpers for the e2e tests. A test sources this file:
##   source "$(dirname "$0")/lib.sh"
## The runner passes the binary as the test's first argument.

BINARY=${1:-bin/shopts}

# Settings from the caller's environment must not change the output.
unset "${!GO_SHOPTS_@}"

_e2e_out=$(mktemp)
_e2e_err=$(mktemp)
trap 'rm -f "${_e2e_out}" "${_e2e_err}"' EXIT

# _e2e_fail MESSAGE: report what the binary printed, then fail the test.
_e2e_fail() {
    printf 'FAIL: %s\n' "$1" >&2
    printf -- '--- stdout\n' >&2
    cat "${_e2e_out}" >&2
    printf -- '--- stderr\n' >&2
    cat "${_e2e_err}" >&2
    exit 1
}

# expect_ok SCHEMA ARGS... <<'EOF'
# SHOPTS_KEY=value
# EOF
# Expects exit 0, empty stderr, and exactly the given lines on stdout. In each
# expected line the first '=' stands for the tab between key and value.
expect_ok() {
    local line want="" rc=0
    while IFS= read -r line; do
        want+="${line/=/$'\t'}"$'\n'
    done
    "${BINARY}" "$@" >"${_e2e_out}" 2>"${_e2e_err}" || rc=$?
    [[ ${rc} -eq 0 ]] || _e2e_fail "exit ${rc}, want 0"
    [[ ! -s "${_e2e_err}" ]] || _e2e_fail "unexpected stderr"
    if ! diff -u <(printf '%s' "${want}") "${_e2e_out}" >&2; then
        _e2e_fail "stdout differs from the expected lines (diff above: - want, + got)"
    fi
}

# expect_fail RC MESSAGE SCHEMA ARGS...
# Expects exit RC, empty stdout, and MESSAGE somewhere in stderr.
expect_fail() {
    local want_rc=$1 want_err=$2 rc=0
    shift 2
    "${BINARY}" "$@" >"${_e2e_out}" 2>"${_e2e_err}" || rc=$?
    [[ ${rc} -eq ${want_rc} ]] || _e2e_fail "exit ${rc}, want ${want_rc}"
    [[ ! -s "${_e2e_out}" ]] || _e2e_fail "stdout must be empty on error"
    grep -qF -- "${want_err}" "${_e2e_err}" || _e2e_fail "stderr does not contain: ${want_err}"
}
