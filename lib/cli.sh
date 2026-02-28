#!/usr/bin/env bash
# cli.sh — Shared CLI framework for workmode commands

# Exit codes
EX_OK=0
EX_USAGE=2
EX_CONFIG=3
EX_NOT_FOUND=4
EX_STATE=5
EX_DEPENDENCY=6

# Output mode (set by parse_global_flags)
OUTPUT_FORMAT="text"
SHOW_HELP=false

die() {
    local code="${code:-1}"
    printf "error: %s\n" "$*" >&2
    exit "$code"
}

warn() {
    printf "warning: %s\n" "$*" >&2
}

# Parse --json and --help from argument list, set globals, echo remaining args
# Usage: eval "$(parse_global_flags "$@")"
parse_global_flags() {
    local args=()
    OUTPUT_FORMAT="text"
    SHOW_HELP=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)  OUTPUT_FORMAT="json" ;;
            --help|-h) SHOW_HELP=true ;;
            *)       args+=("$1") ;;
        esac
        shift
    done

    # Re-set positional parameters to remaining args
    set -- "${args[@]+"${args[@]}"}"
}

# JSON helpers — build JSON without jq
json_escape() {
    local s="$1"
    s="${s//\\/\\\\}"
    s="${s//\"/\\\"}"
    s="${s//$'\n'/\\n}"
    s="${s//$'\r'/\\r}"
    s="${s//$'\t'/\\t}"
    printf '%s' "$s"
}

json_field() {
    local key="$1" value="$2"
    printf '"%s":"%s"' "$key" "$(json_escape "$value")"
}

json_field_num() {
    local key="$1" value="$2"
    printf '"%s":%s' "$key" "$value"
}

json_field_bool() {
    local key="$1" value="$2"
    printf '"%s":%s' "$key" "$value"
}

json_field_array() {
    local key="$1"
    shift
    local items=""
    for item in "$@"; do
        [[ -n "$items" ]] && items+=","
        items+="\"$(json_escape "$item")\""
    done
    printf '"%s":[%s]' "$key" "$items"
}

json_object() {
    printf '{%s}' "$*"
}
