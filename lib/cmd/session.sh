#!/usr/bin/env bash
# lib/cmd/session.sh — Session management commands

dispatch_session() {
    local subcmd="${1:-list}"
    shift || true

    case "$subcmd" in
        list)    cmd_session_list "$@" ;;
        logs|log) cmd_session_logs "$@" ;;
        tail)    cmd_session_tail "$@" ;;
        resume)  cmd_session_resume "$@" ;;
        stop)    cmd_session_stop "$@" ;;
        kill)    cmd_session_kill "$@" ;;
        help|--help|-h) usage_session ;;
        *)
            # If it looks like a session ID, treat as logs
            if [[ "$subcmd" =~ ^wm- ]] || [[ "$subcmd" =~ ^[a-z]+-[0-9a-f]{4}$ ]]; then
                cmd_session_logs "$subcmd" "$@"
            else
                die "Unknown session command: $subcmd"
            fi
            ;;
    esac
}

usage_session() {
    cat <<EOF
Usage: workmode session <command> [options]

Commands:
  list [--json] [--running|--stuck|--completed]   List recent sessions
  logs <id>                                        Show session output
  tail <id>                                        Follow running session
  resume <id>                                      Resume interactive session
  stop <id>                                        Graceful stop (SIGTERM)
  kill <id>                                        Force kill (SIGKILL)

Options:
  --running       Show only running sessions
  --stuck         Show only stuck sessions
  --completed     Show only completed sessions
  --all           Show all sessions (default: last 20)
  -n <count>      Number of sessions to show
  --json          Output as newline-delimited JSON

EOF
    exit 0
}

