#!/usr/bin/env bash
# sessions.sh — Shared session utilities for workmode
# Used by workmode-sessions and workmode-tui

# Colors (disabled when piping)
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    DIM='\033[2m'
    RESET='\033[0m'
    BOLD='\033[1m'
else
    RED='' GREEN='' YELLOW='' BLUE='' DIM='' RESET='' BOLD=''
fi

# Extract JSON field value (simple, no jq dependency)
json_field() {
    local json="$1" field="$2"
    local result
    result="$(echo "$json" | grep -oP "\"${field}\"\\s*:\\s*\"[^\"]*\"" | head -1 | grep -oP '"[^"]*"$' | tr -d '"' || true)"
    echo "$result"
}

json_field_num() {
    local json="$1" field="$2"
    local result
    result="$(echo "$json" | grep -oP "\"${field}\"\\s*:\\s*[0-9]+" | head -1 | grep -oP '[0-9]+$' || true)"
    echo "$result"
}

format_duration() {
    local seconds="$1"
    [[ -z "$seconds" || "$seconds" == "0" ]] && echo "—" && return
    if (( seconds < 60 )); then
        echo "${seconds}s"
    elif (( seconds < 3600 )); then
        echo "$(( seconds / 60 ))m"
    else
        echo "$(( seconds / 3600 ))h$(( (seconds % 3600) / 60 ))m"
    fi
}

format_time() {
    local iso_time="$1"
    [[ -z "$iso_time" ]] && echo "—" && return

    local today
    today="$(date +%Y-%m-%d)"
    local day="${iso_time%%T*}"
    local time="${iso_time#*T}"
    time="${time%%[-+]*}"  # strip timezone
    time="${time%:*}"      # HH:MM

    if [[ "$day" == "$today" ]]; then
        echo "$time today"
    else
        echo "$time ${day#*-}"  # MM-DD
    fi
}

status_icon() {
    local icon text color
    case "$1" in
        completed) icon="✅"; text="done";    color="$GREEN" ;;
        running)   icon="🔄"; text="run";     color="$BLUE" ;;
        stuck)     icon="⏸";  text="stuck";   color="$YELLOW" ;;
        error)     icon="❌"; text="err";     color="$RED" ;;
        stopped)   icon="⏹";  text="stopped"; color="$YELLOW" ;;
        killed)    icon="💀"; text="killed";  color="$RED" ;;
        *)         icon="?";  text="$1";      color="$RESET" ;;
    esac
    # Pad the visible text to 8 chars, then wrap in color codes
    printf "%s %b%-7s%b" "$icon" "$color" "$text" "$RESET"
}

# Plain-text status mark (for fzf, no ANSI width issues)
status_mark() {
    case "$1" in
        completed) echo "done" ;;
        running)   echo "RUN" ;;
        stuck)     echo "STUCK" ;;
        error)     echo "ERR" ;;
        stopped)   echo "STOP" ;;
        killed)    echo "KILL" ;;
        *)         echo "?" ;;
    esac
}

# Resolve a short or full session ID to the history line
resolve_session() {
    local lookup="$1"
    local history_file="$2"

    # Try full ID first
    local match
    match="$(grep "\"id\":\"${lookup}\"" "$history_file" 2>/dev/null | tail -1 || true)"
    if [[ -n "$match" ]]; then
        echo "$match"
        return 0
    fi

    # Try short ID
    match="$(grep "\"short\":\"${lookup}\"" "$history_file" 2>/dev/null | tail -1 || true)"
    if [[ -n "$match" ]]; then
        echo "$match"
        return 0
    fi

    return 1
}

find_session_log() {
    local lookup="$1"
    local log_dir="$2"
    local history_file="$3"

    # Direct log file by full ID
    if [[ -f "$log_dir/${lookup}.log" ]]; then
        echo "$log_dir/${lookup}.log"
        return 0
    fi

    # Resolve short/full ID to get the full ID for the log file
    local session_line
    session_line="$(resolve_session "$lookup" "$history_file")" || return 1

    local full_id
    full_id="$(json_field "$session_line" "id")"
    if [[ -n "$full_id" && -f "$log_dir/${full_id}.log" ]]; then
        echo "$log_dir/${full_id}.log"
        return 0
    fi

    return 1
}

# Parse stream-json log into readable output
format_session_log() {
    local log_file="$1"

    while IFS= read -r line; do
        [[ -z "$line" ]] && continue

        local type
        type="$(echo "$line" | grep -oP '"type"\s*:\s*"[^"]*"' | grep -oP '"[^"]*"$' | tr -d '"' 2>/dev/null || true)"

        case "$type" in
            assistant)
                local text
                text="$(echo "$line" | grep -oP '"content"\s*:\s*"[^"]*"' | head -1 | sed 's/"content"\s*:\s*"//;s/"$//' | sed 's/\\n/\n/g' 2>/dev/null || true)"
                if [[ -n "$text" ]]; then
                    echo -e "$text"
                fi
                ;;
            tool_use)
                local tool_name
                tool_name="$(echo "$line" | grep -oP '"name"\s*:\s*"[^"]*"' | head -1 | grep -oP '"[^"]*"$' | tr -d '"' 2>/dev/null || true)"
                if [[ -n "$tool_name" ]]; then
                    echo -e "${BLUE}[tool: ${tool_name}]${RESET}"
                fi
                ;;
            result)
                local result_text
                result_text="$(echo "$line" | grep -oP '"result"\s*:\s*"[^"]*"' | head -1 | sed 's/"result"\s*:\s*"//;s/"$//' | sed 's/\\n/\n/g' 2>/dev/null || true)"
                if [[ -n "$result_text" ]]; then
                    echo -e "${GREEN}${result_text}${RESET}"
                fi
                ;;
            "")
                echo "$line"
                ;;
        esac
    done < "$log_file"
}

