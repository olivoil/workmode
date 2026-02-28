package app

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/olivoil/workmode/tui/internal/backend"
	"github.com/olivoil/workmode/tui/internal/ui"
	"github.com/olivoil/workmode/tui/internal/views/command"
	"github.com/olivoil/workmode/tui/internal/views/logview"
	"github.com/olivoil/workmode/tui/internal/views/sessions"
	"github.com/olivoil/workmode/tui/internal/views/triggers"
)

const statusPollInterval = 3 * time.Second

// Run starts the TUI application.
func Run() error {
	m := newModel()
	p := tea.NewProgram(m)

	// Wire up program.Send for NL streaming and watcher.
	sendFn := func(msg tea.Msg) { p.Send(msg) }
	m.commandView.SetSend(sendFn)
	m.send = sendFn

	w, err := backend.NewWatcher(m.client, p)
	if err == nil {
		m.watcher = w
		defer w.Close()
	}

	_, err = p.Run()
	return err
}

// viewMode identifies which view is active.
type viewMode int

const (
	viewSessions viewMode = iota
	viewTriggers
	viewLog
	viewCommand
)

// model is the root application model.
type model struct {
	width    int
	height   int
	mode     viewMode
	prevMode viewMode
	ready    bool
	showHelp bool
	keys     KeyMap

	client  *backend.Client
	watcher *backend.Watcher
	send    func(tea.Msg)

	status   backend.Status
	sessions []backend.Session
	triggers []backend.Trigger

	sessionsView sessions.Model
	triggersView triggers.Model
	commandView  command.Model
	logView      logview.Model
}

