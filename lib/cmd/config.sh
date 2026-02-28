#!/usr/bin/env bash
# lib/cmd/config.sh — Config management commands

dispatch_config() {
    local subcmd="${1:-show}"
    shift || true

    case "$subcmd" in
        show)     cmd_config_show "$@" ;;
        edit)     cmd_config_edit "$@" ;;
        validate) cmd_config_validate "$@" ;;
        apply)    cmd_config_apply "$@" ;;
        path)     cmd_config_path "$@" ;;
        help|--help|-h) usage_config ;;
        *)        die "Unknown config command: $subcmd" ;;
    esac
}

usage_config() {
    cat <<EOF
Usage: workmode config <command> [options]

Commands:
  show [--json]    Print parsed config
  edit             Open config in \$EDITOR
  validate         Check syntax and required fields
  apply            Validate + reinstall systemd units
  path             Print config file path

EOF
    exit 0
}

cmd_config_show() {
    parse_global_flags "$@"
    $SHOW_HELP && usage_config

    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        config_to_json
        return
    fi

    echo "Config: $WORKMODE_CONFIG"
    echo ""

    # General section
    echo "General:"
    local state_dir max_parallel
    state_dir="$(config_state_dir)"
    max_parallel="$(config_max_parallel)"
    echo "  state_dir:    $state_dir"
    echo "  max_parallel: $max_parallel"
    echo ""

    # Triggers
    echo "Triggers:"
    for name in $(config_list_triggers); do
        local type
        type="$(config_trigger_field "$name" "type" || echo "?")"
        echo "  - $name ($type)"
    done
}

cmd_config_edit() {
    exec "${EDITOR:-vi}" "$WORKMODE_CONFIG"
}

cmd_config_validate() {
    parse_global_flags "$@"
    $SHOW_HELP && { echo "Usage: workmode config validate [--json]"; exit 0; }

    local errors=() warnings=()

    # Check config file exists
    if [[ ! -f "$WORKMODE_CONFIG" ]]; then
        errors+=("Config file not found: $WORKMODE_CONFIG")
    else
        # Check each trigger has required fields
        for name in $(config_list_triggers); do
            local type
            type="$(config_trigger_field "$name" "type" 2>/dev/null || true)"

            if [[ -z "$type" ]]; then
                errors+=("Trigger '$name': missing required field 'type'")
            elif [[ "$type" != "timer" && "$type" != "file" ]]; then
                errors+=("Trigger '$name': invalid type '$type' (must be 'timer' or 'file')")
            fi

            # Must have either skill or prompt
            local skill prompt_text
            skill="$(config_trigger_field "$name" "skill" 2>/dev/null || true)"
            prompt_text="$(config_trigger_field "$name" "prompt" 2>/dev/null || true)"
            if [[ -z "$skill" && -z "$prompt_text" ]]; then
                errors+=("Trigger '$name': must have either 'skill' or 'prompt'")
            fi

            # Timer-specific checks
            if [[ "$type" == "timer" ]]; then
                local interval cron_expr
                interval="$(config_trigger_field "$name" "interval" 2>/dev/null || true)"
                cron_expr="$(config_trigger_field "$name" "cron" 2>/dev/null || true)"
                if [[ -z "$interval" && -z "$cron_expr" ]]; then
                    errors+=("Trigger '$name': timer trigger needs 'interval' or 'cron'")
                fi
            fi

            # File-specific checks
            if [[ "$type" == "file" ]]; then
                local watch pattern
                watch="$(config_trigger_field "$name" "watch" 2>/dev/null || true)"
                if [[ -z "$watch" ]]; then
                    errors+=("Trigger '$name': file trigger needs 'watch' directory")
                elif [[ ! -d "${watch/#\~/$HOME}" ]]; then
                    warnings+=("Trigger '$name': watch directory does not exist: $watch")
                fi
                pattern="$(config_trigger_field "$name" "pattern" 2>/dev/null || true)"
                if [[ -z "$pattern" ]]; then
                    warnings+=("Trigger '$name': no 'pattern' set, will match all files")
                fi
            fi

            # Working dir check
            local working_dir
            working_dir="$(config_trigger_field "$name" "working_dir" 2>/dev/null || true)"
            if [[ -n "$working_dir" && ! -d "${working_dir/#\~/$HOME}" ]]; then
                warnings+=("Trigger '$name': working_dir does not exist: $working_dir")
            fi

            # Permissions check
            local permissions
            permissions="$(config_trigger_field "$name" "permissions" 2>/dev/null || true)"
            if [[ -n "$permissions" && "$permissions" != "default" && "$permissions" != "skip" && "$permissions" != "readonly" ]]; then
                errors+=("Trigger '$name': invalid permissions '$permissions' (must be 'default', 'skip', or 'readonly')")
            fi

            # Retry check
            local retry
            retry="$(config_trigger_field "$name" "retry" 2>/dev/null || true)"
            if [[ -n "$retry" && "$retry" != "never" && "$retry" != "on_error" && "$retry" != "always" ]]; then
                errors+=("Trigger '$name': invalid retry '$retry' (must be 'never', 'on_error', or 'always')")
            fi
        done
    fi

    local valid=true
    (( ${#errors[@]} > 0 )) && valid=false

    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        local err_items="" warn_items=""
        for e in "${errors[@]+"${errors[@]}"}"; do
            [[ -n "$err_items" ]] && err_items+=","
            err_items+="\"$(json_escape "$e")\""
        done
        for w in "${warnings[@]+"${warnings[@]}"}"; do
            [[ -n "$warn_items" ]] && warn_items+=","
            warn_items+="\"$(json_escape "$w")\""
        done
        printf '{"valid":%s,"errors":[%s],"warnings":[%s]}\n' "$valid" "$err_items" "$warn_items"
    else
        if $valid && (( ${#warnings[@]} == 0 )); then
            echo "Config is valid."
        else
            for e in "${errors[@]+"${errors[@]}"}"; do
                printf "  ERROR: %s\n" "$e"
            done
            for w in "${warnings[@]+"${warnings[@]}"}"; do
                printf "  WARNING: %s\n" "$w"
            done
            echo ""
            if $valid; then
                echo "Config is valid (with warnings)."
            else
                echo "Config has errors."
            fi
        fi
    fi

    $valid || exit $EX_CONFIG
}

cmd_config_apply() {
    # Validate first
    echo "Validating config..."
    cmd_config_validate || exit $?

    echo ""

    # Reinstall systemd units
    "$BIN_DIR/workmode-install" install

    # If workmode is active, reload the changed units
    local active=false
    if systemctl --user is-active "$WATCHER_SERVICE" &>/dev/null; then
        active=true
    fi
    for unit_file in "$HOME/.config/systemd/user"/${UNIT_PREFIX}*.timer; do
        [[ -f "$unit_file" ]] || continue
        local unit_name
        unit_name="$(basename "$unit_file")"
        if systemctl --user is-enabled --quiet "$unit_name" 2>/dev/null; then
            active=true
            break
        fi
    done

    if $active; then
        echo ""
        echo "Reloading active units..."
        "$BIN_DIR/workmode-install" enable

        if systemctl --user is-active "$WATCHER_SERVICE" &>/dev/null; then
            systemctl --user restart "$WATCHER_SERVICE" 2>/dev/null || true
        fi
    fi

    echo ""
    echo "Config applied."
}

cmd_config_path() {
    echo "$WORKMODE_CONFIG"
}
