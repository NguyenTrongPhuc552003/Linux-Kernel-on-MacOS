// Package tui provides the interactive Text User Interface for elmos.
// This file contains color definitions and lipgloss styles.
package tui

import "github.com/charmbracelet/lipgloss"

// Color palette using AdaptiveColor to support both 256-color and 16-color terminals.
// This ensures correct rendering on WSL2 environments that may only have 16-color support.
// Dark values are 256-color palette indices; Light fallbacks use standard ANSI 16-color codes.
var (
	purple      = lipgloss.AdaptiveColor{Light: "5", Dark: "141"} // ANSI Magenta / 256 purple
	cyan        = lipgloss.AdaptiveColor{Light: "6", Dark: "51"}  // ANSI Cyan / 256 bright cyan
	green       = lipgloss.AdaptiveColor{Light: "2", Dark: "120"} // ANSI Green / 256 bright green
	orange      = lipgloss.AdaptiveColor{Light: "3", Dark: "214"} // ANSI Yellow / 256 orange
	red         = lipgloss.AdaptiveColor{Light: "1", Dark: "203"} // ANSI Red / 256 bright red
	white       = lipgloss.AdaptiveColor{Light: "0", Dark: "255"} // ANSI text / 256 white
	grey        = lipgloss.AdaptiveColor{Light: "8", Dark: "245"} // ANSI Dark Gray / 256 medium grey
	darkGrey    = lipgloss.AdaptiveColor{Light: "8", Dark: "238"} // ANSI Dark Gray / 256 dark grey
	borderColor = lipgloss.AdaptiveColor{Light: "8", Dark: "240"} // ANSI Dark Gray / 256 border grey
	inputBg     = lipgloss.AdaptiveColor{Light: "7", Dark: "236"} // ANSI Light Gray / 256 dark input bg
)

// Panel and component styles
var (
	leftPanelStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(purple)
	rightPanelStyle   = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(borderColor)
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(purple)
	menuItemStyle     = lipgloss.NewStyle().Foreground(grey)
	selectedItemStyle = lipgloss.NewStyle().Bold(true).Foreground(white).Background(purple)
	hintStyle         = lipgloss.NewStyle().Foreground(cyan).Border(lipgloss.RoundedBorder()).BorderForeground(cyan).Padding(0, 1)
	descStyle         = lipgloss.NewStyle().Foreground(darkGrey).Italic(true)
	inputStyle        = lipgloss.NewStyle().Foreground(white).Background(inputBg).Padding(0, 1)
	inputLabelStyle   = lipgloss.NewStyle().Foreground(orange).Bold(true)
)
