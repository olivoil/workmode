package sessions

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/olivoil/workmode/tui/internal/backend"
	"github.com/olivoil/workmode/tui/internal/ui"
)

const (
	previewWidthFrac = 0.45
	minPreviewWidth  = 30
)

// Model is the sessions view.
type Model struct {
	table    table.Model
	preview  viewport.Model
	sessions []backend.Session
	width    int
	height   int
	focused  bool

	previewID     string
	previewEvents []backend.StreamEvent
}

// New creates a new sessions view model.
func New() Model {
	cols := []table.Column{
		{Title: " ", Width: 2},
		{Title: "trigger", Width: 14},
		{Title: "time", Width: 10},
		{Title: "dur", Width: 5},
		{Title: "id", Width: 18},
		{Title: "summary", Width: 30},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ui.ColorBorder)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(ui.T.Accent)).
		Bold(true)
	t.SetStyles(s)

	vp := viewport.New(viewport.WithWidth(40), viewport.WithHeight(10))

	return Model{
		table:   t,
		preview: vp,
		focused: true,
	}
}

// RefreshStyles reapplies theme colors to the table (called on theme change).
func (m *Model) RefreshStyles() {
	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ui.ColorBorder)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(ui.T.Accent)).
		Bold(true)
	m.table.SetStyles(s)
}

// SetSessions updates the session data and rebuilds the table rows.
func (m *Model) SetSessions(sessions []backend.Session) {
	m.sessions = sessions
	rows := make([]table.Row, len(sessions))
	for i, s := range sessions {
		rows[i] = table.Row{
			ui.StatusIcon(s.Status),
			s.Trigger,
			ui.FormatTime(s.Started),
			ui.FormatDuration(s.Duration),
			s.Short,
			"",
		}
	}
	m.table.SetRows(rows)
	// Show preview metadata for current selection immediately.
	if s := m.SelectedSession(); s != nil && m.previewID != s.Short {
		m.SetPreview(s.Short, nil)
	}
}

// SetPreview sets the preview content for a session.
func (m *Model) SetPreview(shortID string, events []backend.StreamEvent) {
	m.previewID = shortID
	m.previewEvents = events

	var sess *backend.Session
	var sessIdx int
	for i := range m.sessions {
		if m.sessions[i].Short == shortID {
			sess = &m.sessions[i]
			sessIdx = i
			break
		}
	}

	// Update summary in the table row if we have log events.
	if len(events) > 0 {
		summary := backend.ExtractSummary(events, 60)
		rows := m.table.Rows()
		if sessIdx < len(rows) && len(rows[sessIdx]) == 6 {
			rows[sessIdx][5] = summary
			m.table.SetRows(rows)
		}
	}

	content := m.renderPreview(sess, events)
	m.preview.SetContent(content)
	m.preview.GotoTop()
}

// SetSize updates the view dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h

	previewW := int(float64(w) * previewWidthFrac)
	if previewW < minPreviewWidth {
		previewW = minPreviewWidth
	}
	tableW := w - previewW - 3

	m.table.SetWidth(tableW)
	m.table.SetHeight(h)
	m.preview.SetWidth(previewW)
	m.preview.SetHeight(h)

	fixedW := 2 + 14 + 10 + 5 + 18 + 5
	summaryW := tableW - fixedW
	if summaryW < 10 {
		summaryW = 10
	}
	cols := m.table.Columns()
	if len(cols) == 6 {
		cols[5].Width = summaryW
		m.table.SetColumns(cols)
	}
}

// SelectedSession returns the currently selected session, if any.
func (m *Model) SelectedSession() *backend.Session {
	idx := m.table.Cursor()
	if idx >= 0 && idx < len(m.sessions) {
		return &m.sessions[idx]
	}
	return nil
}

// SelectedShortID returns the short ID of the selected session.
func (m *Model) SelectedShortID() string {
	if s := m.SelectedSession(); s != nil {
		return s.Short
	}
	return ""
}

// Focus sets focus on the sessions table.
func (m *Model) Focus() {
	m.focused = true
	m.table.Focus()
}

// Blur removes focus from the sessions table.
func (m *Model) Blur() {
	m.focused = false
	m.table.Blur()
}

// Update handles messages for the sessions view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the sessions view (table + preview side by side).
func (m Model) View() string {
	tableView := m.table.View()

	previewContent := m.preview.View()
	if previewContent == "" {
		previewContent = ui.StyleDim.Render("Select a session to preview")
	}
	previewStyle := ui.StylePreviewBorder.
		Width(m.previewWidth()).
		Height(m.height)
	previewView := previewStyle.Render(previewContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, tableView, previewView)
}

func (m *Model) previewWidth() int {
	pw := int(float64(m.width) * previewWidthFrac)
	if pw < minPreviewWidth {
		pw = minPreviewWidth
	}
	return pw
}

func (m *Model) renderPreview(sess *backend.Session, events []backend.StreamEvent) string {
	if sess == nil {
		return ui.StyleDim.Render("No session selected")
	}

	var s string

	s += ui.StyleAccent.Render("Session: ") + sess.Short + "\n"
	s += ui.StyleDim.Render("Full ID: ") + sess.ID + "\n"
	s += ui.StyleDim.Render("Status:  ") + sess.Status + "\n"
	s += ui.StyleDim.Render("Trigger: ") + sess.Trigger + "\n"
	if sess.Label != "" && sess.Label != sess.Trigger {
		s += ui.StyleDim.Render("Label:   ") + sess.Label + "\n"
	}
	s += ui.StyleDim.Render("Started: ") + ui.FormatTime(sess.Started) + "\n"
	if sess.Duration > 0 {
		s += ui.StyleDim.Render("Duration:") + " " + ui.FormatDuration(sess.Duration) + "\n"
	}
	s += ui.StyleDim.Render("Dir:     ") + sess.WorkingDir + "\n"
	if sess.SessionID != "" {
		s += ui.StyleDim.Render("Claude:  ") + sess.SessionID + "\n"
	}
	if sess.Error != "" {
		s += ui.StyleError.Render("Error:   "+sess.Error) + "\n"
	}

	s += "\n" + ui.StyleDim.Render("─── Log output ───") + "\n\n"

	if len(events) == 0 {
		s += ui.StyleDim.Render("(no log data)")
	} else {
		s += backend.FormatLogEvents(events)
	}

	return s
}
