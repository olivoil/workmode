#!/usr/bin/env bash
# config.sh — TOML parser and config reader for workmode
# Parses ~/.config/workmode/config.toml

WORKMODE_CONFIG="${WORKMODE_CONFIG:-$HOME/.config/workmode/config.toml}"

# Parse a value from [general] section
# Usage: config_general <key>
config_general() {
    local key="$1"
    local in_general=false
    local value=""

    while IFS= read -r line; do
        # Strip comments and whitespace
        line="${line%%#*}"
        line="$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
        [[ -z "$line" ]] && continue

        if [[ "$line" == "[general]" ]]; then
            in_general=true
            continue
        elif [[ "$line" == "["* ]]; then
            in_general=false
            continue
        fi

        if $in_general && [[ "$line" == "${key} "* || "$line" == "${key}="* ]]; then
            value="${line#*=}"
            value="$(echo "$value" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//;s/^"//;s/"$//')"
            # Expand ~ to HOME
            value="${value/#\~/$HOME}"
            echo "$value"
            return 0
        fi
    done < "$WORKMODE_CONFIG"

    return 1
}

# Get state directory from config or default
config_state_dir() {
    config_general "state_dir" 2>/dev/null || echo "$HOME/.local/share/workmode"
}

# Get max parallel from config or default
config_max_parallel() {
    config_general "max_parallel" 2>/dev/null || echo "2"
}

# List all trigger names
# Usage: config_list_triggers
config_list_triggers() {
    local in_trigger=false

    while IFS= read -r line; do
        line="${line%%#*}"
        line="$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
        [[ -z "$line" ]] && continue

        if [[ "$line" == "[[trigger]]" ]]; then
            in_trigger=true
            continue
        elif [[ "$line" == "["* ]]; then
            in_trigger=false
            continue
        fi

        if $in_trigger && [[ "$line" == "name "* || "$line" == "name="* ]]; then
            local val="${line#*=}"
            val="$(echo "$val" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//;s/^"//;s/"$//')"
            echo "$val"
            in_trigger=false
        fi
    done < "$WORKMODE_CONFIG"
}

# Get a field from a specific trigger block
# Usage: config_trigger_field <trigger_name> <field>
config_trigger_field() {
    local trigger_name="$1"
    local field="$2"
    local in_trigger=false
    local found_trigger=false

    while IFS= read -r line; do
        line="${line%%#*}"
        line="$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
        [[ -z "$line" ]] && continue

        if [[ "$line" == "[[trigger]]" ]]; then
            in_trigger=true
            found_trigger=false
            continue
        elif [[ "$line" == "["* ]]; then
            in_trigger=false
            found_trigger=false
            continue
        fi

        if $in_trigger; then
            if [[ "$line" == "name "* || "$line" == "name="* ]]; then
                local val="${line#*=}"
                val="$(echo "$val" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//;s/^"//;s/"$//')"
                if [[ "$val" == "$trigger_name" ]]; then
                    found_trigger=true
                fi
            fi

            if $found_trigger && [[ "$line" == "${field} "* || "$line" == "${field}="* ]]; then
                local val="${line#*=}"
                val="$(echo "$val" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//;s/^"//;s/"$//')"
                # Expand ~ to HOME
                val="${val/#\~/$HOME}"
                echo "$val"
                return 0
            fi
        fi
    done < "$WORKMODE_CONFIG"

    return 1
}

# Get all fields for a trigger as associative-array-compatible output
# Usage: eval "$(config_trigger <trigger_name>)"
# Sets: TRIGGER_name, TRIGGER_type, TRIGGER_skill, TRIGGER_permissions, etc.
config_trigger() {
    local trigger_name="$1"
    local in_trigger=false
    local found_trigger=false

    while IFS= read -r line; do
        line="${line%%#*}"
        line="$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
        [[ -z "$line" ]] && continue

        if [[ "$line" == "[[trigger]]" ]]; then
            if $found_trigger; then
                return 0
            fi
            in_trigger=true
            found_trigger=false
            continue
        elif [[ "$line" == "["* ]]; then
            if $found_trigger; then
                return 0
            fi
            in_trigger=false
            continue
        fi

        if $in_trigger; then
            local key="${line%%=*}"
            key="$(echo "$key" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
            local val="${line#*=}"
            val="$(echo "$val" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//;s/^"//;s/"$//')"
            val="${val/#\~/$HOME}"

            if [[ "$key" == "name" && "$val" == "$trigger_name" ]]; then
                found_trigger=true
                echo "TRIGGER_name=$(printf '%q' "$val")"
            elif $found_trigger; then
                echo "TRIGGER_${key}=$(printf '%q' "$val")"
            fi
        fi
    done < "$WORKMODE_CONFIG"

    $found_trigger && return 0 || return 1
}

# List triggers of a specific type
# Usage: config_triggers_by_type "timer" or "file"
config_triggers_by_type() {
    local target_type="$1"
    for name in $(config_list_triggers); do
        local type
        type="$(config_trigger_field "$name" "type")"
        if [[ "$type" == "$target_type" ]]; then
            echo "$name"
        fi
    done
}

# Convert interval string (e.g., "2h", "15m", "30s") to minutes
interval_to_minutes() {
    local interval="$1"
    local num="${interval%[hms]*}"
    local unit="${interval##*[0-9]}"

    case "$unit" in
        h) echo $(( num * 60 )) ;;
        m) echo "$num" ;;
        s) echo $(( (num + 59) / 60 )) ;;  # round up
        *) echo "$num" ;;  # assume minutes
    esac
}

# Convert interval to cron schedule
# "2h" → "0 */2 * * *", "15m" → "*/15 * * * *"
interval_to_cron() {
    local interval="$1"
    local num="${interval%[hms]*}"
    local unit="${interval##*[0-9]}"

    case "$unit" in
        h) echo "0 */${num} * * *" ;;
        m) echo "*/${num} * * * *" ;;
        *) echo "*/${num} * * * *" ;;
    esac
}
