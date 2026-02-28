#!/usr/bin/env bash
# lib/cmd/completions.sh — Shell completion generators

dispatch_completions() {
    local shell="${1:-}"
    case "$shell" in
        bash)  _completions_bash ;;
        zsh)   _completions_zsh ;;
        fish)  _completions_fish ;;
        help|--help|-h)
            echo "Usage: workmode completions bash|zsh|fish"
            exit 0
            ;;
        *)
            code=$EX_USAGE die "Usage: workmode completions bash|zsh|fish"
            ;;
    esac
}

_completions_bash() {
    cat <<'BASH_COMPLETIONS'
_workmode() {
    local cur prev words cword
    _init_completion || return

    local top_commands="on off status trigger session config install uninstall tui completions help version"
    local trigger_commands="list show run enable disable"
    local session_commands="list logs tail resume stop kill"
    local config_commands="show edit validate apply path"
    local completions_shells="bash zsh fish"

    case "${cword}" in
        1)
            COMPREPLY=( $(compgen -W "$top_commands" -- "$cur") )
            ;;
        2)
            case "${words[1]}" in
                trigger)
                    COMPREPLY=( $(compgen -W "$trigger_commands" -- "$cur") )
                    ;;
                session)
                    COMPREPLY=( $(compgen -W "$session_commands" -- "$cur") )
                    ;;
                config)
                    COMPREPLY=( $(compgen -W "$config_commands" -- "$cur") )
                    ;;
                completions)
                    COMPREPLY=( $(compgen -W "$completions_shells" -- "$cur") )
                    ;;
                status)
                    COMPREPLY=( $(compgen -W "--json" -- "$cur") )
                    ;;
            esac
            ;;
        3)
            case "${words[1]}" in
                trigger)
                    case "${words[2]}" in
                        show|run|enable|disable)
                            # Complete trigger names
                            local triggers
                            triggers="$(workmode trigger list --json 2>/dev/null | grep -oP '"name"\s*:\s*"[^"]*"' | grep -oP '"[^"]*"$' | tr -d '"')"
                            COMPREPLY=( $(compgen -W "$triggers" -- "$cur") )
                            ;;
                        list)
                            COMPREPLY=( $(compgen -W "--json" -- "$cur") )
                            ;;
                    esac
                    ;;
                session)
                    case "${words[2]}" in
                        logs|tail|resume|stop|kill)
                            # Complete session IDs
                            local sessions
                            sessions="$(workmode session list --json 2>/dev/null | grep -oP '"short"\s*:\s*"[^"]*"' | grep -oP '"[^"]*"$' | tr -d '"')"
                            COMPREPLY=( $(compgen -W "$sessions" -- "$cur") )
                            ;;
                        list)
                            COMPREPLY=( $(compgen -W "--json --running --stuck --completed --all" -- "$cur") )
                            ;;
                    esac
                    ;;
                config)
                    case "${words[2]}" in
                        show|validate)
                            COMPREPLY=( $(compgen -W "--json" -- "$cur") )
                            ;;
                    esac
                    ;;
            esac
            ;;
    esac
}

complete -F _workmode workmode
BASH_COMPLETIONS
}

