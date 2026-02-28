package app

import "github.com/olivoil/workmode/tui/internal/backend"

// StatusLoadedMsg is sent when status data is fetched.
type StatusLoadedMsg struct {
	Status backend.Status
	Err    error
}

// SessionsLoadedMsg is sent when session data is loaded (from file or CLI).
type SessionsLoadedMsg struct {
	Sessions []backend.Session
	Err      error
}

// TriggersLoadedMsg is sent when trigger data is fetched.
type TriggersLoadedMsg struct {
	Triggers []backend.Trigger
	Err      error
}

// LogLoadedMsg is sent when a session's log is loaded.
type LogLoadedMsg struct {
	ShortID string
	Events  []backend.StreamEvent
	Err     error
}

// ActionResultMsg is sent when a CLI action completes.
type ActionResultMsg struct {
	Output string
	Err    error
}

// StatusTickMsg triggers a periodic status refresh.
type StatusTickMsg struct{}

// ResumeExitMsg is sent when a `claude --resume` process exits.
type ResumeExitMsg struct {
	Err error
}

// WatcherReadyMsg carries the watcher reference to the model.
type WatcherReadyMsg struct {
	Watcher *backend.Watcher
}
