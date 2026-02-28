package command

import (
	"sort"
	"strings"
)

// Candidate is a completion option with a description.
type Candidate struct {
	Value string // the text to insert
	Desc  string // short description
}

// Completer provides live completion for the command line.
type Completer struct {
	triggerNames []string
	sessionIDs   []string
}

// NewCompleter creates a completer.
func NewCompleter() *Completer {
	return &Completer{}
}

// SetTriggerNames updates the available trigger names.
func (c *Completer) SetTriggerNames(names []string) {
	c.triggerNames = names
}

// SetSessionIDs updates the available session short IDs.
func (c *Completer) SetSessionIDs(ids []string) {
	c.sessionIDs = ids
}

// command tree with descriptions
type cmdEntry struct {
	subs []subEntry
	desc string
}

type subEntry struct {
	name string
	desc string
}

var commands = map[string]cmdEntry{
	"on":      {desc: "Enable workmode triggers"},
	"off":     {desc: "Disable workmode triggers"},
	"status":  {desc: "Show workmode status"},
	"trigger": {desc: "Manage triggers", subs: []subEntry{
		{"list", "List all triggers"},
		{"show", "Show trigger details"},
		{"run", "Run a trigger now"},
		{"enable", "Enable a trigger"},
		{"disable", "Disable a trigger"},
	}},
	"session": {desc: "Manage sessions", subs: []subEntry{
		{"list", "List all sessions"},
		{"logs", "View session log"},
		{"tail", "Tail session log"},
		{"resume", "Resume session in Claude"},
		{"stop", "Stop running session"},
		{"kill", "Kill running session"},
	}},
	"config": {desc: "Manage configuration", subs: []subEntry{
		{"show", "Show current config"},
		{"edit", "Edit config file"},
		{"validate", "Validate config"},
		{"apply", "Apply config changes"},
		{"path", "Show config file path"},
	}},
	"help":    {desc: "Show help"},
	"version": {desc: "Show version"},
}

// Complete returns candidates for the current input.
func (c *Completer) Complete(input string) []Candidate {
	parts := strings.Fields(input)
	trailing := strings.HasSuffix(input, " ")

	// No input yet or partial first word — show top-level commands.
	if len(parts) == 0 || (len(parts) == 1 && !trailing) {
		prefix := ""
		if len(parts) == 1 {
			prefix = parts[0]
		}
		return c.topLevelCandidates(prefix)
	}

	cmd := parts[0]
	entry, ok := commands[cmd]
	if !ok {
		return nil
	}

	// First word complete, show subcommands.
	if len(parts) == 1 && trailing {
		return subCandidates(entry.subs, "")
	}
	if len(parts) == 2 && !trailing {
		return subCandidates(entry.subs, parts[1])
	}

	// Subcommand complete, show arguments (trigger names or session IDs).
	if (len(parts) == 2 && trailing) || (len(parts) == 3 && !trailing) {
		prefix := ""
		if len(parts) == 3 {
			prefix = parts[2]
		}
		sub := parts[1]

		switch cmd {
		case "trigger":
			if sub == "run" || sub == "show" || sub == "enable" || sub == "disable" {
				return c.dynamicCandidates(c.triggerNames, prefix, "trigger")
			}
		case "session":
			if sub == "logs" || sub == "tail" || sub == "resume" || sub == "stop" || sub == "kill" {
				return c.dynamicCandidates(c.sessionIDs, prefix, "session")
			}
		}
	}

	return nil
}

func (c *Completer) topLevelCandidates(prefix string) []Candidate {
	// Sorted keys.
	keys := make([]string, 0, len(commands))
	for k := range commands {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var result []Candidate
	for _, k := range keys {
		if prefix == "" || strings.HasPrefix(k, prefix) {
			result = append(result, Candidate{Value: k, Desc: commands[k].desc})
		}
	}
	return result
}

func subCandidates(subs []subEntry, prefix string) []Candidate {
	var result []Candidate
	for _, s := range subs {
		if prefix == "" || strings.HasPrefix(s.name, prefix) {
			result = append(result, Candidate{Value: s.name, Desc: s.desc})
		}
	}
	return result
}

func (c *Completer) dynamicCandidates(items []string, prefix, kind string) []Candidate {
	var result []Candidate
	for _, item := range items {
		if prefix == "" || strings.HasPrefix(item, prefix) {
			result = append(result, Candidate{Value: item, Desc: kind})
		}
	}
	return result
}