# Extract a short summary from a session log file
extract_summary() {
    local log_file="$1"
    local max_len="${2:-60}"

    [[ -f "$log_file" ]] || return 0

    local summary=""

    # Strategy 1: "result" field from the result line (clean final output)
    summary="$(grep -m1 '"type":"result"' "$log_file" 2>/dev/null \
        | grep -oP '"result"\s*:\s*"[^"]*"' \
        | head -1 \
        | sed 's/"result"\s*:\s*"//;s/"$//' \
        | sed 's/\\n/ /g' \
        || true)"

    # Strategy 2: first assistant text content
    if [[ -z "$summary" ]]; then
        summary="$(grep -m1 '"type":"assistant"' "$log_file" 2>/dev/null \
            | grep -oP '"text":"[^"]*"' \
            | head -1 \
            | sed 's/"text":"//;s/"$//' \
            | sed 's/\\n/ /g' \
            || true)"
    fi

    # Truncate
    if [[ -n "$summary" ]]; then
        if (( ${#summary} > max_len )); then
            summary="${summary:0:$((max_len - 1))}…"
        fi
        echo "$summary"
    fi
}

# Get PID from a running session's history line
get_session_pid() {
    local session_line="$1"
    local status pid
    status="$(json_field "$session_line" "status")"
    if [[ "$status" == "running" ]]; then
        pid="$(json_field_num "$session_line" "pid")"
        [[ -n "$pid" && "$pid" != "0" ]] && echo "$pid"
    fi
}

# Stop or kill a session by ID
# Usage: terminate_session <id> <action> <history_file> <state_dir>
#   action: "stop" (SIGTERM) or "kill" (SIGKILL)
terminate_session() {
    local lookup="$1"
    local action="$2"
    local history_file="$3"
    local state_dir="$4"

    local session_line
    session_line="$(resolve_session "$lookup" "$history_file")" || {
        echo "Session '$lookup' not found." >&2
        return 1
    }

    local status full_id trigger short_id pid
    status="$(json_field "$session_line" "status")"
    full_id="$(json_field "$session_line" "id")"
    trigger="$(json_field "$session_line" "trigger")"
    short_id="$(json_field "$session_line" "short")"
    [[ -z "$short_id" ]] && short_id="$full_id"

    if [[ "$status" != "running" ]]; then
        echo "Session $short_id is not running (status: $status)." >&2
        return 1
    fi

    pid="$(get_session_pid "$session_line")"
    if [[ -z "$pid" ]]; then
        echo "No PID found for session $short_id." >&2
        return 1
    fi

    if ! kill -0 "$pid" 2>/dev/null; then
        # Process is gone — clean up stale running status
        local started working_dir
        started="$(json_field "$session_line" "started")"
        working_dir="$(json_field "$session_line" "working_dir")"
        local duration=$(( $(date +%s) - $(date -d "$started" +%s 2>/dev/null || echo "$(date +%s)") ))
        printf '{"id":"%s","short":"%s","trigger":"%s","working_dir":"%s","started":"%s","status":"error","duration":%d}\n' \
            "$full_id" "$short_id" "$trigger" "$working_dir" "$started" "$duration" \
            >> "$history_file"
        # Clean up stale lock if present
        rm -f "${state_dir}/locks/${trigger}.lock"
        echo "Process $pid is no longer alive. Marked session $short_id as error."
        return 0
    fi

    local signal label new_status
    case "$action" in
        stop) signal=15; label="SIGTERM"; new_status="stopped" ;;
        kill) signal=9;  label="SIGKILL"; new_status="killed" ;;
        *)    echo "Invalid action: $action" >&2; return 1 ;;
    esac

    if kill -"$signal" "$pid" 2>/dev/null; then
        echo "Sent $label to session $short_id (pid $pid)."

        # Log termination to history
        local started working_dir
        started="$(json_field "$session_line" "started")"
        working_dir="$(json_field "$session_line" "working_dir")"
        local duration=$(( $(date +%s) - $(date -d "$started" +%s 2>/dev/null || echo "$(date +%s)") ))
        printf '{"id":"%s","short":"%s","trigger":"%s","working_dir":"%s","started":"%s","status":"%s","duration":%d}\n' \
            "$full_id" "$short_id" "$trigger" "$working_dir" "$started" "$new_status" "$duration" \
            >> "$history_file"
    else
        echo "Failed to send $label to process $pid." >&2
        return 1
    fi
}

# Deduplicate history lines (latest entry per session ID wins)
# Reads from stdin, outputs deduplicated lines
dedup_sessions() {
    local filter="${1:-}"
    local count="${2:-0}"
    local seen_ids=()
    local n=0

    while IFS= read -r line; do
        [[ -z "$line" ]] && continue
        [[ "$line" == *'"id"'* ]] || continue
        local id
        id="$(json_field "$line" "id")"
        [[ -z "$id" ]] && continue

        local already_seen=false
        for seen in "${seen_ids[@]+"${seen_ids[@]}"}"; do
            if [[ "$seen" == "$id" ]]; then
                already_seen=true
                break
            fi
        done
        $already_seen && continue

        seen_ids+=("$id")

        # Apply filter if set
        if [[ -n "$filter" ]]; then
            local status
            status="$(json_field "$line" "status")"
            [[ "$status" != "$filter" ]] && continue
        fi

        echo "$line"
        (( ++n ))
        if (( count > 0 && n >= count )); then
            break
        fi
    done
}
