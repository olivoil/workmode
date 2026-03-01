package ui

import (
	"fmt"
	"image/color"
	"time"

	"charm.land/lipgloss/v2"
)

// T is the active theme.
var T Theme

func init() {
	T = LoadTheme()
	initStyles()
}

// ReloadTheme re-reads the Omarchy theme and reinitializes all styles.
func ReloadTheme() {
	T = LoadTheme()
	initStyles()
}

// Color aliases for convenience — point into the active theme.
var (
	ColorGreen  color.Color
	ColorRed    color.Color
	ColorYellow color.Color
	ColorBlue   color.Color
	ColorDim    color.Color
	ColorWhite  color.Color
	ColorBorder color.Color
	ColorAccent color.Color
	ColorHeader color.Color
)

// Styles built from the active theme.
var (
	StyleHeader        lipgloss.Style
	StyleActive        lipgloss.Style
	StyleInactive      lipgloss.Style
	StyleDim           lipgloss.Style
	StyleAccent        lipgloss.Style
	StyleError         lipgloss.Style
	StylePreviewBorder lipgloss.Style
	StyleSelected      lipgloss.Style
)

func initStyles() {
	ColorGreen = lipgloss.Color(T.Green)
	ColorRed = lipgloss.Color(T.Red)
	ColorYellow = lipgloss.Color(T.Yellow)
	ColorBlue = lipgloss.Color(T.Blue)
	ColorDim = lipgloss.Color(T.Dim)
	ColorWhite = lipgloss.Color(T.Foreground)
	ColorBorder = lipgloss.Color(T.Border)
	ColorAccent = lipgloss.Color(T.Accent)
	ColorHeader = lipgloss.Color(T.BrightWhite)

	StyleHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorHeader)

	StyleActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorGreen)

	StyleInactive = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorRed)

	StyleDim = lipgloss.NewStyle().
		Foreground(ColorDim)

	StyleAccent = lipgloss.NewStyle().
		Foreground(ColorAccent)

	StyleError = lipgloss.NewStyle().
		Foreground(ColorRed)

	StylePreviewBorder = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ColorBorder).
		PaddingLeft(1)

	StyleSelected = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(T.Accent))
}

// StatusIcon returns a plain indicator character for a session status.
// Returns plain text (no ANSI styling) so the table's Selected style
// can properly override colors on the active row.
func StatusIcon(status string) string {
	switch status {
	case "completed":
		return "✓"
	case "running":
		return "●"
	case "error":
		return "✗"
	case "stuck":
		return "!"
	case "stopped":
		return "■"
	case "killed":
		return "†"
	default:
		return " "
	}
}

// StatusColor returns the theme color for a session status.
func StatusColor(status string) color.Color {
	switch status {
	case "completed":
		return ColorGreen
	case "running":
		return ColorBlue
	case "error":
		return ColorRed
	case "stuck":
		return ColorYellow
	case "stopped", "killed":
		return ColorDim
	default:
		return ColorDim
	}
}

// FormatDuration formats seconds into a human-readable duration.
func FormatDuration(secs int) string {
	if secs <= 0 {
		return ""
	}
	d := time.Duration(secs) * time.Second
	if d < time.Minute {
		return fmt.Sprintf("%ds", secs)
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh%dm", h, m)
}

// FormatTime formats an ISO 8601 timestamp into a short time string.
func FormatTime(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	now := time.Now()
	t = t.Local()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}
	if now.Sub(t) < 7*24*time.Hour {
		return t.Format("Mon 15:04")
	}
	return t.Format("Jan 02")
}
