#!/usr/bin/env bash
# notify.sh — Desktop notification wrapper for workmode

NOTIFY_APP="workmode"

notify_started() {
    local skill="$1"
    local trigger="$2"
    notify-send -a "$NOTIFY_APP" -u low "workmode" "🤖 Running ${skill}..."
}

notify_completed() {
    local skill="$1"
    local trigger="$2"
    local duration="$3"
    local human_duration
    human_duration="$(format_duration "$duration")"
    notify-send -a "$NOTIFY_APP" -u normal "workmode" "✅ ${skill} done (${human_duration})"
}

notify_stuck() {
    local skill="$1"
    local trigger="$2"
    local session_id="$3"
    notify-send -a "$NOTIFY_APP" -u critical "workmode" "⏸ ${skill} needs permission — run: workmode session resume ${session_id}"
}

notify_error() {
    local skill="$1"
    local trigger="$2"
    local session_id="$3"
    notify-send -a "$NOTIFY_APP" -u critical "workmode" "❌ ${skill} failed — run: workmode session logs ${session_id}"
}

format_duration() {
    local seconds="$1"
    if (( seconds < 60 )); then
        echo "${seconds}s"
    elif (( seconds < 3600 )); then
        echo "$(( seconds / 60 ))m"
    else
        echo "$(( seconds / 3600 ))h$(( (seconds % 3600) / 60 ))m"
    fi
}
