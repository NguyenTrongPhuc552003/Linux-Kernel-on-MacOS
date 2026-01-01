package tui

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommandRunner is a function that executes an action and returns output
type CommandRunner func(action string) (string, error)

// MenuItem represents a single menu item
type MenuItem struct {
	Label       string
	Action      string
	Status      string
	Description string
	Command     string // Command to execute (for display)
}

// Category represents a menu category
type Category struct {
	Name     string
	Expanded bool
	Items    []MenuItem
}

// Styles
var (
	accentColor    = lipgloss.Color("#7C3AED")
	successColor   = lipgloss.Color("#10B981")
	warningColor   = lipgloss.Color("#F59E0B")
	errorColor     = lipgloss.Color("#EF4444")
	dimColor       = lipgloss.Color("#6B7280")
	highlightColor = lipgloss.Color("#A78BFA")

	leftPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(0, 1)

	rightPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dimColor).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(accentColor).
			Padding(0, 2)

	categoryStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			PaddingLeft(1)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(4)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(highlightColor).
			PaddingLeft(2).
			PaddingRight(2)

	statusReadyStyle   = lipgloss.NewStyle().Foreground(successColor)
	statusPendingStyle = lipgloss.NewStyle().Foreground(warningColor)
	statusErrorStyle   = lipgloss.NewStyle().Foreground(errorColor)

	footerStyle = lipgloss.NewStyle().Foreground(dimColor)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)

	descriptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9CA3AF"))

	outputStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
	runningStyle    = lipgloss.NewStyle().Foreground(warningColor).Bold(true)
	successMsgStyle = lipgloss.NewStyle().Foreground(successColor)
	errorMsgStyle   = lipgloss.NewStyle().Foreground(errorColor)
	dimStyle        = lipgloss.NewStyle().Foreground(dimColor)
)

// FlatItem represents a flattened menu item for navigation
type FlatItem struct {
	Category    *Category
	Item        *MenuItem
	IsCategory  bool
	CategoryIdx int
	ItemIdx     int
}

// OutputLine represents a line in the output panel
type OutputLine struct {
	Text  string
	Style lipgloss.Style
}

// MenuModel represents the TUI menu state
type MenuModel struct {
	categories    []Category
	flatItems     []FlatItem
	cursor        int
	choice        string
	quitting      bool
	width         int
	height        int
	outputLines   []OutputLine
	isRunning     bool
	lastAction    string
	commandRunner CommandRunner
	scrollOffset  int
}

// Key bindings
type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Toggle key.Binding
	Quit   key.Binding
	Clear  key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("tab", " "),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
	),
	Clear: key.NewBinding(
		key.WithKeys("c"),
	),
}

// CommandResultMsg is sent when a command completes
type CommandResultMsg struct {
	Output string
	Err    error
	Action string
}

// NewMenuModel creates a new menu model with categories
func NewMenuModel() MenuModel {
	categories := []Category{
		{
			Name:     "Setup",
			Expanded: true,
			Items: []MenuItem{
				{
					Label:       "Doctor",
					Action:      "Doctor (Check Environment)",
					Status:      "ready",
					Description: "Check dependencies and environment",
					Command:     "elmos doctor",
				},
				{
					Label:       "Init Workspace",
					Action:      "Init Workspace",
					Status:      "pending",
					Description: "Mount image and clone kernel",
					Command:     "elmos init",
				},
				{
					Label:       "Configure",
					Action:      "Configure (Arch, Jobs...)",
					Description: "View/modify build configuration",
					Command:     "elmos config show",
				},
			},
		},
		{
			Name:     "Build",
			Expanded: true,
			Items: []MenuItem{
				{
					Label:       "Kernel Config",
					Action:      "Kernel Config (defconfig)",
					Description: "Generate default kernel config",
					Command:     "elmos kernel config",
				},
				{
					Label:       "Kernel Menuconfig",
					Action:      "Kernel Menuconfig (UI)",
					Description: "Interactive kernel configuration",
					Command:     "elmos kernel config menuconfig",
				},
				{
					Label:       "Build Kernel",
					Action:      "Build Kernel",
					Description: "Build kernel image and modules",
					Command:     "elmos build",
				},
				{
					Label:       "Build Modules",
					Action:      "Build Modules",
					Description: "Build out-of-tree modules",
					Command:     "elmos module build",
				},
				{
					Label:       "Build Apps",
					Action:      "Build Apps",
					Description: "Build userspace applications",
					Command:     "elmos app build",
				},
			},
		},
		{
			Name:     "Run",
			Expanded: true,
			Items: []MenuItem{
				{
					Label:       "Run QEMU",
					Action:      "Run QEMU",
					Description: "Launch kernel in QEMU",
					Command:     "elmos qemu run",
				},
				{
					Label:       "Run QEMU (Debug)",
					Action:      "Run QEMU (Debug Mode)",
					Description: "Launch with GDB stub",
					Command:     "elmos qemu debug",
				},
			},
		},
	}

	m := MenuModel{
		categories:  categories,
		width:       100,
		height:      24,
		outputLines: []OutputLine{},
	}
	m.buildFlatItems()
	m.addOutputLine("Welcome to ELMOS TUI!", panelTitleStyle)
	m.addOutputLine("", outputStyle)
	m.addOutputLine("Navigate with ↑↓, press Enter to execute.", descriptionStyle)
	m.addOutputLine("Press 'c' to clear output, 'q' to quit.", descriptionStyle)
	return m
}

