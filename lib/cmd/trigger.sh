#!/usr/bin/env bash
# lib/cmd/trigger.sh — Trigger management commands

dispatch_trigger() {
    local subcmd="${1:-list}"
    shift || true

    case "$subcmd" in
        list)    cmd_trigger_list "$@" ;;
        show)    cmd_trigger_show "$@" ;;
        run)     cmd_trigger_run "$@" ;;
        enable)  cmd_trigger_enable "$@" ;;
        disable) cmd_trigger_disable "$@" ;;
        help|--help|-h) usage_trigger ;;
        *)       die "Unknown trigger command: $subcmd" ;;
    esac
}

usage_trigger() {
    cat <<EOF
Usage: workmode trigger <command> [options]

Commands:
  list [--json]          List all configured triggers
  show <name> [--json]   Show parsed config for one trigger
  run <name>             Manually run a trigger
  enable <name>          Enable a trigger's systemd unit
  disable <name>         Disable a trigger's systemd unit

EOF
    exit 0
}

cmd_trigger_list() {
    parse_global_flags "$@"
    $SHOW_HELP && usage_trigger

    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        for name in $(config_list_triggers); do
            _trigger_to_json "$name"
        done
        return
    fi

    printf "  %-14s %-8s %-10s %-28s %s\n" "NAME" "TYPE" "PERMS" "PROMPT/SKILL" "SCHEDULE/WATCH"
    echo "  ────────────── ──────── ────────── ──────────────────────────── ──────────────────────"

    for name in $(config_list_triggers); do
        local type skill prompt_text label permissions

        type="$(config_trigger_field "$name" "type" || echo "?")"
        skill="$(config_trigger_field "$name" "skill" || true)"
        prompt_text="$(config_trigger_field "$name" "prompt" || true)"
        permissions="$(config_trigger_field "$name" "permissions" || echo "default")"

        if [[ -n "$skill" ]]; then
            label="$skill"
        elif [[ -n "$prompt_text" ]]; then
            label="${prompt_text:0:26}"
            (( ${#prompt_text} > 26 )) && label="${label}…"
        else
            label="?"
        fi

        local schedule_info=""
        schedule_info="$(_trigger_schedule_info "$name" "$type")"

        printf "  %-14s %-8s %-10s %-28s %s\n" \
            "$name" "$type" "$permissions" "$label" "$schedule_info"
    done
}

cmd_trigger_show() {
    parse_global_flags "$@"
    $SHOW_HELP && { echo "Usage: workmode trigger show <name> [--json]"; exit 0; }

    local trigger_name="${1:-}"
    [[ -z "$trigger_name" ]] && { code=$EX_USAGE die "Usage: workmode trigger show <name>"; }

    # Verify trigger exists
    local found=false
    for name in $(config_list_triggers); do
        [[ "$name" == "$trigger_name" ]] && { found=true; break; }
    done
    $found || { code=$EX_NOT_FOUND die "Trigger '$trigger_name' not found"; }

    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        _trigger_to_json "$trigger_name"
        return
    fi

    local type skill prompt_text permissions working_dir cooldown check
    type="$(config_trigger_field "$trigger_name" "type" || echo "?")"
    skill="$(config_trigger_field "$trigger_name" "skill" || true)"
    prompt_text="$(config_trigger_field "$trigger_name" "prompt" || true)"
    permissions="$(config_trigger_field "$trigger_name" "permissions" || echo "default")"
    working_dir="$(config_trigger_field "$trigger_name" "working_dir" || true)"
    cooldown="$(config_trigger_field "$trigger_name" "cooldown" || true)"
    check="$(config_trigger_field "$trigger_name" "check" || true)"

    echo "Trigger:     $trigger_name"
    echo "Type:        $type"
    echo "Permissions: $permissions"
    [[ -n "$working_dir" ]] && echo "Working dir: $working_dir"
    [[ -n "$skill" ]] && echo "Skill:       $skill"
    [[ -n "$prompt_text" ]] && echo "Prompt:      $prompt_text"

    if [[ "$type" == "timer" ]]; then
        local interval cron_expr
        interval="$(config_trigger_field "$trigger_name" "interval" || true)"
        cron_expr="$(config_trigger_field "$trigger_name" "cron" || true)"
        [[ -n "$interval" ]] && echo "Interval:    $interval"
        [[ -n "$cron_expr" ]] && echo "Cron:        $cron_expr"
    elif [[ "$type" == "file" ]]; then
        local watch pattern settle
        watch="$(config_trigger_field "$trigger_name" "watch" || true)"
        pattern="$(config_trigger_field "$trigger_name" "pattern" || true)"
        settle="$(config_trigger_field "$trigger_name" "settle" || true)"
        [[ -n "$watch" ]] && echo "Watch:       $watch"
        [[ -n "$pattern" ]] && echo "Pattern:     $pattern"
        [[ -n "$settle" ]] && echo "Settle:      ${settle}s"
    fi

    [[ -n "$cooldown" ]] && echo "Cooldown:    ${cooldown}s"
    [[ -n "$check" ]] && echo "Check:       $check"

    # Retry settings
    local retry retry_max retry_delay
    retry="$(config_trigger_field "$trigger_name" "retry" || true)"
    retry_max="$(config_trigger_field "$trigger_name" "retry_max" || true)"
    retry_delay="$(config_trigger_field "$trigger_name" "retry_delay" || true)"
    [[ -n "$retry" ]] && echo "Retry:       $retry"
    [[ -n "$retry_max" ]] && echo "Retry max:   $retry_max"
    [[ -n "$retry_delay" ]] && echo "Retry delay: ${retry_delay}s"

    # Systemd unit status
    local unit_name="${UNIT_PREFIX}${trigger_name}"
    if [[ "$type" == "timer" ]]; then
        if systemctl --user is-enabled --quiet "${unit_name}.timer" 2>/dev/null; then
            echo "Systemd:     enabled"
        else
            echo "Systemd:     disabled"
        fi
    fi
}

cmd_trigger_run() {
    local trigger_name="${1:-}"
    [[ -z "$trigger_name" ]] && { code=$EX_USAGE die "Usage: workmode trigger run <name>"; }
    exec "$BIN_DIR/workmode-run" --trigger "$trigger_name"
}

cmd_trigger_enable() {
    local trigger_name="${1:-}"
    [[ -z "$trigger_name" ]] && { code=$EX_USAGE die "Usage: workmode trigger enable <name>"; }

    local type
    type="$(config_trigger_field "$trigger_name" "type" 2>/dev/null)" || {
        code=$EX_NOT_FOUND die "Trigger '$trigger_name' not found"
    }

    local unit_name="${UNIT_PREFIX}${trigger_name}"

    if [[ "$type" == "timer" ]]; then
        systemctl --user enable "${unit_name}.timer" 2>/dev/null || true
        systemctl --user start "${unit_name}.timer" 2>/dev/null || true
        echo "Enabled timer: $trigger_name"
    else
        echo "Only timer triggers can be individually enabled/disabled."
        echo "File triggers are managed by the watcher service."
    fi
}

cmd_trigger_disable() {
    local trigger_name="${1:-}"
    [[ -z "$trigger_name" ]] && { code=$EX_USAGE die "Usage: workmode trigger disable <name>"; }

    local type
    type="$(config_trigger_field "$trigger_name" "type" 2>/dev/null)" || {
        code=$EX_NOT_FOUND die "Trigger '$trigger_name' not found"
    }

    local unit_name="${UNIT_PREFIX}${trigger_name}"

    if [[ "$type" == "timer" ]]; then
        systemctl --user stop "${unit_name}.timer" 2>/dev/null || true
        systemctl --user disable "${unit_name}.timer" 2>/dev/null || true
        echo "Disabled timer: $trigger_name"
    else
        echo "Only timer triggers can be individually enabled/disabled."
        echo "File triggers are managed by the watcher service."
    fi
}

# --- Helpers ---

_trigger_schedule_info() {
    local name="$1" type="$2"
    if [[ "$type" == "timer" ]]; then
        local cron_expr
        cron_expr="$(config_trigger_field "$name" "cron" || true)"
        if [[ -n "$cron_expr" ]]; then
            echo "cron: $cron_expr"
        else
            local interval
            interval="$(config_trigger_field "$name" "interval" || echo "?")"
            echo "every $interval"
        fi
    elif [[ "$type" == "file" ]]; then
        local watch pattern
        watch="$(config_trigger_field "$name" "watch" || echo "?")"
        pattern="$(config_trigger_field "$name" "pattern" || echo "*")"
        echo "$watch ($pattern)"
    fi
}

_trigger_to_json() {
    local name="$1"
    local type skill prompt_text permissions working_dir cooldown check

    type="$(config_trigger_field "$name" "type" || echo "")"
    skill="$(config_trigger_field "$name" "skill" || true)"
    prompt_text="$(config_trigger_field "$name" "prompt" || true)"
    permissions="$(config_trigger_field "$name" "permissions" || echo "default")"
    working_dir="$(config_trigger_field "$name" "working_dir" || true)"
    cooldown="$(config_trigger_field "$name" "cooldown" || true)"
    check="$(config_trigger_field "$name" "check" || true)"

    local fields
    fields="$(json_field "name" "$name"),$(json_field "type" "$type"),$(json_field "permissions" "$permissions")"
    [[ -n "$skill" ]] && fields+=",$(json_field "skill" "$skill")"
    [[ -n "$prompt_text" ]] && fields+=",$(json_field "prompt" "$prompt_text")"
    [[ -n "$working_dir" ]] && fields+=",$(json_field "working_dir" "$working_dir")"
    [[ -n "$cooldown" ]] && fields+=",$(json_field_num "cooldown" "$cooldown")"
    [[ -n "$check" ]] && fields+=",$(json_field "check" "$check")"

    if [[ "$type" == "timer" ]]; then
        local interval cron_expr
        interval="$(config_trigger_field "$name" "interval" || true)"
        cron_expr="$(config_trigger_field "$name" "cron" || true)"
        [[ -n "$interval" ]] && fields+=",$(json_field "interval" "$interval")"
        [[ -n "$cron_expr" ]] && fields+=",$(json_field "cron" "$cron_expr")"
    elif [[ "$type" == "file" ]]; then
        local watch pattern settle
        watch="$(config_trigger_field "$name" "watch" || true)"
        pattern="$(config_trigger_field "$name" "pattern" || true)"
        settle="$(config_trigger_field "$name" "settle" || true)"
        [[ -n "$watch" ]] && fields+=",$(json_field "watch" "$watch")"
        [[ -n "$pattern" ]] && fields+=",$(json_field "pattern" "$pattern")"
        [[ -n "$settle" ]] && fields+=",$(json_field_num "settle" "$settle")"
    fi

    local retry retry_max retry_delay
    retry="$(config_trigger_field "$name" "retry" || true)"
    retry_max="$(config_trigger_field "$name" "retry_max" || true)"
    retry_delay="$(config_trigger_field "$name" "retry_delay" || true)"
    [[ -n "$retry" ]] && fields+=",$(json_field "retry" "$retry")"
    [[ -n "$retry_max" ]] && fields+=",$(json_field_num "retry_max" "$retry_max")"
    [[ -n "$retry_delay" ]] && fields+=",$(json_field_num "retry_delay" "$retry_delay")"

    json_object "$fields"
    echo
}
