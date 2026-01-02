// Package ui provides console output helpers for elmos.
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles for console output.
var (
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red
	WarnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow
	InfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // Blue
	AccentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("13")) // Purple
)
