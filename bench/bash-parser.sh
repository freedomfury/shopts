#!/usr/bin/env bash
set -euo pipefail

prefix="${GO_GETOPT_PREFIX:-GO_GETOPT_}"
list_delim="${GO_GETOPT_LIST_DELIM:-,}"
upcase="${GO_GETOPT_UPCASE:-}"

usage() {
    cat <<USAGE
Usage: $0 [--help] [-u|--user USER] [-p|--pass PASS] [-m|--mode MODE] [-t|--tags TAG]...
Options:
  -u, --user   USER   (required)
  -p, --pass   PASS   (required)
  -m, --mode   MODE   (dev|prod|test) default: dev
  -t, --tags   TAG    repeatable
USAGE
}

# defaults
_user=""
_pass=""
_mode="dev"
_tags=()
_verbose=""

# parse
while [ $# -gt 0 ]; do
    case "$1" in
    -h | --help)
        usage
        exit 0
        ;;
    --)
        shift
        break
        ;;
    --user=*)
        _user="${1#*=}"
        shift
        ;;
    --user)
        _user="$2"
        shift 2
        ;;
    -u=*)
        _user="${1#*=}"
        shift
        ;;
    -u=*)
        _user="${1#*=}"
        shift
        ;;
    -u)
        _user="$2"
        shift 2
        ;;
    --pass=*)
        _pass="${1#*=}"
        shift
        ;;
    --pass)
        _pass="$2"
        shift 2
        ;;
    -p=*)
        _pass="${1#*=}"
        shift
        ;;
    -p)
        _pass="$2"
        shift 2
        ;;
    --mode=*)
        _mode="${1#*=}"
        shift
        ;;
    --mode)
        _mode="$2"
        shift 2
        ;;
    -m=*)
        _mode="${1#*=}"
        shift
        ;;
    -m)
        _mode="$2"
        shift 2
        ;;
    --tags=*)
        _tags+=("${1#*=}")
        shift
        ;;
    --tags)
        _tags+=("$2")
        shift 2
        ;;
    -t=*)
        _tags+=("${1#*=}")
        shift
        ;;
    -t)
        _tags+=("$2")
        shift 2
        ;;
    -v)
        _verbose="true"
        shift
        ;;
    --verbose)
        _verbose="true"
        shift
        ;;
    *)
        printf 'Unknown option: %s\n' "$1" >&2
        exit 2
        ;;
    esac
done

# validations
if [ -z "${_user}" ]; then
    printf 'missing required option "--user"\n' >&2
    exit 1
fi
if [ -z "${_pass}" ]; then
    printf 'missing required option "--pass"\n' >&2
    exit 1
fi
case "${_mode}" in
dev | prod | test) : ;;
*)
    printf 'option "--mode" invalid: must be one of: dev, prod, test\n' >&2
    exit 1
    ;;
esac

# helper to emit KEY\0VALUE\n records
_kv() {
    local key="$1"
    local val="$2"
    if [ -n "${upcase}" ] && [ "${upcase}" != "0" ]; then
        key="$(printf '%s' "${key}" | tr '[:lower:]' '[:upper:]')"
    fi
    key="${prefix}${key}"
    printf '%s\0%s\n' "${key}" "${val}"
}

# emit
_kv "user" "${_user}"
_kv "pass" "${_pass}"
_kv "mode" "${_mode}"

if [[ "${#_tags[@]}" -gt 0 ]]; then
    joined=""
    printf -v joined '%s' "${_tags[0]}"
    for ((i = 1; i < ${#_tags[@]}; i++)); do
        joined="${joined}${list_delim}${_tags[i]}"
    done
    _kv "tags" "${joined}"
fi

exit 0
