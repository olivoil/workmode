package backend

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ParseSessionsFromFile reads history.jsonl and returns deduplicated sessions.
// Later entries for the same ID overwrite earlier ones (last-write-wins).
func ParseSessionsFromFile(path string) ([]Session, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open history: %w", err)
	}
	defer f.Close()

	seen := make(map[string]int) // id → index in result
	var sessions []Session

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var s Session
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			continue // skip malformed lines
		}
		if s.ID == "" {
			continue
		}
		if idx, ok := seen[s.ID]; ok {
			sessions[idx] = s // overwrite with latest status
		} else {
			seen[s.ID] = len(sessions)
			sessions = append(sessions, s)
		}
	}

	// Reverse so newest sessions are first.
	for i, j := 0, len(sessions)-1; i < j; i, j = i+1, j-1 {
		sessions[i], sessions[j] = sessions[j], sessions[i]
	}

	return sessions, scanner.Err()
}

// ParseLogFile reads a stream-json log file and returns parsed events.
func ParseLogFile(path string) ([]StreamEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open log: %w", err)
	}
	defer f.Close()

	var events []StreamEvent
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e StreamEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		events = append(events, e)
	}
	return events, scanner.Err()
}

// ExtractSummary returns a short text summary from log events.
// It takes the first non-empty text content from the last assistant message,
// or the result field, truncated to maxLen.
func ExtractSummary(events []StreamEvent, maxLen int) string {
	var lastText, result string
	for _, e := range events {
		switch e.Type {
		case "assistant":
			if e.Message != nil {
				for _, c := range e.Message.Content {
					if c.Type == "text" && c.Text != "" {
						lastText = c.Text
					}
				}
			}
		case "result":
			if e.Result != "" {
				result = e.Result
			}
		}
	}

	summary := result
	if summary == "" {
		summary = lastText
	}
	if summary == "" {
		return ""
	}

	// Take first line only.
	if idx := strings.IndexByte(summary, '\n'); idx >= 0 {
		summary = summary[:idx]
	}
	summary = strings.TrimSpace(summary)
	if len(summary) > maxLen {
		summary = summary[:maxLen-1] + "…"
	}
	return summary
}

// FormatLogEvents renders stream-json events into human-readable text.
func FormatLogEvents(events []StreamEvent) string {
	var b strings.Builder
	for _, e := range events {
		switch e.Type {
		case "assistant":
			if e.Message != nil {
				for _, c := range e.Message.Content {
					switch c.Type {
					case "text":
						b.WriteString(c.Text)
						b.WriteByte('\n')
					case "tool_use":
						fmt.Fprintf(&b, "[tool: %s]\n", c.Name)
					}
				}
			}
		case "tool_use":
			fmt.Fprintf(&b, "[tool: %s]\n", e.Name)
		case "result":
			if e.Result != "" {
				b.WriteString(e.Result)
				b.WriteByte('\n')
			}
		}
	}
	return b.String()
}
