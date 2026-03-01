package command

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/olivoil/workmode/tui/internal/ui"
)

// ExecuteMsg is sent when a CLI command should be executed by the parent.
type ExecuteMsg struct {
	Args []string
}

// NLRequestMsg is sent when input should be routed to Claude as NL.
type NLRequestMsg struct {
	Input string
}

// Model is the command line + completion menu + result viewport.
type Model struct {
	input     textinput.Model
	result    viewport.Model
	completer *Completer
	focused   bool
	width     int
	height    int
	hasResult bool

	// Inline completion menu.
	candidates []Candidate
	selected   int // index into candidates, -1 = none

	// NL streaming state.
	streaming bool
	nlProc    *NLProcess
	nlBuffer  strings.Builder
	send      func(tea.Msg)
}

// New creates a new command model.
func New() Model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "type a command or ask a question..."
	ti.CharLimit = 256

	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(10))

	return Model{
		input:     ti,
		result:    vp,
		completer: NewCompleter(),
		selected:  -1,
	}
}

// SetSend stores the program.Send function for NL streaming.
func (m *Model) SetSend(send func(tea.Msg)) {
	m.send = send
}

// SetSize updates dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.input.SetWidth(w - 4)
	m.result.SetWidth(w - 2)
	m.result.SetHeight(h - 3)
}

// SetTriggerNames updates tab completion for trigger names.
func (m *Model) SetTriggerNames(names []string) {
	m.completer.SetTriggerNames(names)
}

// SetSessionIDs updates tab completion for session IDs.
func (m *Model) SetSessionIDs(ids []string) {
	m.completer.SetSessionIDs(ids)
}

// SetResult sets the result viewport content.
func (m *Model) SetResult(content string) {
	m.hasResult = true
	m.streaming = false
	m.candidates = nil
	m.result.SetContent(content)
	m.result.GotoTop()
}

// SetError sets an error in the result viewport.
func (m *Model) SetError(err error) {
	m.hasResult = true
	m.streaming = false
	m.candidates = nil
	m.result.SetContent(ui.StyleError.Render("Error: " + err.Error()))
	m.result.GotoTop()
}

// ClearResult clears the result viewport.
func (m *Model) ClearResult() {
	m.hasResult = false
	m.streaming = false
	m.candidates = nil
	m.selected = -1
	m.result.SetContent("")
	if m.nlProc != nil {
		m.nlProc.Kill()
		m.nlProc = nil
	}
	m.nlBuffer.Reset()
}

// Focus activates the command line input.
func (m *Model) Focus() tea.Cmd {
	m.focused = true
	m.candidates = nil
	m.selected = -1
	m.hasResult = false
	// Show all top-level candidates immediately.
	m.updateCandidates()
	return m.input.Focus()
}

// Blur deactivates the command line input.
func (m *Model) Blur() {
	m.focused = false
	m.candidates = nil
	m.selected = -1
	m.input.Blur()
}

// Focused returns whether the command line has focus.
func (m *Model) Focused() bool {
	return m.focused
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case NLStreamMsg:
		m.nlBuffer.WriteString(msg.Text)
		m.hasResult = true
		m.candidates = nil
		m.result.SetContent(m.nlBuffer.String())
		m.result.GotoBottom()
		return m, nil

	case NLDoneMsg:
		m.streaming = false
		m.nlProc = nil
		if msg.Err != nil && m.nlBuffer.Len() == 0 {
			m.SetError(msg.Err)
		}
		return m, nil
	}

	if !m.focused {
		return m, nil
	}

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		key := keyMsg.String()

		// Navigate completion menu.
		if len(m.candidates) > 0 {
			switch key {
			case "down":
				m.selected++
				if m.selected >= len(m.candidates) {
					m.selected = 0
				}
				return m, nil
			case "up":
				m.selected--
				if m.selected < 0 {
					m.selected = len(m.candidates) - 1
				}
				return m, nil
			case "tab":
				// Accept selected candidate or first candidate.
				idx := m.selected
				if idx < 0 {
					idx = 0
				}
				if idx < len(m.candidates) {
					m.acceptCandidate(idx)
					m.updateCandidates()
				}
				return m, nil
			}
		}

		switch key {
		case "enter":
			// If a candidate is selected, accept it first.
			if m.selected >= 0 && m.selected < len(m.candidates) {
				m.acceptCandidate(m.selected)
				m.updateCandidates()
				return m, nil
			}

			input := strings.TrimSpace(m.input.Value())
			if input == "" {
				return m, nil
			}
			m.input.SetValue("")
			m.candidates = nil
			m.selected = -1

			route := ParseRoute(input)
			switch route.Kind {
			case RouteCLI:
				return m, func() tea.Msg { return ExecuteMsg{Args: route.Args} }
			case RouteNL:
				return m, func() tea.Msg { return NLRequestMsg{Input: route.Raw} }
			}

		case "esc":
			m.Blur()
			m.ClearResult()
			return m, nil
		}

		// If a result is showing and user starts typing, clear it.
		if m.hasResult && key != "up" && key != "down" && key != "tab" {
			m.hasResult = false
			m.streaming = false
			m.candidates = nil
			m.result.SetContent("")
			if m.nlProc != nil {
				m.nlProc.Kill()
				m.nlProc = nil
			}
			m.nlBuffer.Reset()
		}

		// Pass key to textinput, then update candidates.
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.updateCandidates()
		return m, cmd
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) acceptCandidate(idx int) {
	if idx < 0 || idx >= len(m.candidates) {
		return
	}
	c := m.candidates[idx]
	val := m.input.Value()
	parts := strings.Fields(val)

	if strings.HasSuffix(val, " ") || len(parts) == 0 {
		// Append the candidate.
		m.input.SetValue(val + c.Value + " ")
	} else {
		// Replace the last partial word.
		parts[len(parts)-1] = c.Value
		m.input.SetValue(strings.Join(parts, " ") + " ")
	}
	m.input.CursorEnd()
	m.selected = -1
}

