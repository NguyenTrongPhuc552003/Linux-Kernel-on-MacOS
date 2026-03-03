// Package tui provides the interactive Text User Interface for elmos.
// This file contains the View function for UI rendering.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model and renders the entire TUI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	panelHeight := m.height - 2
	leftPanel := m.renderLeftPanel(panelHeight)
	rightPanel := m.renderRightPanel(panelHeight)
	main := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, main, footer)
}

// renderLeftPanel renders the menu panel.
func (m Model) renderLeftPanel(panelHeight int) string {
	var left strings.Builder
	title := "ELMOS"
	if m.parentTitle != "" {
		title = m.parentTitle
	}
	left.WriteString(titleStyle.Render("─ "+title+" ─") + "\n\n")
	if len(m.menuStack) > 0 {
		left.WriteString(lipgloss.NewStyle().Foreground(darkGrey).Render("  ← Back (Esc)") + "\n\n")
	}
	for i, item := range m.currentMenu {
		prefix := getMenuItemPrefix(item)
		label := prefix + item.Label
		if i == m.cursor {
			left.WriteString(selectedItemStyle.Render(" "+label+" ") + "\n")
		} else {
			left.WriteString(menuItemStyle.Render(label) + "\n")
		}
	}
	for i := strings.Count(left.String(), "\n"); i < panelHeight-4; i++ {
		left.WriteString("\n")
	}
	return leftPanelStyle.Width(m.leftWidth).Height(panelHeight).Render(left.String())
}

// getMenuItemPrefix returns the appropriate prefix for a menu item.
func getMenuItemPrefix(item MenuItem) string {
	if len(item.Children) > 0 {
		return "▸ "
	}
	if item.Interactive {
		return "⚡"
	}
	if item.NeedsInput {
		return "✎ "
	}
	return "• "
}

// renderRightPanel renders the output/input panel.
func (m Model) renderRightPanel(panelHeight int) string {
	var right strings.Builder
	rightTitle := m.getRightPanelTitle()
	scrollInfo := m.getScrollInfo()
	right.WriteString(titleStyle.Render("─ "+rightTitle+scrollInfo+" ─") + "\n\n")

	if m.inputMode {
		m.renderInputSection(&right)
	} else if m.cursor < len(m.currentMenu) && !m.isRunning {
		m.renderMenuHint(&right)
	}
	right.WriteString(m.viewport.View())

	return rightPanelStyle.Width(m.rightWidth).Height(panelHeight).Render(right.String())
}

// getRightPanelTitle returns the title for the right panel.
func (m Model) getRightPanelTitle() string {
	if m.isRunning {
		return m.spinner.View() + " " + m.currentTask
	}
	if m.inputMode {
		return ">> Input Required"
	}
	return "Output"
}

// getScrollInfo returns scroll percentage info if applicable.
func (m Model) getScrollInfo() string {
	if m.viewport.TotalLineCount() > m.viewport.Height {
		return fmt.Sprintf(" [%d%%]", int(m.viewport.ScrollPercent()*100))
	}
	return ""
}

// renderInputSection renders the input mode UI.
func (m Model) renderInputSection(w *strings.Builder) {
	w.WriteString(inputLabelStyle.Render(m.inputPrompt) + "\n\n")
	w.WriteString(inputStyle.Render(m.textInput.View()) + "\n\n")
	w.WriteString(descStyle.Render("  Press Enter to confirm, Esc to cancel") + "\n\n")
}

// renderMenuHint renders the hint for the current menu item.
func (m Model) renderMenuHint(w *strings.Builder) {
	item := m.currentMenu[m.cursor]
	if item.Command != "" {
		w.WriteString(hintStyle.Render(" $ "+item.Command+" ") + "\n")
		if item.Desc != "" {
			w.WriteString(descStyle.Render("  "+item.Desc) + "\n")
		}
		if item.NeedsInput {
			w.WriteString("\n" + inputLabelStyle.Render("  ✎ Press Enter to type: "+item.InputPrompt) + "\n")
		}
	} else if len(item.Children) > 0 {
		w.WriteString(descStyle.Render("  Press Enter to expand → "+item.Desc) + "\n")
	}
	w.WriteString("\n")
}

// renderFooter renders the keybindings footer.
func (m Model) renderFooter() string {
	return lipgloss.NewStyle().Foreground(darkGrey).Render(
		lipgloss.NewStyle().Foreground(cyan).Render("↑↓") + " Navigate  " +
			lipgloss.NewStyle().Foreground(cyan).Render("⏎") + " Select  " +
			lipgloss.NewStyle().Foreground(cyan).Render("Esc") + " Back  " +
			lipgloss.NewStyle().Foreground(cyan).Render("[ ]") + " Scroll  " +
			lipgloss.NewStyle().Foreground(cyan).Render("c") + " Clear  " +
			lipgloss.NewStyle().Foreground(cyan).Render("q") + " Quit")
}
