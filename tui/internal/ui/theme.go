package ui

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Theme holds the resolved color palette as hex strings.
type Theme struct {
	Foreground          string
	Background          string
	Accent              string
	SelectionForeground string
	SelectionBackground string
	Dim                 string
	Red                 string
	Green               string
	Yellow              string
	Blue                string
	Border              string
	BrightWhite         string
}

// omarchyColors matches the colors.toml format.
type omarchyColors struct {
	Accent              string `toml:"accent"`
	Foreground          string `toml:"foreground"`
	Background          string `toml:"background"`
	Cursor              string `toml:"cursor"`
	SelectionForeground string `toml:"selection_foreground"`
	SelectionBackground string `toml:"selection_background"`
	Color0              string `toml:"color0"`
	Color1              string `toml:"color1"`
	Color2              string `toml:"color2"`
	Color3              string `toml:"color3"`
	Color4              string `toml:"color4"`
	Color5              string `toml:"color5"`
	Color6              string `toml:"color6"`
	Color7              string `toml:"color7"`
	Color8              string `toml:"color8"`
	Color9              string `toml:"color9"`
	Color10             string `toml:"color10"`
	Color11             string `toml:"color11"`
	Color12             string `toml:"color12"`
	Color13             string `toml:"color13"`
	Color14             string `toml:"color14"`
	Color15             string `toml:"color15"`
}

// defaultTheme returns the built-in fallback theme.
func defaultTheme() Theme {
	return Theme{
		Foreground:          "#e5e7eb",
		Background:          "#1a1b26",
		Accent:              "#8b5cf6",
		SelectionForeground: "#e5e7eb",
		SelectionBackground: "#8b5cf6",
		Dim:                 "#6b7280",
		Red:                 "#ef4444",
		Green:               "#22c55e",
		Yellow:              "#eab308",
		Blue:                "#3b82f6",
		Border:              "#374151",
		BrightWhite:         "#f9fafb",
	}
}

// LoadTheme tries to read the Omarchy theme, falling back to defaults.
func LoadTheme() Theme {
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultTheme()
	}

	colorsPath := filepath.Join(home, ".config", "omarchy", "current", "theme", "colors.toml")
	var oc omarchyColors
	if _, err := toml.DecodeFile(colorsPath, &oc); err != nil {
		return defaultTheme()
	}

	t := defaultTheme()

	if oc.Foreground != "" {
		t.Foreground = oc.Foreground
	}
	if oc.Background != "" {
		t.Background = oc.Background
	}
	if oc.Accent != "" {
		t.Accent = oc.Accent
	}
	if oc.SelectionForeground != "" {
		t.SelectionForeground = oc.SelectionForeground
	}
	if oc.SelectionBackground != "" {
		t.SelectionBackground = oc.SelectionBackground
	}
	if oc.Color0 != "" {
		t.Dim = oc.Color0
	}
	if oc.Color1 != "" {
		t.Red = oc.Color1
	}
	if oc.Color2 != "" {
		t.Green = oc.Color2
	}
	if oc.Color3 != "" {
		t.Yellow = oc.Color3
	}
	if oc.Color4 != "" {
		t.Blue = oc.Color4
	}
	if oc.Color8 != "" {
		t.Border = oc.Color8
	}
	if oc.Color15 != "" {
		t.BrightWhite = oc.Color15
	}

	return t
}
