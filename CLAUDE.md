# Workmode — Claude Code Context

Automated Claude Code skill runner. Triggers run Claude skills or prompts on timers or file events.

## Repository Structure

```
bin/
  workmode              # Main CLI entrypoint (subcommands: on, off, status, run, sessions, logs, tail, resume, triggers)
  workmode-run          # Core runner — dedup, cooldown, max_parallel, runs claude -p, logs JSONL, notifies
  workmode-install      # Reads config → generates systemd user timers + inotifywait watcher service
  workmode-sessions     # Session list/logs/tail/resume CLI
lib/
  config.sh             # TOML parser for ~/.config/workmode/config.toml
  notify.sh             # Desktop notification wrapper (notify-send)
systemd/
  workmode-watcher.service.template  # Reference template for the file watcher service
config.toml.example     # Example configuration
```

## Key Design Decisions

- **Pure bash** — no external dependencies beyond Claude CLI, systemd, inotifywait, notify-send
- **TOML config** parsed with a custom bash parser in `lib/config.sh` (no `dasel` or other tools)
- **systemd user timers** instead of crontab (Arch Linux doesn't ship cron)
- **`claude -p`** (print mode) for non-interactive execution; sessions are persisted and resumable
- **`--resume` is project-scoped** — must run from the same `working_dir`, so we track it in JSONL
- **Short session IDs** like `refine-3e97` (trigger name + 4 hex chars) for human-friendly references
- **ANSI colors** auto-disabled when stdout is piped (TTY detection)
- **`env -u CLAUDECODE`** needed to launch claude from within another claude session

## State & Logs

All state lives in `~/.local/share/workmode/` (configurable via `state_dir` in config):
- `history.jsonl` — append-only session log, one JSON object per line per status change
- `logs/<session-id>.log` — raw stream-json output from each claude run
- `locks/<trigger>.lock` — PID-based dedup locks (auto-cleaned on stale)
- `state` — plain text "active" or "inactive"

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

Then: `workmode run test`
