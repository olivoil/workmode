package command

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/olivoil/workmode/tui/internal/backend"
)

// NLStreamMsg carries a streamed text chunk from Claude.
type NLStreamMsg struct {
	Text string
}

// NLDoneMsg indicates NL streaming is complete.
type NLDoneMsg struct {
	Err error
}

// NLProcess manages a running Claude NL process.
type NLProcess struct {
	cmd    *exec.Cmd
	cancel func()
}

// StartNL spawns `claude -p --output-format stream-json --skill workmode "input"`
// and streams parsed text back via program.Send.
func StartNL(input string, send func(tea.Msg)) *NLProcess {
	cmd := exec.Command("claude", "-p",
		"--output-format", "stream-json",
		"--skill", "workmode",
		input,
	)
	// Unset CLAUDECODE for nested invocation.
	cmd.Env = append(cmd.Environ(), "CLAUDECODE=")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		send(NLDoneMsg{Err: err})
		return nil
	}

	if err := cmd.Start(); err != nil {
		send(NLDoneMsg{Err: err})
		return nil
	}

	proc := &NLProcess{cmd: cmd}

	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var event backend.StreamEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				continue
			}

			text := extractText(event)
			if text != "" {
				send(NLStreamMsg{Text: text})
			}
		}

		err := cmd.Wait()
		send(NLDoneMsg{Err: err})
	}()

	return proc
}

// Kill terminates the NL process.
func (p *NLProcess) Kill() {
	if p != nil && p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
}

func extractText(event backend.StreamEvent) string {
	switch event.Type {
	case "assistant":
		if event.Message != nil {
			var parts []string
			for _, c := range event.Message.Content {
				switch c.Type {
				case "text":
					parts = append(parts, c.Text)
				case "tool_use":
					parts = append(parts, fmt.Sprintf("[tool: %s]", c.Name))
				}
			}
			return strings.Join(parts, "\n")
		}
	case "tool_use":
		return fmt.Sprintf("[tool: %s]", event.Name)
	case "result":
		return event.Result
	}
	return ""
}