_completions_zsh() {
    cat <<'ZSH_COMPLETIONS'
#compdef workmode

_workmode() {
    local -a top_commands trigger_commands session_commands config_commands

    top_commands=(
        'on:Activate all triggers'
        'off:Deactivate all triggers'
        'status:Show current state and summary'
        'trigger:Manage triggers'
        'session:Manage sessions'
        'config:Manage configuration'
        'install:Install systemd units'
        'uninstall:Remove systemd units'
        'tui:Interactive browser'
        'completions:Generate shell completions'
        'help:Show help'
        'version:Show version'
    )

    trigger_commands=(
        'list:List configured triggers'
        'show:Show trigger config'
        'run:Manually run a trigger'
        'enable:Enable a trigger systemd unit'
        'disable:Disable a trigger systemd unit'
    )

    session_commands=(
        'list:List recent sessions'
        'logs:Show session output'
        'tail:Follow running session'
        'resume:Resume interactive session'
        'stop:Graceful stop'
        'kill:Force kill'
    )

    config_commands=(
        'show:Print parsed config'
        'edit:Open in editor'
        'validate:Check syntax and fields'
        'apply:Validate and reinstall units'
        'path:Print config file path'
    )

    case "$words[2]" in
        trigger)
            if (( CURRENT == 3 )); then
                _describe 'trigger command' trigger_commands
            elif (( CURRENT == 4 )); then
                case "$words[3]" in
                    show|run|enable|disable)
                        local -a triggers
                        triggers=(${(f)"$(workmode trigger list --json 2>/dev/null | grep -oP '"name"\s*:\s*"[^"]*"' | grep -oP '"[^"]*"$' | tr -d '"')"})
                        _describe 'trigger name' triggers
                        ;;
                    list)
                        _arguments '--json[Output as JSON]'
                        ;;
                esac
            fi
            ;;
        session)
            if (( CURRENT == 3 )); then
                _describe 'session command' session_commands
            elif (( CURRENT == 4 )); then
                case "$words[3]" in
                    logs|tail|resume|stop|kill)
                        local -a sessions
                        sessions=(${(f)"$(workmode session list --json 2>/dev/null | grep -oP '"short"\s*:\s*"[^"]*"' | grep -oP '"[^"]*"$' | tr -d '"')"})
                        _describe 'session id' sessions
                        ;;
                    list)
                        _arguments \
                            '--json[Output as JSON]' \
                            '--running[Show running only]' \
                            '--stuck[Show stuck only]' \
                            '--completed[Show completed only]' \
                            '--all[Show all sessions]'
                        ;;
                esac
            fi
            ;;
        config)
            if (( CURRENT == 3 )); then
                _describe 'config command' config_commands
            elif (( CURRENT == 4 )); then
                case "$words[3]" in
                    show|validate)
                        _arguments '--json[Output as JSON]'
                        ;;
                esac
            fi
            ;;
        completions)
            if (( CURRENT == 3 )); then
                _describe 'shell' '(bash zsh fish)'
            fi
            ;;
        status)
            _arguments '--json[Output as JSON]'
            ;;
        *)
            if (( CURRENT == 2 )); then
                _describe 'workmode command' top_commands
            fi
            ;;
    esac
}

_workmode "$@"
ZSH_COMPLETIONS
}

