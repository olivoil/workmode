package triggers

import (
	"fmt"
	"strings"

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

// Model is the triggers view.
type Model struct {
	table    table.Model
	preview  viewport.Model
	triggers []backend.Trigger
	sessions []backend.Session // all sessions, for showing recent per trigger
	width    int
	height   int
	focused  bool
}

// New creates a new triggers view model.
func New() Model {
	cols := []table.Column{
		{Title: "name", Width: 16},
		{Title: "type", Width: 6},
		{Title: "schedule", Width: 24},
		{Title: "permissions", Width: 10},
		{Title: "label", Width: 20},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(false),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ui.ColorBorder)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#ffffff")).
		Background(ui.ColorAccent).
		Bold(true)
	t.SetStyles(s)

	vp := viewport.New(viewport.WithWidth(40), viewport.WithHeight(10))

	return Model{
		table:   t,
		preview: vp,
	}
}

// SetTriggers updates the trigger data.
func (m *Model) SetTriggers(triggers []backend.Trigger) {
	m.triggers = triggers
	rows := make([]table.Row, len(triggers))
	for i, t := range triggers {
		label := t.Skill
		if label == "" && t.Prompt != "" {
			label = truncate(t.Prompt, 20)
		}
		rows[i] = table.Row{
			t.Name,
			t.Type,
			t.Schedule(),
			t.Permissions,
			label,
		}
	}
	m.table.SetRows(rows)
	m.updatePreview()
}

// SetSessions stores session data for the recent-sessions preview.
func (m *Model) SetSessions(sessions []backend.Session) {
	m.sessions = sessions
	m.updatePreview()
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
}

// SelectedTrigger returns the currently selected trigger, if any.
func (m *Model) SelectedTrigger() *backend.Trigger {
	idx := m.table.Cursor()
	if idx >= 0 && idx < len(m.triggers) {
		return &m.triggers[idx]
	}
	return nil
}

// Focus sets focus on the triggers table.
func (m *Model) Focus() {
	m.focused = true
	m.table.Focus()
}

// Blur removes focus from the triggers table.
func (m *Model) Blur() {
	m.focused = false
	m.table.Blur()
}

// Update handles messages for the triggers view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	prev := m.table.Cursor()
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	if m.table.Cursor() != prev {
		m.updatePreview()
	}
	return m, cmd
}

// View renders the triggers view.
func (m Model) View() string {
	tableView := m.table.View()
	previewStyle := ui.StylePreviewBorder.Height(m.height)
	previewView := previewStyle.Render(m.preview.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, tableView, previewView)
}

func (m *Model) updatePreview() {
	trig := m.SelectedTrigger()
	if trig == nil {
		m.preview.SetContent(ui.StyleDim.Render("No trigger selected"))
		return
	}

	var b strings.Builder

	b.WriteString(ui.StyleAccent.Render("Trigger: ") + trig.Name + "\n")
	b.WriteString(ui.StyleDim.Render("Type:    ") + trig.Type + "\n")
	b.WriteString(ui.StyleDim.Render("Schedule:") + " " + trig.Schedule() + "\n")
	if trig.Permissions != "" {
		b.WriteString(ui.StyleDim.Render("Perms:   ") + trig.Permissions + "\n")
	}
	if trig.WorkingDir != "" {
		b.WriteString(ui.StyleDim.Render("Dir:     ") + trig.WorkingDir + "\n")
	}
	if trig.Skill != "" {
		b.WriteString(ui.StyleDim.Render("Skill:   ") + trig.Skill + "\n")
	}
	if trig.Prompt != "" {
		b.WriteString(ui.StyleDim.Render("Prompt:  ") + truncate(trig.Prompt, 60) + "\n")
	}
	if trig.Cooldown > 0 {
		b.WriteString(ui.StyleDim.Render("Cooldown:") + fmt.Sprintf(" %ds", trig.Cooldown) + "\n")
	}
	if trig.Check != "" {
		b.WriteString(ui.StyleDim.Render("Check:   ") + trig.Check + "\n")
	}
	if trig.Retry != "" && trig.Retry != "never" {
		retry := trig.Retry
		if trig.RetryMax > 0 {
			retry += fmt.Sprintf(" (max %d", trig.RetryMax)
			if trig.RetryDelay > 0 {
				retry += fmt.Sprintf(", delay %ds", trig.RetryDelay)
			}
			retry += ")"
		}
		b.WriteString(ui.StyleDim.Render("Retry:   ") + retry + "\n")
	}

	// Recent sessions for this trigger.
	b.WriteString("\n" + ui.StyleDim.Render("─── Recent sessions ───") + "\n\n")

	count := 0
	for _, s := range m.sessions {
		if s.Trigger != trig.Name {
			continue
		}
		icon := ui.StatusIcon(s.Status)
		dur := ui.FormatDuration(s.Duration)
		b.WriteString(fmt.Sprintf("%s  %s  %s  %s\n", icon, ui.FormatTime(s.Started), dur, s.Short))
		count++
		if count >= 5 {
			break
		}
	}
	if count == 0 {
		b.WriteString(ui.StyleDim.Render("(no sessions)"))
	}

	m.preview.SetContent(b.String())
	m.preview.GotoTop()
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen-1] + "…"
	}
	return s
}
