// Package tui provides the interactive Text User Interface for elmos.
package tui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Theme colors - Dark kernel-like theme
var (
	primaryBlue   = lipgloss.Color("33")  // Blue
	highlightBlue = lipgloss.Color("39")  // Cyan
	darkGrey      = lipgloss.Color("235") // Dark bg
	midGrey       = lipgloss.Color("240") // Border
	lightGrey     = lipgloss.Color("250") // Text
	successGreen  = lipgloss.Color("40")  // Green
	warningYellow = lipgloss.Color("220") // Yellow
	errorRed      = lipgloss.Color("196") // Red
	dimText       = lipgloss.Color("245") // Dimmed

	leftPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryBlue)

	rightPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(midGrey)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")). // White
			Background(primaryBlue).
			Padding(0, 1)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(lightGrey)

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("16")). // Black text
				Background(highlightBlue)

	helpStyle = lipgloss.NewStyle().
			Foreground(dimText)
)

// MenuItem represents a menu item.
type MenuItem struct {
	Label       string
	Action      string
	Command     string
	Interactive bool
	Args        []string
	Children    []MenuItem
}

// MenuCategory represents a top-level menu category.
type MenuCategory struct {
	Name  string
	Icon  string
	Items []MenuItem
}

// Model holds the TUI state.
type Model struct {
	categories  []MenuCategory
	menuStack   [][]MenuItem
	currentMenu []MenuItem
	cursor      int
	parentTitle string

	viewport    viewport.Model
	logLines    []string
	spinner     spinner.Model
	isRunning   bool
	currentTask string

	width, height         int
	leftWidth, rightWidth int

	quitting bool
	execPath string
}

// CommandDoneMsg signals command completion.
type CommandDoneMsg struct {
	Action string
	Err    error
	Output string
}

type keyMap struct {
	Up, Down, Enter, Back, Quit, Clear key.Binding
}

var keys = keyMap{
	Up:    key.NewBinding(key.WithKeys("up", "k")),
	Down:  key.NewBinding(key.WithKeys("down", "j")),
	Enter: key.NewBinding(key.WithKeys("enter")),
	Back:  key.NewBinding(key.WithKeys("esc", "backspace")),
	Quit:  key.NewBinding(key.WithKeys("q", "ctrl+c")),
	Clear: key.NewBinding(key.WithKeys("c")),
}

// NewModel creates a new TUI model.
func NewModel() Model {
	exe, _ := os.Executable()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(warningYellow)

	categories := buildMenuStructure()

	var topLevel []MenuItem
	for _, cat := range categories {
		topLevel = append(topLevel, MenuItem{
			Label:    cat.Icon + " " + cat.Name,
			Children: cat.Items,
		})
	}

	m := Model{
		categories:  categories,
		currentMenu: topLevel,
		menuStack:   make([][]MenuItem, 0),
		cursor:      0,
		spinner:     s,
		width:       120,
		height:      30,
		leftWidth:   42,
		rightWidth:  78,
		execPath:    exe,
		logLines:    make([]string, 0),
	}

	m.viewport = viewport.New(60, 20)

	// Initial welcome message - plain text, no backgrounds
	m.logLines = append(m.logLines,
		colorText("╔══════════════════════════════════════════════════════════╗", primaryBlue),
		colorText("║            ELMOS - Embedded Linux on MacOS               ║", primaryBlue),
		colorText("║            Professional Kernel Build System              ║", primaryBlue),
		colorText("╚══════════════════════════════════════════════════════════╝", primaryBlue),
		"",
		colorText("Navigate: ↑↓  Select: Enter  Back: Esc  Clear: c  Quit: q", dimText),
		"",
	)
	m.refreshViewport()

	return m
}

