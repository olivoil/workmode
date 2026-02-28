# Workmode — Claude Code Skill

Manage workmode triggers, sessions, and configuration from Claude Code.

Workmode is an automated Claude Code skill runner that triggers Claude skills or prompts on timers or file events using systemd.

## Quick Reference

### Lifecycle
```bash
workmode on                    # Activate all triggers
workmode off                   # Deactivate all triggers
workmode status [--json]       # Show state + summary
```

### Triggers
```bash
workmode trigger list [--json]          # List all triggers
workmode trigger show <name> [--json]   # Show one trigger's config
workmode trigger run <name>             # Manually run a trigger
workmode trigger enable <name>          # Enable a single trigger
workmode trigger disable <name>         # Disable a single trigger
```

### Sessions
```bash
workmode session list [--json] [--running|--stuck|--completed]
workmode session logs <id>              # Show session output
workmode session tail <id>              # Follow running session
workmode session resume <id>            # Resume interactive session
workmode session stop <id>              # Graceful stop (SIGTERM)
workmode session kill <id>              # Force kill (SIGKILL)
```

### Config
```bash
workmode config show [--json]    # Print parsed config
workmode config edit             # Open in $EDITOR
workmode config validate         # Check syntax + required fields
workmode config apply            # Validate + reinstall systemd units
workmode config path             # Print config file path
```

### System
```bash
workmode install                 # Install/update systemd units
workmode uninstall               # Remove systemd units
```

## Config File Format

Location: `~/.config/workmode/config.toml`

Use `workmode config path` to get the exact path, then Read it.

### General section

```toml
[general]
state_dir = "~/.local/share/workmode"   # Where sessions/logs/locks live
max_parallel = 2                         # Max concurrent triggers
```

### Trigger blocks

Each trigger is a `[[trigger]]` block. Required fields: `name`, `type`, and either `skill` or `prompt`.

#### Timer trigger (runs on a schedule)

```toml
[[trigger]]
name = "refine"                          # Unique identifier
type = "timer"                           # "timer" or "file"
interval = "2h"                          # Repeat interval: Nh, Nm, Ns
# cron = "45 8 * * 1-5"                 # OR cron expression (5-field)
skill = "/refine"                        # Claude Code skill to run
# prompt = "Do something..."            # OR a prompt string
permissions = "skip"                     # "default", "skip", or "readonly"
working_dir = "~/Code/myproject"         # Working directory for claude
cooldown = 120                           # Min seconds between runs (optional)
check = "some-command"                   # Pre-check: skip if output is 0/empty (optional)
retry = "on_error"                       # "never" (default), "on_error", "always"
retry_max = 3                            # Max attempts (0 = unlimited)
retry_delay = 30                         # Seconds between retries
```

#### File trigger (runs when files change)

```toml
[[trigger]]
name = "transcribe"
type = "file"
watch = "~/Videos"                       # Directory to watch (recursive)
pattern = "screenrecording-*.mp4"        # Glob pattern to match
settle = 10                              # Wait N seconds for file to stabilize (optional)
skill = "/transcribe-meeting"
permissions = "skip"
working_dir = "~/Code/myproject"
```

### Field Reference

| Field | Required | Values | Default |
|-------|----------|--------|---------|
| `name` | yes | unique string | — |
| `type` | yes | `"timer"` or `"file"` | — |
| `skill` | one of skill/prompt | skill name (e.g. `"/refine"`) | — |
| `prompt` | one of skill/prompt | any text | — |
| `permissions` | no | `"default"`, `"skip"`, `"readonly"` | `"default"` |
| `working_dir` | no | path (~ expanded) | current dir |
| `interval` | timer only | `"Nh"`, `"Nm"`, `"Ns"` | — |
| `cron` | timer only | 5-field cron | — |
| `watch` | file only | directory path | — |
| `pattern` | file only | glob pattern | `"*"` |
| `settle` | file only | seconds (int) | `0` |
| `cooldown` | no | seconds (int) | `0` |
| `check` | no | shell command | — |
| `retry` | no | `"never"`, `"on_error"`, `"always"` | `"never"` |
| `retry_max` | no | int (0=unlimited) | `3` |
| `retry_delay` | no | seconds (int) | `30` |

## Workflows

### Reading config

```
1. workmode config path          → get file path
2. Read the file                 → understand current config
```

Or use `workmode config show --json` for structured output.

### Editing config

```
1. workmode config path          → get file path
2. Read the file                 → see current config
3. Edit the file                 → add/modify/remove [[trigger]] blocks
4. workmode config apply         → validate + reinstall (one step)
```

`config apply` validates the config (exits on error), reinstalls systemd units, and reloads active units if workmode is on.

### Adding a recurring automation

Add a `[[trigger]]` block with `type = "timer"`:

```toml
[[trigger]]
name = "daily-summary"
type = "timer"
interval = "24h"
prompt = "Generate a summary of today's work and save to daily note."
permissions = "skip"
working_dir = "~/Code/myproject"
```

Then run `workmode config apply`.

### Adding a file watcher

Add a `[[trigger]]` block with `type = "file"`:

```toml
[[trigger]]
name = "process-screenshots"
type = "file"
watch = "~/Screenshots"
pattern = "*.png"
prompt = "Analyze this screenshot and add notes to today's daily note: {file}"
permissions = "skip"
working_dir = "~/Code/myproject"
```

The `{file}` placeholder is replaced with the actual file path. Then run `workmode config apply`.

### Running a trigger manually

```bash
workmode trigger run <name>
```

### Checking session results

```bash
workmode session list                    # See recent sessions
workmode session list --running          # See what's running now
workmode session logs <id>               # See output of a session
```

### Self-disabling trigger

Include the disable command in the prompt so Claude turns off the trigger when done:

```toml
[[trigger]]
name = "one-time-task"
type = "timer"
interval = "15m"
prompt = "Check if the deployment is complete. If yes, run `workmode trigger disable one-time-task` and report the result."
permissions = "skip"
working_dir = "~/Code/myproject"
```

## State Directory

All state lives in `~/.local/share/workmode/` (configurable via `state_dir`):

- `history.jsonl` — append-only session log
- `logs/<session-id>.log` — raw stream-json output per session
- `locks/<trigger>.lock` — PID-based dedup locks