// SetCommandRunner sets the function to run commands
func (m *MenuModel) SetCommandRunner(runner CommandRunner) {
	m.commandRunner = runner
}

func (m *MenuModel) buildFlatItems() {
	m.flatItems = nil
	for catIdx := range m.categories {
		cat := &m.categories[catIdx]
		m.flatItems = append(m.flatItems, FlatItem{
			Category:    cat,
			IsCategory:  true,
			CategoryIdx: catIdx,
		})
		if cat.Expanded {
			for itemIdx := range cat.Items {
				m.flatItems = append(m.flatItems, FlatItem{
					Category:    cat,
					Item:        &cat.Items[itemIdx],
					IsCategory:  false,
					CategoryIdx: catIdx,
					ItemIdx:     itemIdx,
				})
			}
		}
	}
}

func (m *MenuModel) addOutputLine(text string, style lipgloss.Style) {
	m.outputLines = append(m.outputLines, OutputLine{Text: text, Style: style})
	// Auto-scroll to bottom
	maxVisible := m.height - 10
	if len(m.outputLines) > maxVisible {
		m.scrollOffset = len(m.outputLines) - maxVisible
	}
}

func (m *MenuModel) clearOutput() {
	m.outputLines = []OutputLine{}
	m.scrollOffset = 0
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case CommandResultMsg:
		m.isRunning = false
		// Add output lines
		lines := strings.Split(msg.Output, "\n")
		for _, line := range lines {
			if line != "" {
				m.addOutputLine(line, outputStyle)
			}
		}
		if msg.Err != nil {
			m.addOutputLine(fmt.Sprintf("Error: %v", msg.Err), errorMsgStyle)
			// Update status
			m.updateItemStatus(msg.Action, "error")
		} else {
			m.addOutputLine("✓ Completed successfully", successMsgStyle)
			m.updateItemStatus(msg.Action, "ready")
		}
		m.addOutputLine("", outputStyle)
		return m, nil

	case tea.KeyMsg:
		if m.isRunning {
			// Only allow quit while running
			if key.Matches(msg, keys.Quit) {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.flatItems)-1 {
				m.cursor++
			}

		case key.Matches(msg, keys.Clear):
			m.clearOutput()

		case key.Matches(msg, keys.Toggle):
			if m.cursor < len(m.flatItems) {
				item := m.flatItems[m.cursor]
				if item.IsCategory {
					m.categories[item.CategoryIdx].Expanded = !m.categories[item.CategoryIdx].Expanded
					m.buildFlatItems()
					if m.cursor >= len(m.flatItems) {
						m.cursor = len(m.flatItems) - 1
					}
				}
			}

		case key.Matches(msg, keys.Enter):
			if m.cursor < len(m.flatItems) {
				item := m.flatItems[m.cursor]
				if item.IsCategory {
					m.categories[item.CategoryIdx].Expanded = !m.categories[item.CategoryIdx].Expanded
					m.buildFlatItems()
				} else if item.Item != nil {
					// Execute command
					m.isRunning = true
					m.lastAction = item.Item.Action
					m.choice = item.Item.Action
					m.addOutputLine(fmt.Sprintf("▶ Running: %s", item.Item.Label), runningStyle)
					m.addOutputLine(fmt.Sprintf("  $ %s", item.Item.Command), dimStyle)
					m.addOutputLine("", outputStyle)

					// Trigger execution
					return m, m.executeCommandCmd(item.Item.Action)
				}
			}
		}
	}

	return m, nil
}

func (m *MenuModel) executeCommandCmd(action string) tea.Cmd {
	return func() tea.Msg {
		if m.commandRunner == nil {
			return CommandResultMsg{Action: action, Err: fmt.Errorf("no command runner configured")}
		}

		output, err := m.commandRunner(action)
		return CommandResultMsg{Action: action, Output: output, Err: err}
	}
}

func (m *MenuModel) updateItemStatus(action string, status string) {
	for catIdx := range m.categories {
		for itemIdx := range m.categories[catIdx].Items {
			if m.categories[catIdx].Items[itemIdx].Action == action {
				m.categories[catIdx].Items[itemIdx].Status = status
				return
			}
		}
	}
}