func (m *Model) updateCandidates() {
	if m.hasResult {
		m.candidates = nil
		return
	}
	m.candidates = m.completer.Complete(m.input.Value())
	m.selected = -1
}

// StartNLStream begins streaming NL output. Called by the parent model.
func (m *Model) StartNLStream(input string, send func(tea.Msg)) {
	m.ClearResult()
	m.hasResult = true
	m.streaming = true
	m.nlBuffer.Reset()
	m.result.SetContent(ui.StyleDim.Render("Thinking..."))
	m.nlProc = StartNL(input, send)
}

// MenuHeight returns the number of lines the completion menu occupies (excluding input line).
// Accounts for the border (2 lines) around the panel.
func (m Model) MenuHeight() int {
	if !m.focused || m.hasResult || len(m.candidates) == 0 {
		return 0
	}
	n := len(m.candidates)
	if n > 10 {
		n = 10
	}
	return n + 2 // +2 for top/bottom border
}

// ViewInput renders the input line + completion menu (for embedding at bottom of screen).
func (m Model) ViewInput() string {
	if !m.focused {
		return ""
	}

	var b strings.Builder

	// Completion menu panel (rendered above the input line).
	if len(m.candidates) > 0 && !m.hasResult {
		b.WriteString(m.renderCandidates())
		b.WriteByte('\n')
	}

	// Input line with a subtle prompt.
	b.WriteString(m.input.View())
	return b.String()
}

// ViewResult renders the result viewport (for embedding in main area).
func (m Model) ViewResult() string {
	if !m.hasResult {
		return ""
	}
	return m.result.View()
}

var (
	styleMenuPanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.ColorBorder).
			PaddingLeft(1).
			PaddingRight(1)

	styleSelectedRow = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(ui.T.Background)).
				Background(lipgloss.Color(ui.T.Accent))

	styleNormalValue = lipgloss.NewStyle().
				Foreground(ui.ColorWhite)

	styleDesc = lipgloss.NewStyle().
			Foreground(ui.ColorDim)
)

func (m *Model) renderCandidates() string {
	maxValue := 0
	for _, c := range m.candidates {
		if len(c.Value) > maxValue {
			maxValue = len(c.Value)
		}
	}

	shown := m.candidates
	if len(shown) > 10 {
		shown = shown[:10]
	}

	var rows strings.Builder
	for i, c := range shown {
		if i > 0 {
			rows.WriteByte('\n')
		}
		value := fmt.Sprintf("%-*s", maxValue, c.Value)
		desc := ""
		if c.Desc != "" {
			desc = "  " + c.Desc
		}

		if i == m.selected {
			// Highlight entire row.
			line := value + desc
			rows.WriteString(styleSelectedRow.Render(line))
		} else {
			rows.WriteString(styleNormalValue.Render(value))
			rows.WriteString(styleDesc.Render(desc))
		}
	}

	// Wrap in a bordered panel.
	panelWidth := m.width - 4
	if panelWidth < 40 {
		panelWidth = 40
	}
	panel := styleMenuPanel.Width(panelWidth).Render(rows.String())
	return panel
}
