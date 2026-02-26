# workmode

Run [Claude Code](https://docs.anthropic.com/en/docs/claude-code) prompts automatically — on a schedule, or when files change.

Define triggers in a config file. Workmode turns them into systemd timers and file watchers that run `claude -p` in the background, track every session, and send desktop notifications so you know what's happening.

## What you can do with it

- Run a prompt every 2 hours to tidy up your notes
- Watch a folder and process new files as they appear (e.g. transcribe recordings, resize images, convert formats)
- Poll an API on a schedule and only run Claude when there's work to do (e.g. new PRs to review)
- Fire any Claude Code slash command or custom prompt, with configurable permission levels
- Monitor running sessions, view their logs, and resume any session interactively

## Platform support

Currently **Linux only** — requires `systemd` for timers and `inotifywait` for file watching. Tested on Arch Linux / [omarchy](https://omarchy.com). macOS and other platforms are not yet supported.

## How it works

```
config.toml → workmode install → systemd timers + inotifywait watcher
                                        ↓
                              workmode-run --trigger {name}
                                        ↓
                          claude -p "{prompt}" [--dangerously-skip-permissions]
                                        ↓
                              session log (JSONL) + desktop notification
```

1. **Config** (`~/.config/workmode/config.toml`) defines triggers
2. **`workmode install`** reads config, creates systemd user timers and a file watcher service
3. When a trigger fires, **`workmode-run`** executes the configured prompt via `claude -p`
4. Each execution is logged to `~/.local/share/workmode/history.jsonl` with session ID, status, duration
5. Desktop notifications (via `notify-send`) keep you aware of what's running

## Install

```bash
git clone https://github.com/olivoil/workmode.git
cd workmode
ln -sf "$(pwd)/bin/workmode" ~/.local/bin/workmode
cp config.toml.example ~/.config/workmode/config.toml
# Edit config.toml to your needs, then:
workmode on
```

### Requirements

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) CLI (`claude`)
- `systemd` (user timers for scheduled triggers)
- `inotifywait` (from `inotify-tools`, for file triggers)
- `notify-send` (for desktop notifications)
- bash 4+

## Configuration

`~/.config/workmode/config.toml`:

```toml
[general]
state_dir = "~/.local/share/workmode"
max_parallel = 2

# Run a prompt on a schedule
[[trigger]]
name = "tidy-notes"
type = "timer"
interval = "2h"
prompt = "Review today's notes and fix formatting, spelling, and broken links."
permissions = "skip"
working_dir = "~/notes"

# Run a Claude Code skill on a schedule, with a pre-check
[[trigger]]
name = "pr-reviews"
type = "timer"
interval = "15m"
check = "gh api /user/requested_reviews --jq 'length'"
prompt = "Check my open PR review requests and prepare a summary of each."
permissions = "default"
working_dir = "~/code/my-project"

# React to new files in a directory
[[trigger]]
name = "process-recordings"
type = "file"
watch = "~/Videos"
pattern = "*.mp4"
cooldown = 120
prompt = "Transcribe the meeting recording at $FILE and save a summary."
permissions = "skip"
working_dir = "~/notes"

# Use an explicit cron schedule (weekdays at 8:45am)
# [[trigger]]
# name = "standup-prep"
# type = "timer"
# cron = "45 8 * * 1-5"
# prompt = "Look at my git commits from yesterday and prepare a standup summary."
# permissions = "skip"
# working_dir = "~/code/my-project"
```

Each trigger uses either `prompt` (any text you'd send to Claude) or `skill` (a Claude Code slash command like `/commit`).

### Trigger fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Unique trigger name (used in short session IDs) |
| `type` | yes | `timer` or `file` |
| `prompt` | one of | Any text prompt for Claude |
| `skill` | one of | Claude Code slash command (e.g. `/commit`) |
| `permissions` | no | `skip`, `default`, or `readonly` (default: `default`) |
| `working_dir` | no | Directory to run Claude in |
| `interval` | timer | Repeat interval: `15m`, `2h`, etc. |
| `cron` | timer | 5-field cron expression (alternative to `interval`) |
| `check` | timer | Shell command — trigger skipped if it returns 0 or empty |
| `watch` | file | Directory to watch for new files |
| `pattern` | file | Glob pattern to match filenames |
| `cooldown` | file | Minimum seconds between runs |

### Permission modes

| Setting | Behavior |
|---------|----------|
| `permissions = "skip"` | Fully autonomous — `--dangerously-skip-permissions` |
| `permissions = "default"` | Normal permissions — may block if approval needed |
| `permissions = "readonly"` | Read-only tool access |

When a session blocks on permissions, it's marked **stuck** and a notification is sent so you can resume it interactively.

## Usage

```bash
workmode on                # activate all triggers
workmode off               # deactivate all triggers
workmode status            # show state + trigger list
workmode triggers          # list configured triggers
workmode run <trigger>     # manually fire a trigger

workmode sessions          # list recent sessions (last 20)
workmode sessions --stuck  # show stuck sessions
workmode logs <id>         # view session output
workmode tail <id>         # follow a running session
workmode resume <id>       # jump into a session with claude --resume
```

### Session IDs

Sessions get short memorable IDs like `tidy-notes-a3f0` that you can use everywhere:

```
$ workmode sessions
STATUS     TRIGGER          LABEL                  STARTED        DURATION   ID
✅ done    tidy-notes       Review today's note…   14:00 today    3m         tidy-notes-a3f0
⏸ stuck    pr-reviews       Check my open PR re…   14:15 today    —          pr-reviews-b2c1
🔄 run     process-recor…   Transcribe the meet…   14:30 today    2m         process-recordings-d4e5
```

## Architecture

```
bin/
  workmode              Main CLI (on, off, status, run, sessions, logs, tail, resume)
  workmode-run          Core runner: dedup, cooldown, max_parallel, claude -p, JSONL log, notify
  workmode-install      Config → systemd timers + inotifywait watcher service
  workmode-sessions     Session list, logs, tail, resume
lib/
  config.sh             TOML parser
  notify.sh             Desktop notification wrapper (notify-send)
```

State is stored in `~/.local/share/workmode/`:
- `history.jsonl` — session history (append-only)
- `logs/` — per-session output logs
- `locks/` — dedup lock files
- `state` — on/off flag

## License

MIT