func newModel() model {
	client := backend.NewClient(CLIBinary, AppName)
	return model{
		mode:         viewSessions,
		keys:         DefaultKeyMap(),
		client:       client,
		sessionsView: sessions.New(),
		triggersView: triggers.New(),
		commandView:  command.New(),
		logView:      logview.New(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.loadStatus,
		m.loadSessions,
		m.loadTriggers,
		m.tickStatus(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.layoutViews()
		return m, nil

	case StatusLoadedMsg:
		if msg.Err == nil {
			m.status = backend.DeriveStats(msg.Status, m.sessions)
		}
		return m, nil

	case SessionsLoadedMsg:
		if msg.Err == nil {
			m.sessions = msg.Sessions
			m.sessionsView.SetSessions(msg.Sessions)
			m.triggersView.SetSessions(msg.Sessions)
			ids := make([]string, len(msg.Sessions))
			for i, s := range msg.Sessions {
				ids[i] = s.Short
			}
			m.commandView.SetSessionIDs(ids)
			m.status = backend.DeriveStats(m.status, m.sessions)
			return m, m.loadSelectedPreview()
		}
		return m, nil

	case TriggersLoadedMsg:
		if msg.Err == nil {
			m.triggers = msg.Triggers
			m.triggersView.SetTriggers(msg.Triggers)
			names := make([]string, len(msg.Triggers))
			for i, t := range msg.Triggers {
				names[i] = t.Name
			}
			m.commandView.SetTriggerNames(names)
		}
		return m, nil

	case LogLoadedMsg:
		if msg.Err != nil {
			return m, nil
		}
		switch m.mode {
		case viewSessions:
			m.sessionsView.SetPreview(msg.ShortID, msg.Events)
		case viewLog:
			if !m.logView.Active() {
				// First load — find the session and show.
				for i := range m.sessions {
					if m.sessions[i].Short == msg.ShortID {
						m.logView.Show(&m.sessions[i], msg.Events)
						break
					}
				}
			} else {
				m.logView.UpdateLog(msg.Events)
			}
		}
		return m, nil

	case backend.WatchMsg:
		switch msg.Kind {
		case backend.WatchHistory:
			return m, m.loadSessions
		case backend.WatchLog:
			if m.mode == viewLog {
				if s := m.logView.Session(); s != nil {
					return m, m.loadPreview(s.Short)
				}
			} else {
				return m, m.loadSelectedPreview()
			}
		}
		return m, nil

	case StatusTickMsg:
		return m, tea.Batch(m.loadStatus, m.tickStatus())

	case command.ExecuteMsg:
		return m, m.executeCommand(msg.Args)

	case command.NLRequestMsg:
		if m.send != nil {
			m.commandView.StartNLStream(msg.Input, m.send)
		}
		return m, nil

	case ActionResultMsg:
		if msg.Err != nil {
			m.commandView.SetError(msg.Err)
		} else {
			m.commandView.SetResult(msg.Output)
		}
		return m, tea.Batch(m.loadStatus, m.loadSessions, m.loadTriggers)

	case command.NLStreamMsg, command.NLDoneMsg:
		var cmd tea.Cmd
		m.commandView, cmd = m.commandView.Update(msg)
		return m, cmd

	case ResumeExitMsg:
		// Claude --resume exited. Refresh everything.
		return m, tea.Batch(m.loadStatus, m.loadSessions, m.loadTriggers)

	case tea.KeyPressMsg:
		// If command line has focus, let it handle keys first.
		if m.commandView.Focused() {
			var cmd tea.Cmd
			m.commandView, cmd = m.commandView.Update(msg)
			if !m.commandView.Focused() {
				m.restorePreviousView()
			}
			return m, cmd
		}
		return m.handleKey(msg)
	}

	return m.updateActiveView(msg)
}

func (m model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Log view has its own key handling.
	if m.mode == viewLog {
		switch key {
		case "q", "esc":
			m.logView.Hide()
			if m.watcher != nil {
				m.watcher.WatchLog("")
			}
			m.mode = m.prevMode
			m.focusCurrentView()
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+r":
			if s := m.logView.Session(); s != nil && s.SessionID != "" {
				return m, m.resumeSession(*s)
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.logView, cmd = m.logView.Update(msg)
		return m, cmd
	}

	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "tab":
		switch m.mode {
		case viewSessions:
			m.mode = viewTriggers
			m.sessionsView.Blur()
			m.triggersView.Focus()
		case viewTriggers:
			m.mode = viewSessions
			m.triggersView.Blur()
			m.sessionsView.Focus()
		}
		return m, nil

	case "enter":
		return m.handleEnter()

	case "ctrl+r":
		if m.mode == viewSessions {
			if s := m.sessionsView.SelectedSession(); s != nil && s.SessionID != "" {
				return m, m.resumeSession(*s)
			}
		}
		return m, nil

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "/":
		m.prevMode = m.mode
		m.sessionsView.Blur()
		m.triggersView.Blur()
		cmd := m.commandView.Focus()
		return m, cmd

	case "ctrl+l":
		return m, tea.Batch(m.loadStatus, m.loadSessions, m.loadTriggers)
	}

	return m.updateActiveView(msg)
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.mode {
	case viewSessions:
		s := m.sessionsView.SelectedSession()
		if s == nil {
			return m, nil
		}
		m.prevMode = m.mode
		m.mode = viewLog
		m.sessionsView.Blur()
		// Start watching the log file for live updates.
		if m.watcher != nil && s.Status == "running" {
			m.watcher.WatchLog(s.Short)
		}
		return m, m.openLogView(*s)

	case viewTriggers:
		t := m.triggersView.SelectedTrigger()
		if t == nil {
			return m, nil
		}
		return m, m.executeCommand([]string{"trigger", "run", t.Name})
	}
	return m, nil
}

func (m *model) restorePreviousView() {
	m.mode = m.prevMode
	m.focusCurrentView()
}

func (m *model) focusCurrentView() {
	switch m.mode {
	case viewSessions:
		m.sessionsView.Focus()
	case viewTriggers:
		m.triggersView.Focus()
	}
}

func (m model) updateActiveView(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case viewSessions:
		prev := m.sessionsView.SelectedShortID()
		var cmd tea.Cmd
		m.sessionsView, cmd = m.sessionsView.Update(msg)
		curr := m.sessionsView.SelectedShortID()
		if curr != prev && curr != "" {
			return m, tea.Batch(cmd, m.loadPreview(curr))
		}
		return m, cmd
	case viewTriggers:
		var cmd tea.Cmd
		m.triggersView, cmd = m.triggersView.Update(msg)
		return m, cmd
	case viewLog:
		var cmd tea.Cmd
		m.logView, cmd = m.logView.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) View() tea.View {
	var v tea.View
	v.AltScreen = true

	if !m.ready {
		v.SetContent("Loading...")
		return v
	}

	var b strings.Builder

	// Help overlay.
	if m.showHelp {
		v.SetContent(m.renderHelpOverlay())
		return v
	}

	// Full-screen log view (no header/footer).
	if m.mode == viewLog {
		b.WriteString(m.logView.View())
		b.WriteByte('\n')
		b.WriteString(ui.StyleDim.Render(" esc back  │  ctrl+r resume  │  j/k scroll  │  q quit"))
		v.SetContent(b.String())
		return v
	}

	// Header (2 lines: title + bar).
	b.WriteString(m.renderHeader())
	b.WriteByte('\n')

	// Calculate bottom area height.
	bottomLines := 1 // help line or input line
	menuHeight := m.commandView.MenuHeight()
	bottomLines += menuHeight

	// Resize content area to fit.
	contentHeight := m.height - 3 - menuHeight // 3 = header(2) + bottom(1)
	if contentHeight < 5 {
		contentHeight = 5
	}
	m.sessionsView.SetSize(m.width, contentHeight)
	m.triggersView.SetSize(m.width, contentHeight)

	// Main content area.
	if resultView := m.commandView.ViewResult(); resultView != "" {
		b.WriteString(resultView)
	} else {
		switch m.mode {
		case viewSessions:
			b.WriteString(m.sessionsView.View())
		case viewTriggers:
			b.WriteString(m.triggersView.View())
		default:
			b.WriteString(ui.StyleDim.Render(" ..."))
		}
	}

	// Bottom: command input (with menu) or help line.
	b.WriteByte('\n')
	if m.commandView.Focused() {
		b.WriteString(m.commandView.ViewInput())
	} else {
		b.WriteString(m.renderHelpLine())
	}

	v.SetContent(b.String())
	return v
}

func (m *model) renderHeader() string {
	title := ui.StyleHeader.Render(fmt.Sprintf(" %s ", AppName))

	var statusStr string
	if m.status.Active {
		statusStr = ui.StyleActive.Render("● ACTIVE")
	} else {
		statusStr = ui.StyleInactive.Render("○ INACTIVE")
	}

	var watcherStr string
	if m.status.Watcher {
		watcherStr = ui.StyleDim.Render("watcher: ") + ui.StyleActive.Render("up")
	} else {
		watcherStr = ui.StyleDim.Render("watcher: ") + ui.StyleInactive.Render("down")
	}

	stats := ui.StyleDim.Render(fmt.Sprintf(
		"timers: %d/%d   running: %d   today: %d",
		m.status.Timers, m.status.Triggers, m.status.Running, m.status.Today,
	))

	sep := ui.StyleDim.Render("   ")
	header := lipgloss.JoinHorizontal(lipgloss.Center,
		title, sep, statusStr, sep, watcherStr, sep, stats,
	)

	bar := strings.Repeat("━", m.width)
	return header + "\n" + ui.StyleDim.Render(bar)
}

func (m *model) renderHelpLine() string {
	var parts []string
	switch m.mode {
	case viewSessions:
		parts = []string{"↑↓ navigate", "enter open", "ctrl+r resume", "/ command", "tab triggers", "q quit"}
	case viewTriggers:
		parts = []string{"↑↓ navigate", "enter run", "tab sessions", "/ command", "q quit"}
	}
	return ui.StyleDim.Render(" " + strings.Join(parts, "  │  "))
}

func (m *model) renderHelpOverlay() string {
	title := ui.StyleHeader.Render(fmt.Sprintf(" %s help ", AppName))
	help := `
  Navigation
    ↑/↓, j/k       Navigate list
    tab             Switch sessions ↔ triggers
    enter           Open session log / run trigger
    esc             Back to previous view
    q, ctrl+c       Quit

  Session Actions
    ctrl+r          Resume session in Claude
    ctrl+s          Stop running session
    ctrl+k          Kill running session

  Command Line
    /               Open command line
    enter           Execute command
    tab             Tab completion
    esc             Close command line

  Commands
    on / off        Enable/disable workmode
    status          Show status
    trigger run X   Run trigger X
    session logs X  View session X logs
    <anything>      Ask Claude (natural language)

  Other
    ctrl+l          Refresh all data
    ?               Toggle this help

  ` + ui.StyleDim.Render("Press ? to close")
	return title + "\n" + help
}

func (m *model) layoutViews() {
	viewHeight := m.height - 3
	if viewHeight < 5 {
		viewHeight = 5
	}
	m.sessionsView.SetSize(m.width, viewHeight)
	m.triggersView.SetSize(m.width, viewHeight)
	m.commandView.SetSize(m.width, viewHeight)
	m.logView.SetSize(m.width, m.height-1) // full height minus help line
}

// --- Commands ---

func (m *model) loadStatus() tea.Msg {
	s, err := m.client.Status()
	return StatusLoadedMsg{Status: s, Err: err}
}

func (m *model) loadSessions() tea.Msg {
	sessions, err := m.client.ReadSessions()
	return SessionsLoadedMsg{Sessions: sessions, Err: err}
}

func (m *model) loadTriggers() tea.Msg {
	triggers, err := m.client.Triggers()
	return TriggersLoadedMsg{Triggers: triggers, Err: err}
}

func (m *model) loadPreview(shortID string) tea.Cmd {
	return func() tea.Msg {
		events, err := m.client.ReadLog(shortID)
		return LogLoadedMsg{ShortID: shortID, Events: events, Err: err}
	}
}

func (m *model) loadSelectedPreview() tea.Cmd {
	id := m.sessionsView.SelectedShortID()
	if id == "" {
		return nil
	}
	return m.loadPreview(id)
}

func (m *model) openLogView(s backend.Session) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		events, _ := client.ReadLog(s.Short)
		return LogLoadedMsg{ShortID: s.Short, Events: events}
	}
}

func (m *model) resumeSession(s backend.Session) tea.Cmd {
	cmd := m.client.ResumeCmd(s)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return ResumeExitMsg{Err: err}
	})
}

func (m *model) tickStatus() tea.Cmd {
	return tea.Tick(statusPollInterval, func(time.Time) tea.Msg {
		return StatusTickMsg{}
	})
}

func (m *model) executeCommand(args []string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		out, err := client.RunCommand(args...)
		return ActionResultMsg{Output: string(out), Err: err}
	}
}