func (m MenuModel) View() string {
	if m.quitting {
		return ""
	}

	leftWidth := 40
	rightWidth := m.width - leftWidth - 4
	if rightWidth < 35 {
		rightWidth = 35
	}
	panelHeight := m.height - 3

	// Left panel - Menu
	var leftContent strings.Builder

	leftContent.WriteString(titleStyle.Render("ELMOS"))
	leftContent.WriteString("\n\n")

	for i, flatItem := range m.flatItems {
		isSelected := i == m.cursor

		if flatItem.IsCategory {
			arrow := "▼"
			if !flatItem.Category.Expanded {
				arrow = "▶"
			}
			catText := fmt.Sprintf("%s %s", arrow, flatItem.Category.Name)
			if isSelected {
				leftContent.WriteString(selectedStyle.Render(catText))
			} else {
				leftContent.WriteString(categoryStyle.Render(catText))
			}
			leftContent.WriteString("\n")
		} else if flatItem.Item != nil {
			status := m.renderStatus(flatItem.Item.Status)
			label := flatItem.Item.Label

			maxLen := leftWidth - 14
			if len(label) > maxLen {
				label = label[:maxLen-2] + ".."
			}

			if isSelected {
				line := fmt.Sprintf("▶ %s", label)
				if status != "" {
					padding := leftWidth - 10 - len(line)
					if padding < 1 {
						padding = 1
					}
					line += strings.Repeat(" ", padding) + status
				}
				leftContent.WriteString(selectedStyle.Render(line))
			} else {
				line := fmt.Sprintf("  %s", label)
				if status != "" {
					padding := leftWidth - 10 - len(line)
					if padding < 1 {
						padding = 1
					}
					line += strings.Repeat(" ", padding) + status
				}
				leftContent.WriteString(itemStyle.Render(line))
			}
			leftContent.WriteString("\n")
		}
	}

	// Pad menu
	menuLines := strings.Count(leftContent.String(), "\n")
	for i := menuLines; i < panelHeight-2; i++ {
		leftContent.WriteString("\n")
	}

	// Footer
	if m.isRunning {
		leftContent.WriteString(runningStyle.Render("⏳ Running..."))
	} else {
		leftContent.WriteString(footerStyle.Render("↑↓:Nav ⏎:Run c:Clear q:Quit"))
	}

	// Right panel - Output
	var rightContent strings.Builder

	// Show current selection info at top
	if m.cursor < len(m.flatItems) && !m.flatItems[m.cursor].IsCategory {
		item := m.flatItems[m.cursor].Item
		if item != nil {
			rightContent.WriteString(panelTitleStyle.Render(item.Label))
			rightContent.WriteString("\n")
			rightContent.WriteString(descriptionStyle.Render(item.Description))
			rightContent.WriteString("\n")
			rightContent.WriteString(dimStyle.Render(fmt.Sprintf("$ %s", item.Command)))
			rightContent.WriteString("\n")
			rightContent.WriteString(strings.Repeat("─", rightWidth-4))
			rightContent.WriteString("\n")
		}
	} else {
		rightContent.WriteString(panelTitleStyle.Render("📋 Output"))
		rightContent.WriteString("\n")
		rightContent.WriteString(strings.Repeat("─", rightWidth-4))
		rightContent.WriteString("\n")
	}

	// Output lines
	maxVisible := panelHeight - 8
	start := m.scrollOffset
	end := start + maxVisible
	if end > len(m.outputLines) {
		end = len(m.outputLines)
	}

	for i := start; i < end; i++ {
		line := m.outputLines[i]
		// Truncate long lines
		text := line.Text
		if len(text) > rightWidth-6 {
			text = text[:rightWidth-9] + "..."
		}
		rightContent.WriteString(line.Style.Render(text))
		rightContent.WriteString("\n")
	}

	// Pad output
	outputLines := strings.Count(rightContent.String(), "\n")
	for i := outputLines; i < panelHeight-1; i++ {
		rightContent.WriteString("\n")
	}

	// Scroll indicator
	if len(m.outputLines) > maxVisible {
		rightContent.WriteString(dimStyle.Render(fmt.Sprintf("(%d/%d lines)", end, len(m.outputLines))))
	}

	left := leftPanelStyle.Width(leftWidth).Height(panelHeight).Render(leftContent.String())
	right := rightPanelStyle.Width(rightWidth).Height(panelHeight).Render(rightContent.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m MenuModel) renderStatus(status string) string {
	switch status {
	case "ready":
		return statusReadyStyle.Render("[✓]")
	case "pending":
		return statusPendingStyle.Render("[○]")
	case "error":
		return statusErrorStyle.Render("[✗]")
	default:
		return ""
	}
}

func (m MenuModel) Choice() string {
	return m.choice
}

// RunCommand runs the elmos command and captures output
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(&stdout)
	cmd.Stderr = io.MultiWriter(&stderr)
	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}
	return output, err
}
