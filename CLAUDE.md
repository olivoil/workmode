# Workmode — Claude Code Context

Automated Claude Code skill runner. Triggers run Claude skills or prompts on timers or file events.

## Repository Structure

```
bin/
  workmode              # Main CLI entrypoint — two-level dispatcher
  workmode-run          # Core runner — dedup, cooldown, max_parallel, runs claude -p, logs JSONL, notifies
  workmode-install      # Reads config → generates systemd user timers + inotifywait watcher service + skill symlink
lib/
  cli.sh                # Shared framework: die, warn, exit codes, --json, JSON builders
  config.sh             # TOML parser for ~/.config/workmode/config.toml + config_to_json()
  notify.sh             # Desktop notification wrapper (notify-send)
  sessions.sh           # Shared session utilities (parse_json_field, format_session_log, etc.)
  cmd/
    trigger.sh          # trigger list/show/run/enable/disable
    session.sh          # session list/logs/tail/resume/stop/kill
    config.sh           # config show/edit/validate/apply/path
    completions.sh      # bash/zsh/fish completion generators
    tui.sh              # Interactive fzf browser
skill/
  SKILL.md              # Claude Code skill for managing workmode
config.toml.example     # Example configuration
```

## CLI Command Hierarchy

```
workmode on|off|status [--json]
workmode trigger list|show|run|enable|disable [--json]
workmode session list|logs|tail|resume|stop|kill [--json]
workmode config show|edit|validate|apply|path [--json]
workmode install|uninstall|tui|completions|help|version
```

## Key Design Decisions

- **Pure bash** — no external dependencies beyond Claude CLI, systemd, inotifywait, notify-send
- **Two-level dispatch** — `bin/workmode` routes to `lib/cmd/*.sh` modules
- **TOML config** parsed with a custom bash parser in `lib/config.sh` (no `dasel` or other tools)
- **systemd user timers** instead of crontab (Arch Linux doesn't ship cron)
- **`claude -p`** (print mode) for non-interactive execution; sessions are persisted and resumable
- **`--resume` is project-scoped** — must run from the same `working_dir`, so we track it in JSONL
- **Short session IDs** like `refine-3e97` (trigger name + 4 hex chars) for human-friendly references
- **`--json` output** — newline-delimited JSON objects for lists, single JSON for detail views
- **ANSI colors** auto-disabled when stdout is piped (TTY detection)
- **`env -u CLAUDECODE`** needed to launch claude from within another claude session
- **JSON function naming** — `lib/cli.sh` defines `json_field()` etc. for *building* JSON; `lib/sessions.sh` defines `parse_json_field()` for *extracting* from JSON strings. Command modules use `sess_json_field()` aliases.

## State & Logs

All state lives in `~/.local/share/workmode/` (configurable via `state_dir` in config):
- `history.jsonl` — append-only session log, one JSON object per line per status change
- `logs/<session-id>.log` — raw stream-json output from each claude run
- `locks/<trigger>.lock` — PID-based dedup locks (auto-cleaned on stale)

## Config Location

`~/.config/workmode/config.toml` — see `config.toml.example` for format.

## Testing

Use the `test` trigger (add to config) for quick validation:
```toml
[[trigger]]
name = "test"
type = "timer"
interval = "24h"
prompt = "Say 'Hello from workmode!' and nothing else."
permissions = "skip"
working_dir = "~/Code/github.com/olivoil/obsidian"
```

Then: `workmode trigger run test`
