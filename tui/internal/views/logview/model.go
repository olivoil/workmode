package logview

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/viewport"

	"github.com/olivoil/workmode/tui/internal/backend"
	"github.com/olivoil/workmode/tui/internal/ui"
)

// Model is the full-screen log viewer.
type Model struct {
	viewport viewport.Model
	session  *backend.Session
	width    int
	height   int
	active   bool
}

// New creates a new log view model.
func New() Model {
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(24))
	return Model{
		viewport: vp,
	}
}

// SetSize updates the view dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.SetWidth(w - 2)
	m.viewport.SetHeight(h)
}

// Show opens the log view for a session.
func (m *Model) Show(session *backend.Session, events []backend.StreamEvent) {
	m.session = session
	m.active = true
	m.setContent(session, events)
}

// UpdateLog refreshes the log content (for live tail).
func (m *Model) UpdateLog(events []backend.StreamEvent) {
	if m.session == nil {
		return
	}
	atBottom := m.viewport.AtBottom()
	m.setContent(m.session, events)
	if atBottom {
		m.viewport.GotoBottom()
	}
}

// Hide closes the log view.
func (m *Model) Hide() {
	m.active = false
	m.session = nil
}

// Active returns whether the log view is visible.
func (m *Model) Active() bool {
	return m.active
}

// Session returns the current session being viewed.
func (m *Model) Session() *backend.Session {
	return m.session
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the log view.
func (m Model) View() string {
	return m.viewport.View()
}

func (m *Model) setContent(sess *backend.Session, events []backend.StreamEvent) {
	var s string

	if sess != nil {
		s += ui.StyleAccent.Render(sess.Short)
		s += "  " + ui.StyleDim.Render(sess.Status)
		if sess.Duration > 0 {
			s += "  " + ui.FormatDuration(sess.Duration)
		}
		s += "\n"
		if sess.WorkingDir != "" {
			s += ui.StyleDim.Render(sess.WorkingDir) + "\n"
		}
		s += ui.StyleDim.Render("────────────────────────────────────────") + "\n\n"
	}

	if len(events) == 0 {
		s += ui.StyleDim.Render("(no log data)")
	} else {
		s += backend.FormatLogEvents(events)
	}

	m.viewport.SetContent(s)
}
