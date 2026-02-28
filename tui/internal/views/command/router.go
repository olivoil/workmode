package command

import "strings"

// RouteKind identifies whether input is a CLI command or natural language.
type RouteKind int

const (
	RouteNL  RouteKind = iota // natural language → send to Claude
	RouteCLI                  // structured CLI command
)

// Route represents a parsed command input.
type Route struct {
	Kind RouteKind
	Args []string // for CLI commands
	Raw  string   // original input
}

// known top-level commands and their subcommands
var commandTree = map[string][]string{
	"on":      nil,
	"off":     nil,
	"status":  nil,
	"trigger": {"list", "show", "run", "enable", "disable"},
	"session": {"list", "logs", "tail", "resume", "stop", "kill"},
	"config":  {"show", "edit", "validate", "apply", "path"},
	"help":    nil,
	"version": nil,
}

// ParseRoute decides whether input is a CLI command or NL.
func ParseRoute(input string) Route {
	input = strings.TrimSpace(input)
	if input == "" {
		return Route{Kind: RouteNL, Raw: input}
	}

	parts := strings.Fields(input)
	cmd := parts[0]

	if _, ok := commandTree[cmd]; ok {
		return Route{Kind: RouteCLI, Args: parts, Raw: input}
	}

	return Route{Kind: RouteNL, Args: parts, Raw: input}
}