func buildMenuStructure() []MenuCategory {
	return []MenuCategory{
		{Name: "Initialization", Icon: "🚀", Items: []MenuItem{
			{Label: "Initialize Workspace", Action: "init:workspace", Command: "elmos init"},
			{Label: "Mount Volume", Action: "init:mount", Command: "elmos init mount"},
			{Label: "Clone Kernel", Action: "init:clone", Command: "elmos init clone"},
		}},
		{Name: "Kernel", Icon: "🐧", Items: []MenuItem{
			{Label: "Default Config", Action: "kernel:defconfig", Command: "elmos kernel config"},
			{Label: "Menu Config", Action: "kernel:menuconfig", Command: "elmos kernel config menuconfig", Interactive: true, Args: []string{"kernel", "config", "menuconfig"}},
			{Label: "Build", Action: "kernel:build", Command: "elmos build"},
			{Label: "Clean", Action: "kernel:clean", Command: "elmos kernel clean"},
		}},
		{Name: "Modules", Icon: "📦", Items: []MenuItem{
			{Label: "List", Action: "module:list", Command: "elmos module list"},
			{Label: "Build All", Action: "module:build", Command: "elmos module build"},
		}},
		{Name: "Apps", Icon: "📱", Items: []MenuItem{
			{Label: "List", Action: "app:list", Command: "elmos app list"},
			{Label: "Build All", Action: "app:build", Command: "elmos app build"},
		}},
		{Name: "QEMU", Icon: "🖥", Items: []MenuItem{
			{Label: "Run", Action: "qemu:run", Command: "elmos qemu run", Interactive: true, Args: []string{"qemu", "run"}},
			{Label: "Debug", Action: "qemu:debug", Command: "elmos qemu debug", Interactive: true, Args: []string{"qemu", "debug"}},
		}},
		{Name: "RootFS", Icon: "💾", Items: []MenuItem{
			{Label: "Create", Action: "rootfs:create", Command: "elmos rootfs create"},
		}},
		{Name: "Doctor", Icon: "🩺", Items: []MenuItem{
			{Label: "Check Env", Action: "doctor:check", Command: "elmos doctor"},
		}},
	}
}

// colorText applies foreground color only (no background)
func colorText(text string, color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(color).Render(text)
}

// refreshViewport updates the viewport content
func (m *Model) refreshViewport() {
	m.viewport.SetContent(strings.Join(m.logLines, "\n"))
	m.viewport.GotoBottom()
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.leftWidth = maxInt(20, int(float64(m.width)*0.35))
		m.rightWidth = maxInt(20, m.width-m.leftWidth-4)
		m.viewport.Width = m.rightWidth - 4
		m.viewport.Height = m.height - 8
		m.refreshViewport()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case CommandDoneMsg:
		m.isRunning = false
		if msg.Output != "" {
			for _, line := range strings.Split(strings.TrimSpace(msg.Output), "\n") {
				m.logLines = append(m.logLines, colorText("  "+line, lightGrey))
			}
		}
		if msg.Err != nil {
			m.logLines = append(m.logLines, colorText(fmt.Sprintf("✗ Error: %v", msg.Err), errorRed))
		} else {
			m.logLines = append(m.logLines, colorText("✓ Completed", successGreen))
		}
		m.logLines = append(m.logLines, "")
		m.refreshViewport()
		m.currentTask = ""

	case tea.KeyMsg:
		if m.isRunning {
			if key.Matches(msg, keys.Quit) {
				m.quitting = true
				return m, tea.Quit
			}
			return m, tea.Batch(cmds...)
		}

		switch {
		case key.Matches(msg, keys.Quit):
			if len(m.menuStack) > 0 {
				m.currentMenu = m.menuStack[len(m.menuStack)-1]
				m.menuStack = m.menuStack[:len(m.menuStack)-1]
				m.cursor = 0
				m.parentTitle = ""
			} else {
				m.quitting = true
				return m, tea.Quit
			}

		case key.Matches(msg, keys.Back):
			if len(m.menuStack) > 0 {
				m.currentMenu = m.menuStack[len(m.menuStack)-1]
				m.menuStack = m.menuStack[:len(m.menuStack)-1]
				m.cursor = 0
				m.parentTitle = ""
			}

		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			// Don't update viewport - only menu cursor changed
			return m, tea.Batch(cmds...)

		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.currentMenu)-1 {
				m.cursor++
			}
			// Don't update viewport - only menu cursor changed
			return m, tea.Batch(cmds...)

		case key.Matches(msg, keys.Clear):
			m.logLines = make([]string, 0)
			m.refreshViewport()

		case key.Matches(msg, keys.Enter):
			if m.cursor < len(m.currentMenu) {
				item := m.currentMenu[m.cursor]

				if len(item.Children) > 0 {
					m.menuStack = append(m.menuStack, m.currentMenu)
					m.parentTitle = item.Label
					m.currentMenu = item.Children
					m.cursor = 0
					return m, nil
				}

				if item.Interactive {
					m.logLines = append(m.logLines, colorText(fmt.Sprintf("→ Launching: %s", item.Command), primaryBlue))
					m.refreshViewport()
					c := exec.Command(m.execPath, item.Args...)
					c.Stdin = os.Stdin
					c.Stdout = os.Stdout
					c.Stderr = os.Stderr
					return m, tea.ExecProcess(c, func(err error) tea.Msg {
						return CommandDoneMsg{Action: item.Action, Err: err}
					})
				}

				if item.Action != "" {
					m.isRunning = true
					m.currentTask = item.Label
					m.logLines = append(m.logLines, colorText(fmt.Sprintf("→ Running: %s", item.Command), primaryBlue))
					m.refreshViewport()
					return m, m.runCommand(item)
				}
			}
		}
	}

	// Only pass non-key messages to viewport (for scrolling via mouse etc)
	// This prevents arrow keys from scrolling the viewport
	if _, isKey := msg.(tea.KeyMsg); !isKey {
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) runCommand(item MenuItem) tea.Cmd {
	return func() tea.Msg {
		args := m.actionToArgs(item.Action)
		cmd := exec.Command(m.execPath, args...)
		var output bytes.Buffer
		cmd.Stdout = &output
		cmd.Stderr = &output
		err := cmd.Run()
		return CommandDoneMsg{Action: item.Action, Err: err, Output: output.String()}
	}
}