_completions_fish() {
    cat <<'FISH_COMPLETIONS'
# Fish completions for workmode

# Disable file completions by default
complete -c workmode -f

# Top-level commands
complete -c workmode -n '__fish_use_subcommand' -a 'on' -d 'Activate all triggers'
complete -c workmode -n '__fish_use_subcommand' -a 'off' -d 'Deactivate all triggers'
complete -c workmode -n '__fish_use_subcommand' -a 'status' -d 'Show current state'
complete -c workmode -n '__fish_use_subcommand' -a 'trigger' -d 'Manage triggers'
complete -c workmode -n '__fish_use_subcommand' -a 'session' -d 'Manage sessions'
complete -c workmode -n '__fish_use_subcommand' -a 'config' -d 'Manage configuration'
complete -c workmode -n '__fish_use_subcommand' -a 'install' -d 'Install systemd units'
complete -c workmode -n '__fish_use_subcommand' -a 'uninstall' -d 'Remove systemd units'
complete -c workmode -n '__fish_use_subcommand' -a 'tui' -d 'Interactive browser'
complete -c workmode -n '__fish_use_subcommand' -a 'completions' -d 'Generate completions'
complete -c workmode -n '__fish_use_subcommand' -a 'help' -d 'Show help'
complete -c workmode -n '__fish_use_subcommand' -a 'version' -d 'Show version'

# status flags
complete -c workmode -n '__fish_seen_subcommand_from status' -l json -d 'JSON output'

# trigger subcommands
complete -c workmode -n '__fish_seen_subcommand_from trigger; and not __fish_seen_subcommand_from list show run enable disable' -a 'list' -d 'List triggers'
complete -c workmode -n '__fish_seen_subcommand_from trigger; and not __fish_seen_subcommand_from list show run enable disable' -a 'show' -d 'Show trigger config'
complete -c workmode -n '__fish_seen_subcommand_from trigger; and not __fish_seen_subcommand_from list show run enable disable' -a 'run' -d 'Run a trigger'
complete -c workmode -n '__fish_seen_subcommand_from trigger; and not __fish_seen_subcommand_from list show run enable disable' -a 'enable' -d 'Enable trigger'
complete -c workmode -n '__fish_seen_subcommand_from trigger; and not __fish_seen_subcommand_from list show run enable disable' -a 'disable' -d 'Disable trigger'

# session subcommands
complete -c workmode -n '__fish_seen_subcommand_from session; and not __fish_seen_subcommand_from list logs tail resume stop kill' -a 'list' -d 'List sessions'
complete -c workmode -n '__fish_seen_subcommand_from session; and not __fish_seen_subcommand_from list logs tail resume stop kill' -a 'logs' -d 'Show session output'
complete -c workmode -n '__fish_seen_subcommand_from session; and not __fish_seen_subcommand_from list logs tail resume stop kill' -a 'tail' -d 'Follow session'
complete -c workmode -n '__fish_seen_subcommand_from session; and not __fish_seen_subcommand_from list logs tail resume stop kill' -a 'resume' -d 'Resume session'
complete -c workmode -n '__fish_seen_subcommand_from session; and not __fish_seen_subcommand_from list logs tail resume stop kill' -a 'stop' -d 'Stop session'
complete -c workmode -n '__fish_seen_subcommand_from session; and not __fish_seen_subcommand_from list logs tail resume stop kill' -a 'kill' -d 'Kill session'

# config subcommands
complete -c workmode -n '__fish_seen_subcommand_from config; and not __fish_seen_subcommand_from show edit validate apply path' -a 'show' -d 'Show config'
complete -c workmode -n '__fish_seen_subcommand_from config; and not __fish_seen_subcommand_from show edit validate apply path' -a 'edit' -d 'Edit config'
complete -c workmode -n '__fish_seen_subcommand_from config; and not __fish_seen_subcommand_from show edit validate apply path' -a 'validate' -d 'Validate config'
complete -c workmode -n '__fish_seen_subcommand_from config; and not __fish_seen_subcommand_from show edit validate apply path' -a 'apply' -d 'Apply config'
complete -c workmode -n '__fish_seen_subcommand_from config; and not __fish_seen_subcommand_from show edit validate apply path' -a 'path' -d 'Show config path'

# completions subcommands
complete -c workmode -n '__fish_seen_subcommand_from completions' -a 'bash zsh fish'

# session list flags
complete -c workmode -n '__fish_seen_subcommand_from session; and __fish_seen_subcommand_from list' -l json -d 'JSON output'
complete -c workmode -n '__fish_seen_subcommand_from session; and __fish_seen_subcommand_from list' -l running -d 'Running only'
complete -c workmode -n '__fish_seen_subcommand_from session; and __fish_seen_subcommand_from list' -l stuck -d 'Stuck only'
complete -c workmode -n '__fish_seen_subcommand_from session; and __fish_seen_subcommand_from list' -l completed -d 'Completed only'
complete -c workmode -n '__fish_seen_subcommand_from session; and __fish_seen_subcommand_from list' -l all -d 'Show all'

# Dynamic trigger name completion
complete -c workmode -n '__fish_seen_subcommand_from trigger; and __fish_seen_subcommand_from show run enable disable' -a '(workmode trigger list --json 2>/dev/null | string match -r \'"name":"[^"]*"\' | string replace -r \'"name":"([^"]*)"\' \'$1\')'

# Dynamic session ID completion
complete -c workmode -n '__fish_seen_subcommand_from session; and __fish_seen_subcommand_from logs tail resume stop kill' -a '(workmode session list --json 2>/dev/null | string match -r \'"short":"[^"]*"\' | string replace -r \'"short":"([^"]*)"\' \'$1\')'
FISH_COMPLETIONS
}
