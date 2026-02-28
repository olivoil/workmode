package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

)

// Client wraps the workmode CLI and direct file access.
type Client struct {
	bin        string // path or name of CLI binary
	appName    string
	stateDir   string // e.g. ~/.local/share/workmode
	configPath string
}

// NewClient creates a client. cliBinary is the CLI command name (e.g. "workmode").
// appName is used for default state dir (e.g. "workmode" → ~/.local/share/workmode).
func NewClient(cliBinary, appName string) *Client {
	c := &Client{
		bin:        cliBinary,
		appName:    appName,
		configPath: DefaultConfigPath(appName),
	}

	// Read state_dir directly from TOML config (no subprocess).
	if cfg, err := ReadConfigFile(c.configPath); err == nil && cfg.General.StateDir != "" {
		c.stateDir = expandHome(cfg.General.StateDir)
	}
	if c.stateDir == "" {
		home, _ := os.UserHomeDir()
		c.stateDir = filepath.Join(home, ".local", "share", appName)
	}
	return c
}

// StateDir returns the resolved state directory.
func (c *Client) StateDir() string { return c.stateDir }

// HistoryPath returns the path to history.jsonl.
func (c *Client) HistoryPath() string {
	return filepath.Join(c.stateDir, "history.jsonl")
}

// LogPath returns the path to a session's log file.
func (c *Client) LogPath(shortID string) string {
	return filepath.Join(c.stateDir, "logs", shortID+".log")
}

// --- Direct file access (fast, used for live updates) ---

// ReadSessions reads and deduplicates sessions from history.jsonl.
func (c *Client) ReadSessions() ([]Session, error) {
	return ParseSessionsFromFile(c.HistoryPath())
}

// ReadLog reads and parses a session's log file.
func (c *Client) ReadLog(shortID string) ([]StreamEvent, error) {
	return ParseLogFile(c.LogPath(shortID))
}

// --- Direct file access (config) ---

// ReadTriggers reads triggers directly from the TOML config file.
func (c *Client) ReadTriggers() ([]Trigger, error) {
	cfg, err := ReadConfigFile(c.configPath)
	if err != nil {
		return nil, err
	}
	return cfg.Triggers, nil
}

// --- CLI wrappers (for actions and systemd status) ---

// Status calls `workmode status --json`.
func (c *Client) Status() (Status, error) {
	out, err := c.run("status", "--json")
	if err != nil {
		return Status{}, err
	}
	var s Status
	if err := json.Unmarshal(out, &s); err != nil {
		return Status{}, fmt.Errorf("parse status: %w", err)
	}
	return s, nil
}

// Triggers calls `workmode trigger list --json`.
func (c *Client) Triggers() ([]Trigger, error) {
	out, err := c.run("trigger", "list", "--json")
	if err != nil {
		return nil, err
	}
	return parseNDJSON[Trigger](out)
}

// Sessions calls `workmode session list --json`.
func (c *Client) Sessions() ([]Session, error) {
	out, err := c.run("session", "list", "--json")
	if err != nil {
		return nil, err
	}
	return parseNDJSON[Session](out)
}

// Config calls `workmode config show --json`.
func (c *Client) Config() (Config, error) {
	out, err := c.run("config", "show", "--json")
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(out, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// --- Actions ---

// TriggerRun calls `workmode trigger run <name>`.
func (c *Client) TriggerRun(name string) ([]byte, error) {
	return c.run("trigger", "run", name)
}

// TriggerEnable calls `workmode trigger enable <name>`.
func (c *Client) TriggerEnable(name string) ([]byte, error) {
	return c.run("trigger", "enable", name)
}

// TriggerDisable calls `workmode trigger disable <name>`.
func (c *Client) TriggerDisable(name string) ([]byte, error) {
	return c.run("trigger", "disable", name)
}

// SessionStop calls `workmode session stop <id>`.
func (c *Client) SessionStop(id string) ([]byte, error) {
	return c.run("session", "stop", id)
}

// SessionKill calls `workmode session kill <id>`.
func (c *Client) SessionKill(id string) ([]byte, error) {
	return c.run("session", "kill", id)
}

// On calls `workmode on`.
func (c *Client) On() ([]byte, error) {
	return c.run("on")
}

// Off calls `workmode off`.
func (c *Client) Off() ([]byte, error) {
	return c.run("off")
}

// RunCommand executes an arbitrary workmode CLI command and returns combined output.
func (c *Client) RunCommand(args ...string) ([]byte, error) {
	return c.run(args...)
}

// ResumeCmd returns an *exec.Cmd for `claude --resume <sessionID>` in the correct working dir.
func (c *Client) ResumeCmd(s Session) *exec.Cmd {
	cmd := exec.Command("claude", "--resume", s.SessionID)
	dir := expandHome(s.WorkingDir)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "CLAUDECODE=") // unset CLAUDECODE
	return cmd
}

// --- internal ---

func (c *Client) run(args ...string) ([]byte, error) {
	cmd := exec.Command(c.bin, args...)
	cmd.Env = append(os.Environ(), "CLAUDECODE=") // unset CLAUDECODE for nested claude calls
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = stdout.String()
		}
		return nil, fmt.Errorf("%s %s: %w: %s", c.bin, strings.Join(args, " "), err, strings.TrimSpace(msg))
	}
	return stdout.Bytes(), nil
}

func parseNDJSON[T any](data []byte) ([]T, error) {
	var result []T
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var v T
		if err := json.Unmarshal(line, &v); err != nil {
			continue // skip malformed
		}
		result = append(result, v)
	}
	return result, nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// DeriveStats computes Running and Today counts from sessions and merges into Status.
func DeriveStats(status Status, sessions []Session) Status {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for _, s := range sessions {
		if s.Status == "running" {
			status.Running++
		}
		if t := s.StartedTime(); !t.IsZero() && t.After(todayStart) {
			status.Today++
		}
	}
	return status
}