func (m *Model) actionToArgs(action string) []string {
	switch action {
	case "init:workspace":
		return []string{"init"}
	case "init:mount":
		return []string{"init", "mount"}
	case "init:clone":
		return []string{"init", "clone"}
	case "kernel:defconfig":
		return []string{"kernel", "config"}
	case "kernel:build":
		return []string{"build"}
	case "kernel:clean":
		return []string{"kernel", "clean"}
	case "module:list":
		return []string{"module", "list"}
	case "module:build":
		return []string{"module", "build"}
	case "app:list":
		return []string{"app", "list"}
	case "app:build":
		return []string{"app", "build"}
	case "rootfs:create":
		return []string{"rootfs", "create"}
	case "doctor:check":
		return []string{"doctor"}
	default:
		return []string{}
	}
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	panelHeight := maxInt(10, m.height-3)

	// LEFT PANEL - Menu
	var left strings.Builder
	title := "ELMOS"
	if m.parentTitle != "" {
		title = m.parentTitle
	}
	left.WriteString(titleStyle.Render(" " + title + " "))
	left.WriteString("\n\n")

	if len(m.menuStack) > 0 {
		left.WriteString(colorText("  ← Back (Esc)", dimText))
		left.WriteString("\n\n")
	}

	for i, item := range m.currentMenu {
		prefix := "  "
		if len(item.Children) > 0 {
			prefix = "▸ "
		} else if item.Interactive {
			prefix = "⌨ "
		} else if item.Action != "" {
			prefix = "• "
		}

		label := prefix + item.Label
		maxLen := maxInt(8, m.leftWidth-6)
		if len(label) > maxLen {
			label = label[:maxLen-2] + ".."
		}

		if i == m.cursor {
			left.WriteString(selectedItemStyle.Render(" " + label + " "))
		} else {
			left.WriteString(menuItemStyle.Render(label))
		}
		left.WriteString("\n")
	}

	// Padding
	for i := strings.Count(left.String(), "\n"); i < panelHeight-4; i++ {
		left.WriteString("\n")
	}

	// RIGHT PANEL - Output
	var right strings.Builder
	header := "📋 Output"
	if m.isRunning {
		header = m.spinner.View() + " " + m.currentTask
	}
	right.WriteString(titleStyle.Render(" " + header + " "))
	right.WriteString("\n\n")
	right.WriteString(m.viewport.View())

	// COMBINE
	leftPanel := leftPanelStyle.Width(m.leftWidth).Height(panelHeight).Render(left.String())
	rightPanel := rightPanelStyle.Width(m.rightWidth).Height(panelHeight).Render(right.String())
	main := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// FOOTER
	footer := helpStyle.Render(" ↑/↓: Navigate │ ⏎: Select │ Esc: Back │ c: Clear │ q: Quit ")

	return lipgloss.JoinVertical(lipgloss.Left, main, footer)
}

// CommandRunner for future extension.
type CommandRunner func(action string, output io.Writer) error

// Run starts the TUI application.
func Run() error {
	m := NewModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