cmd_session_list() {
    local filter="" count=20

    # Parse flags before global flags
    local args=()
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --running)   filter="running"; shift ;;
            --stuck)     filter="stuck"; shift ;;
            --completed) filter="completed"; shift ;;
            --all)       count=0; shift ;;
            -n)          count="$2"; shift 2 ;;
            *)           args+=("$1"); shift ;;
        esac
    done
    parse_global_flags "${args[@]+"${args[@]}"}"
    $SHOW_HELP && usage_session

    if [[ ! -f "$HISTORY_FILE" ]]; then
        if [[ "$OUTPUT_FORMAT" == "json" ]]; then
            return
        fi
        echo "No sessions yet."
        return
    fi

    local lines=()
    while IFS= read -r line; do
        lines+=("$line")
    done < <(tac "$HISTORY_FILE" | dedup_sessions "$filter" "$count")

    if [[ ${#lines[@]} -eq 0 ]]; then
        if [[ "$OUTPUT_FORMAT" == "json" ]]; then
            return
        fi
        echo "No sessions found${filter:+ with status '$filter'}."
        return
    fi

    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        printf '%s\n' "${lines[@]}"
        return
    fi

    # Table header
    printf "${BOLD}%-10s %-14s %-14s %-10s %-14s %s${RESET}\n" \
        "STATUS" "TRIGGER" "STARTED" "DURATION" "ID" "SUMMARY"

    for line in "${lines[@]}"; do
        local status trigger started duration short_id full_id summary
        status="$(sess_json_field "$line" "status")"
        trigger="$(sess_json_field "$line" "trigger")"
        started="$(sess_json_field "$line" "started")"
        duration="$(sess_json_field_num "$line" "duration")"
        short_id="$(sess_json_field "$line" "short")"
        [[ -z "$short_id" ]] && short_id="$(sess_json_field "$line" "id")"
        full_id="$(sess_json_field "$line" "id")"

        summary=""
        if [[ -f "$LOG_DIR/${full_id}.log" && -s "$LOG_DIR/${full_id}.log" ]]; then
            summary="$(extract_summary "$LOG_DIR/${full_id}.log" 50)"
        fi
        if [[ -z "$summary" ]]; then
            local stderr_file="$LOG_DIR/${full_id}.stderr"
            if [[ -f "$stderr_file" && -s "$stderr_file" ]]; then
                summary="$(head -1 "$stderr_file" | head -c 50)"
                (( ${#summary} > 49 )) && summary="${summary:0:49}…"
            fi
        fi
        if [[ -z "$summary" ]]; then
            local error_hint
            error_hint="$(sess_json_field "$line" "error")"
            if [[ -n "$error_hint" ]]; then
                summary="$error_hint"
                (( ${#summary} > 50 )) && summary="${summary:0:49}…"
            else
                local exit_code
                exit_code="$(sess_json_field_num "$line" "exit_code")"
                if [[ -n "$exit_code" && "$exit_code" != "0" ]]; then
                    summary="exit code $exit_code"
                fi
            fi
        fi

        printf "%s %-14s %-14s %-10s %-14s %b%s%b\n" \
            "$(status_icon "$status")" \
            "$trigger" \
            "$(format_time "$started")" \
            "$(format_duration "$duration")" \
            "$short_id" \
            "$DIM" "$summary" "$RESET"
    done
}

cmd_session_logs() {
    local target_id="${1:-}"
    [[ -z "$target_id" ]] && { code=$EX_USAGE die "Usage: workmode session logs <session-id>"; }

    # Show session metadata from history
    local session_line
    session_line="$(resolve_session "$target_id" "$HISTORY_FILE" 2>/dev/null || true)"

    if [[ -n "$session_line" ]]; then
        local full_id status trigger started duration exit_code attempt session_id working_dir
        full_id="$(sess_json_field "$session_line" "id")"
        status="$(sess_json_field "$session_line" "status")"
        trigger="$(sess_json_field "$session_line" "trigger")"
        started="$(sess_json_field "$session_line" "started")"
        duration="$(sess_json_field_num "$session_line" "duration")"
        exit_code="$(sess_json_field_num "$session_line" "exit_code")"
        attempt="$(sess_json_field_num "$session_line" "attempt")"
        session_id="$(sess_json_field "$session_line" "session_id")"
        working_dir="$(sess_json_field "$session_line" "working_dir")"

        echo -e "${BOLD}Session: ${target_id}${RESET}"
        echo "Status:   $status"
        echo "Trigger:  $trigger"
        echo "Started:  $(format_time "$started")"
        [[ -n "$duration" && "$duration" != "0" ]] && echo "Duration: $(format_duration "$duration")"
        [[ -n "$working_dir" ]] && echo "Dir:      $working_dir"
        [[ -n "$session_id" ]] && echo "Claude:   $session_id"
        [[ -n "$exit_code" ]] && echo -e "Exit:     ${RED}${exit_code}${RESET}"
        [[ -n "$attempt" && "$attempt" != "1" ]] && echo "Attempt:  $attempt"
        echo ""
    else
        echo -e "${BOLD}Session: ${target_id}${RESET}"
        echo ""
    fi

    local log_file
    log_file="$(find_session_log "$target_id" "$LOG_DIR" "$HISTORY_FILE" 2>/dev/null || true)"

    if [[ -n "$log_file" && -s "$log_file" ]]; then
        echo -e "${DIM}─── Log output ───${RESET}"
        echo ""
        format_session_log "$log_file"
    elif [[ -n "$log_file" && -f "$log_file" ]]; then
        echo -e "${DIM}(log file is empty — session may have crashed before producing output)${RESET}"
    else
        echo -e "${DIM}(no log file found)${RESET}"
    fi

    # Show stderr if present
    if [[ -n "$log_file" ]]; then
        local stderr_file="${log_file%.log}.stderr"
        if [[ -f "$stderr_file" && -s "$stderr_file" ]]; then
            echo ""
            echo -e "${DIM}─── stderr ───${RESET}"
            echo ""
            echo -e "${RED}$(cat "$stderr_file")${RESET}"
        fi
    fi
}

cmd_session_tail() {
    local target_id="${1:-}"
    [[ -z "$target_id" ]] && { code=$EX_USAGE die "Usage: workmode session tail <session-id>"; }

    local log_file
    log_file="$(find_session_log "$target_id" "$LOG_DIR" "$HISTORY_FILE")" || {
        echo "No log found for session '$target_id'." >&2
        echo "Logs are stored in: $LOG_DIR/" >&2
        exit 1
    }

    echo -e "${BOLD}Tailing session: ${target_id}${RESET}"
    echo -e "(Ctrl+C to stop)"
    echo ""

    if [[ -s "$log_file" ]]; then
        format_session_log "$log_file"
        echo ""
    fi

    tail -n 0 -f "$log_file" | while IFS= read -r line; do
        [[ -z "$line" ]] && continue
        if command -v jq &>/dev/null; then
            echo "$line" | jq -r --arg blue "${BLUE}" --arg green "${GREEN}" --arg reset "${RESET}" '
                if .type == "assistant" then
                    (.message.content // [])[] |
                    if .type == "text" then .text
                    elif .type == "tool_use" then "\($blue)[tool: \(.name)]\($reset)"
                    else empty
                    end
                elif .type == "result" then
                    .result // empty | "\($green)\(.)\($reset)"
                else empty
                end
            ' 2>/dev/null || true
        else
            local type
            type="$(echo "$line" | grep -oP '"type"\s*:\s*"[^"]*"' | head -1 | grep -oP '"[^"]*"$' | tr -d '"' 2>/dev/null || true)"
            [[ "$type" == "assistant" || "$type" == "result" ]] && echo "$line"
        fi
    done
}

cmd_session_resume() {
    local resume_id="${1:-}"
    [[ -z "$resume_id" ]] && { code=$EX_USAGE die "Usage: workmode session resume <id>"; }

    local session_line
    session_line="$(resolve_session "$resume_id" "$HISTORY_FILE")" || {
        code=$EX_NOT_FOUND die "Session '$resume_id' not found."
    }

    local claude_session_id
    claude_session_id="$(sess_json_field "$session_line" "session_id")"

    if [[ -z "$claude_session_id" ]]; then
        echo "No Claude session ID found for '$resume_id'." >&2
        echo "The session may not have started successfully."
        exit 1
    fi

    local working_dir
    working_dir="$(sess_json_field "$session_line" "working_dir")"
    if [[ -n "$working_dir" && -d "$working_dir" ]]; then
        echo "Resuming in: $working_dir"
        cd "$working_dir"
    fi

    echo "Resuming Claude session: $claude_session_id"
    exec claude --resume "$claude_session_id"
}

cmd_session_stop() {
    local target_id="${1:-}"
    [[ -z "$target_id" ]] && { code=$EX_USAGE die "Usage: workmode session stop <id>"; }
    terminate_session "$target_id" "stop" "$HISTORY_FILE" "$STATE_DIR"
}

cmd_session_kill() {
    local target_id="${1:-}"
    [[ -z "$target_id" ]] && { code=$EX_USAGE die "Usage: workmode session kill <id>"; }
    terminate_session "$target_id" "kill" "$HISTORY_FILE" "$STATE_DIR"
}

# Alias for sessions.sh json_field to avoid conflict with cli.sh json_field
sess_json_field() {
    local json="$1" field="$2"
    local result
    result="$(echo "$json" | grep -oP "\"${field}\"\\s*:\\s*\"[^\"]*\"" | head -1 | grep -oP '"[^"]*"$' | tr -d '"' || true)"
    echo "$result"
}

sess_json_field_num() {
    local json="$1" field="$2"
    local result
    result="$(echo "$json" | grep -oP "\"${field}\"\\s*:\\s*[0-9]+" | head -1 | grep -oP '[0-9]+$' || true)"
    echo "$result"
}
