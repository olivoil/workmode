package ui

import (
	"fmt"
	"time"

	"charm.land/lipgloss/v2"
)

var (
	ColorGreen   = lipgloss.Color("#22c55e")
	ColorRed     = lipgloss.Color("#ef4444")
	ColorYellow  = lipgloss.Color("#eab308")
	ColorBlue    = lipgloss.Color("#3b82f6")
	ColorDim     = lipgloss.Color("#6b7280")
	ColorWhite   = lipgloss.Color("#e5e7eb")
	ColorBorder  = lipgloss.Color("#374151")
	ColorAccent  = lipgloss.Color("#8b5cf6")
	ColorHeader  = lipgloss.Color("#f9fafb")

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
)

// StatusIcon returns an icon for a session status.
func StatusIcon(status string) string {
	switch status {
	case "completed":
		return lipgloss.NewStyle().Foreground(ColorGreen).Render("✅")
	case "running":
		return lipgloss.NewStyle().Foreground(ColorBlue).Render("🔄")
	case "error":
		return lipgloss.NewStyle().Foreground(ColorRed).Render("❌")
	case "stuck":
		return lipgloss.NewStyle().Foreground(ColorYellow).Render("⚠️")
	case "stopped":
		return lipgloss.NewStyle().Foreground(ColorDim).Render("⏹")
	case "killed":
		return lipgloss.NewStyle().Foreground(ColorRed).Render("💀")
	default:
		return " "
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
