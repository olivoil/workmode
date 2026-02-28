package backend

import "time"

// Status represents the output of `workmode status --json`.
type Status struct {
	Active  bool `json:"active"`
	Watcher bool `json:"watcher"`
	Timers  int  `json:"timers"`
	// Triggers is the total number of configured triggers.
	Triggers int `json:"triggers"`
	// Running is the count of currently running sessions (derived from session data).
	Running int `json:"-"`
	// Today is the count of sessions started today (derived from session data).
	Today int `json:"-"`
}

// Session represents a session entry from history.jsonl or `workmode session list --json`.
type Session struct {
	ID         string `json:"id"`
	Short      string `json:"short"`
	Trigger    string `json:"trigger"`
	Label      string `json:"label"`
	WorkingDir string `json:"working_dir"`
	Started    string `json:"started"`
	Status     string `json:"status"`
	Duration   int    `json:"duration,omitempty"`
	PID        int    `json:"pid,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	File       string `json:"file,omitempty"`
	Attempt    int    `json:"attempt,omitempty"`
	ExitCode   int    `json:"exit_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

// StartedTime parses the Started field as time.Time.
func (s Session) StartedTime() time.Time {
	t, err := time.Parse(time.RFC3339, s.Started)
	if err != nil {
		return time.Time{}
	}
	return t
}

// Trigger represents a trigger from `workmode trigger list --json`.
type Trigger struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Permissions string `json:"permissions,omitempty"`
	Skill       string `json:"skill,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
	WorkingDir  string `json:"working_dir,omitempty"`
	Cooldown    int    `json:"cooldown,omitempty"`
	Check       string `json:"check,omitempty"`

	// Timer-specific
	Interval string `json:"interval,omitempty"`
	Cron     string `json:"cron,omitempty"`

	// File-specific
	Watch   string `json:"watch,omitempty"`
	Pattern string `json:"pattern,omitempty"`
	Settle  int    `json:"settle,omitempty"`

	// Retry
	Retry      string `json:"retry,omitempty"`
	RetryMax   int    `json:"retry_max,omitempty"`
	RetryDelay int    `json:"retry_delay,omitempty"`
}

// Schedule returns a human-readable schedule string for the trigger.
func (t Trigger) Schedule() string {
	switch t.Type {
	case "timer":
		if t.Cron != "" {
			return t.Cron
		}
		return t.Interval
	case "file":
		s := t.Watch
		if t.Pattern != "" {
			s += " (" + t.Pattern + ")"
		}
		return s
	}
	return ""
}

// Config represents the output of `workmode config show --json`.
type Config struct {
	General struct {
		StateDir    string `json:"state_dir"`
		MaxParallel int    `json:"max_parallel"`
	} `json:"general"`
	Triggers []Trigger `json:"triggers"`
}

// StreamEvent represents a single line from Claude's stream-json output.
type StreamEvent struct {
	Type    string       `json:"type"`
	Message *MessageBody `json:"message,omitempty"`
	Result  string       `json:"result,omitempty"`
	// tool_use fields (flattened in stream-json)
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`
}

// MessageBody is the message field inside an assistant StreamEvent.
type MessageBody struct {
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents a content block inside a message.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}
