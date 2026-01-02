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
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ANSI color palette
var (
	purple      = lipgloss.Color("141")
	cyan        = lipgloss.Color("51")
	green       = lipgloss.Color("120")
	orange      = lipgloss.Color("214")
	red         = lipgloss.Color("203")
	white       = lipgloss.Color("255")
	grey        = lipgloss.Color("245")
	darkGrey    = lipgloss.Color("238")
	borderColor = lipgloss.Color("240")

	leftPanelStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(purple)
	rightPanelStyle   = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(borderColor)
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(purple)
	menuItemStyle     = lipgloss.NewStyle().Foreground(grey)
	selectedItemStyle = lipgloss.NewStyle().Bold(true).Foreground(white).Background(purple)
	hintStyle         = lipgloss.NewStyle().Foreground(cyan).Border(lipgloss.RoundedBorder()).BorderForeground(cyan).Padding(0, 1)
	descStyle         = lipgloss.NewStyle().Foreground(darkGrey).Italic(true)
	inputStyle        = lipgloss.NewStyle().Foreground(white).Background(lipgloss.Color("236")).Padding(0, 1)
	inputLabelStyle   = lipgloss.NewStyle().Foreground(orange).Bold(true)
)

type MenuItem struct {
	Label, Desc, Action, Command string
	Interactive                  bool
	NeedsInput                   bool   // Whether this option needs text input
	InputPrompt                  string // Prompt to show for text input
	InputPlaceholder             string // Placeholder text
	Args                         []string
	Children                     []MenuItem
}

type Model struct {
	menuStack             [][]MenuItem
	currentMenu           []MenuItem
	cursor                int
	parentTitle           string
	viewport              viewport.Model
	logLines              []string
	spinner               spinner.Model
	isRunning             bool
	currentTask           string
	width, height         int
	leftWidth, rightWidth int
	quitting              bool
	execPath              string

	// Text input state
	textInput   textinput.Model
	inputMode   bool
	inputAction string
	inputPrompt string
}

type CommandDoneMsg struct {
	Action string
	Err    error
	Output string
}

type keyMap struct {
	Up, Down, Enter, Back, Quit, Clear     key.Binding
	PageUp, PageDown, ScrollUp, ScrollDown key.Binding
}

var keys = keyMap{
	Up:         key.NewBinding(key.WithKeys("up", "k")),
	Down:       key.NewBinding(key.WithKeys("down", "j")),
	Enter:      key.NewBinding(key.WithKeys("enter")),
	Back:       key.NewBinding(key.WithKeys("esc", "backspace")),
	Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c")),
	Clear:      key.NewBinding(key.WithKeys("c")),
	PageUp:     key.NewBinding(key.WithKeys("[", "ctrl+u")),
	PageDown:   key.NewBinding(key.WithKeys("]", "ctrl+d")),
	ScrollUp:   key.NewBinding(key.WithKeys("{")),
	ScrollDown: key.NewBinding(key.WithKeys("}")),
}

func NewModel() Model {
	exe, _ := os.Executable()
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(orange)

	ti := textinput.New()
	ti.CharLimit = 64
	ti.Width = 40

	m := Model{
		currentMenu: buildMenuStructure(),
		menuStack:   make([][]MenuItem, 0),
		spinner:     s,
		width:       120,
		height:      30,
		leftWidth:   30,
		rightWidth:  90,
		execPath:    exe,
		logLines:    make([]string, 0),
		textInput:   ti,
	}
	m.viewport = viewport.New(60, 20)
	return m
}

func buildMenuStructure() []MenuItem {
	return []MenuItem{
		{Label: "Workspace", Desc: "Initialize and manage workspace", Children: []MenuItem{
			{Label: "Initialize All", Desc: "Create image + mount + setup", Action: "init:workspace", Command: "elmos init"},
			{Label: "Mount Volume", Desc: "Attach the disk image", Action: "init:mount", Command: "elmos init mount"},
			{Label: "Unmount Volume", Desc: "Detach the disk image", Action: "init:unmount", Command: "elmos init unmount"},
			{Label: "Clone Kernel", Desc: "Download Linux source", Action: "init:clone", Command: "elmos init clone"},
		}},
		{Label: "Kernel", Desc: "Configure and build Linux kernel", Children: []MenuItem{
			{Label: "Default Config", Desc: "Standard defconfig", Action: "kernel:defconfig", Command: "elmos kernel config"},
			{Label: "Tiny Config", Desc: "Minimal kernel", Action: "kernel:tinyconfig", Command: "elmos kernel config tinyconfig"},
			{Label: "KVM Guest", Desc: "Optimized for VMs", Action: "kernel:kvmconfig", Command: "elmos kernel config kvm_guest.config"},
			{Label: "Menu Config", Desc: "Interactive ncurses", Action: "kernel:menuconfig", Command: "elmos kernel config menuconfig", Interactive: true, Args: []string{"kernel", "config", "menuconfig"}},
			{Label: "Build", Desc: "Compile the kernel", Action: "kernel:build", Command: "elmos build"},
			{Label: "Clean", Desc: "Remove build artifacts", Action: "kernel:clean", Command: "elmos kernel clean"},
		}},
		{Label: "Modules", Desc: "Manage kernel modules", Children: []MenuItem{
			{Label: "List", Desc: "Show available modules", Action: "module:list", Command: "elmos module list"},
			{Label: "Build All", Desc: "Compile all modules", Action: "module:build", Command: "elmos module build"},
			{Label: "Build One", Desc: "Compile specific module", Action: "module:build:one", Command: "elmos module build <name>", NeedsInput: true, InputPrompt: "Module name:", InputPlaceholder: "my_module"},
			{Label: "Create New", Desc: "Generate module template", Action: "module:new", Command: "elmos module new <name>", NeedsInput: true, InputPrompt: "New module name:", InputPlaceholder: "hello_world"},
			{Label: "Clean", Desc: "Remove module binaries", Action: "module:clean", Command: "elmos module clean"},
		}},
		{Label: "Apps", Desc: "Manage userspace applications", Children: []MenuItem{
			{Label: "List", Desc: "Show available apps", Action: "app:list", Command: "elmos app list"},
			{Label: "Build All", Desc: "Compile all apps", Action: "app:build", Command: "elmos app build"},
			{Label: "Build One", Desc: "Compile specific app", Action: "app:build:one", Command: "elmos app build <name>", NeedsInput: true, InputPrompt: "App name:", InputPlaceholder: "my_app"},
			{Label: "Create New", Desc: "Generate app template", Action: "app:new", Command: "elmos app new <name>", NeedsInput: true, InputPrompt: "New app name:", InputPlaceholder: "hello_app"},
			{Label: "Clean", Desc: "Remove app binaries", Action: "app:clean", Command: "elmos app clean"},
		}},
		{Label: "QEMU", Desc: "Run kernel in emulator", Children: []MenuItem{
			{Label: "Run (Console)", Desc: "Boot in terminal", Action: "qemu:run", Command: "elmos qemu run", Interactive: true, Args: []string{"qemu", "run"}},
			{Label: "Run (GUI)", Desc: "Boot with display", Action: "qemu:graphical", Command: "elmos qemu run -g", Interactive: true, Args: []string{"qemu", "run", "-g"}},
			{Label: "Debug Mode", Desc: "Start GDB server", Action: "qemu:debug", Command: "elmos qemu debug", Interactive: true, Args: []string{"qemu", "debug"}},
			{Label: "Connect GDB", Desc: "Attach debugger", Action: "qemu:gdb", Command: "elmos qemu gdb", Interactive: true, Args: []string{"qemu", "gdb"}},
		}},
		{Label: "RootFS", Desc: "Create root filesystem", Children: []MenuItem{
			{Label: "Create (5G)", Desc: "Default size rootfs", Action: "rootfs:create", Command: "elmos rootfs create"},
			{Label: "Create Custom", Desc: "Specify size (e.g. 10G)", Action: "rootfs:create:custom", Command: "elmos rootfs create -s <size>", NeedsInput: true, InputPrompt: "Size (e.g. 10G):", InputPlaceholder: "10G"},
		}},
		{Label: "Config", Desc: "Manage settings", Children: []MenuItem{
			{Label: "Show", Desc: "Display current settings", Action: "config:show", Command: "elmos config show"},
			{Label: "Set Architecture", Desc: "arm64, arm, riscv", Action: "config:arch", Command: "elmos config set arch <arch>", NeedsInput: true, InputPrompt: "Architecture (arm64/arm/riscv):", InputPlaceholder: "arm64"},
			{Label: "Set Build Jobs", Desc: "Parallel compilation", Action: "config:jobs", Command: "elmos config set jobs <n>", NeedsInput: true, InputPrompt: "Number of jobs:", InputPlaceholder: "8"},
			{Label: "Set QEMU Memory", Desc: "Memory allocation", Action: "config:memory", Command: "elmos config set memory <size>", NeedsInput: true, InputPrompt: "Memory (e.g. 2G):", InputPlaceholder: "2G"},
		}},
		{Label: "Doctor", Desc: "Check environment", Action: "doctor:check", Command: "elmos doctor"},
	}
}

func (m *Model) refreshViewport() {
	m.viewport.SetContent(strings.Join(m.logLines, "\n"))
	m.viewport.GotoBottom()
}

func (m Model) Init() tea.Cmd { return m.spinner.Tick }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// If in input mode, handle text input
	if m.inputMode {
		return m.handleInputMode(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.leftWidth = maxInt(25, int(float64(m.width)*0.30))
		m.rightWidth = m.width - m.leftWidth - 3
		m.viewport.Width = m.rightWidth - 4
		m.viewport.Height = m.height - 10
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
				m.logLines = append(m.logLines, "  "+line)
			}
		}
		if msg.Err != nil {
			m.logLines = append(m.logLines, lipgloss.NewStyle().Foreground(red).Render(fmt.Sprintf("  ✗ Error: %v", msg.Err)))
		} else {
			m.logLines = append(m.logLines, lipgloss.NewStyle().Foreground(green).Render("  ✓ Completed"))
		}
		m.logLines = append(m.logLines, "")
		m.refreshViewport()
		m.currentTask = ""

	case tea.KeyMsg:
		if m.isRunning && !key.Matches(msg, keys.Quit) {
			return m, tea.Batch(cmds...)
		}

		switch {
		case key.Matches(msg, keys.Quit):
			if len(m.menuStack) > 0 {
				m.currentMenu = m.menuStack[len(m.menuStack)-1]
				m.menuStack = m.menuStack[:len(m.menuStack)-1]
				m.cursor, m.parentTitle = 0, ""
			} else {
				m.quitting = true
				return m, tea.Quit
			}
		case key.Matches(msg, keys.Back):
			if len(m.menuStack) > 0 {
				m.currentMenu = m.menuStack[len(m.menuStack)-1]
				m.menuStack = m.menuStack[:len(m.menuStack)-1]
				m.cursor, m.parentTitle = 0, ""
			}
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.currentMenu)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.PageUp):
			m.viewport.PageUp()
		case key.Matches(msg, keys.PageDown):
			m.viewport.PageDown()
		case key.Matches(msg, keys.ScrollUp):
			m.viewport.ScrollUp(1)
		case key.Matches(msg, keys.ScrollDown):
			m.viewport.ScrollDown(1)
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

				// If item needs input, enter input mode
				if item.NeedsInput {
					m.inputMode = true
					m.inputAction = item.Action
					m.inputPrompt = item.InputPrompt
					m.textInput.Placeholder = item.InputPlaceholder
					m.textInput.SetValue("")
					m.textInput.Focus()
					return m, textinput.Blink
				}

				if item.Interactive {
					m.logLines = append(m.logLines, lipgloss.NewStyle().Foreground(cyan).Render("  ▶ "+item.Command))
					m.refreshViewport()
					c := exec.Command(m.execPath, item.Args...)
					c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
					return m, tea.ExecProcess(c, func(err error) tea.Msg {
						return CommandDoneMsg{Action: item.Action, Err: err}
					})
				}
				if item.Action != "" {
					m.isRunning = true
					m.currentTask = item.Label
					m.logLines = append(m.logLines, lipgloss.NewStyle().Foreground(cyan).Render("  ▶ "+item.Command))
					m.refreshViewport()
					return m, m.runCommand(item.Action, "")
				}
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m Model) handleInputMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.inputMode = false
			m.textInput.Blur()
			return m, nil
		case "enter":
			value := strings.TrimSpace(m.textInput.Value())
			if value == "" {
				return m, nil
			}
			m.inputMode = false
			m.textInput.Blur()

			// Build command with the input value
			cmdStr := m.getCommandWithInput(m.inputAction, value)
			m.logLines = append(m.logLines, lipgloss.NewStyle().Foreground(cyan).Render("  ▶ "+cmdStr))
			m.refreshViewport()

			m.isRunning = true
			m.currentTask = m.inputPrompt + " " + value
			return m, m.runCommand(m.inputAction, value)
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *Model) getCommandWithInput(action, value string) string {
	switch action {
	case "module:new":
		return "elmos module new " + value
	case "module:build:one":
		return "elmos module build " + value
	case "app:new":
		return "elmos app new " + value
	case "app:build:one":
		return "elmos app build " + value
	case "config:arch":
		return "elmos config set arch " + value
	case "config:jobs":
		return "elmos config set jobs " + value
	case "config:memory":
		return "elmos config set memory " + value
	case "rootfs:create:custom":
		return "elmos rootfs create -s " + value
	default:
		return "elmos " + action
	}
}

func (m *Model) runCommand(action, inputValue string) tea.Cmd {
	return func() tea.Msg {
		args := m.actionToArgs(action, inputValue)
		cmd := exec.Command(m.execPath, args...)
		var output bytes.Buffer
		cmd.Stdout, cmd.Stderr = &output, &output
		err := cmd.Run()
		return CommandDoneMsg{Action: action, Err: err, Output: output.String()}
	}
}

func (m *Model) actionToArgs(action, inputValue string) []string {
	switch action {
	case "init:workspace":
		return []string{"init"}
	case "init:mount":
		return []string{"init", "mount"}
	case "init:unmount":
		return []string{"init", "unmount"}
	case "init:clone":
		return []string{"init", "clone"}
	case "kernel:defconfig":
		return []string{"kernel", "config"}
	case "kernel:tinyconfig":
		return []string{"kernel", "config", "tinyconfig"}
	case "kernel:kvmconfig":
		return []string{"kernel", "config", "kvm_guest.config"}
	case "kernel:build":
		return []string{"build"}
	case "kernel:clean":
		return []string{"kernel", "clean"}
	case "module:list":
		return []string{"module", "list"}
	case "module:build":
		return []string{"module", "build"}
	case "module:build:one":
		return []string{"module", "build", inputValue}
	case "module:new":
		return []string{"module", "new", inputValue}
	case "module:clean":
		return []string{"module", "clean"}
	case "app:list":
		return []string{"app", "list"}
	case "app:build":
		return []string{"app", "build"}
	case "app:build:one":
		return []string{"app", "build", inputValue}
	case "app:new":
		return []string{"app", "new", inputValue}
	case "app:clean":
		return []string{"app", "clean"}
	case "rootfs:create":
		return []string{"rootfs", "create"}
	case "rootfs:create:custom":
		return []string{"rootfs", "create", "-s", inputValue}
	case "config:show":
		return []string{"config", "show"}
	case "config:arch":
		return []string{"config", "set", "arch", inputValue}
	case "config:jobs":
		return []string{"config", "set", "jobs", inputValue}
	case "config:memory":
		return []string{"config", "set", "memory", inputValue}
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

	panelHeight := m.height - 2

	// LEFT PANEL
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
		var prefix string
		if len(item.Children) > 0 {
			prefix = "▸ "
		} else if item.Interactive {
			prefix = "⚡"
		} else if item.NeedsInput {
			prefix = "✎ "
		} else {
			prefix = "• "
		}
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

	// RIGHT PANEL
	var right strings.Builder
	rightTitle := "Output"
	if m.isRunning {
		rightTitle = m.spinner.View() + " " + m.currentTask
	} else if m.inputMode {
		rightTitle = "📝 Input Required"
	}
	scrollInfo := ""
	if m.viewport.TotalLineCount() > m.viewport.Height {
		scrollInfo = fmt.Sprintf(" [%d%%]", int(m.viewport.ScrollPercent()*100))
	}
	right.WriteString(titleStyle.Render("─ "+rightTitle+scrollInfo+" ─") + "\n\n")

	// Show input field if in input mode
	if m.inputMode {
		right.WriteString(inputLabelStyle.Render(m.inputPrompt) + "\n\n")
		right.WriteString(inputStyle.Render(m.textInput.View()) + "\n\n")
		right.WriteString(descStyle.Render("  Press Enter to confirm, Esc to cancel") + "\n\n")
	} else if m.cursor < len(m.currentMenu) && !m.isRunning {
		item := m.currentMenu[m.cursor]
		if item.Command != "" {
			right.WriteString(hintStyle.Render(" $ "+item.Command+" ") + "\n")
			if item.Desc != "" {
				right.WriteString(descStyle.Render("  "+item.Desc) + "\n")
			}
			if item.NeedsInput {
				right.WriteString("\n" + inputLabelStyle.Render("  ✎ Press Enter to type: "+item.InputPrompt) + "\n")
			}
		} else if len(item.Children) > 0 {
			right.WriteString(descStyle.Render("  Press Enter to expand → "+item.Desc) + "\n")
		}
		right.WriteString("\n")
	}
	right.WriteString(m.viewport.View())

	leftPanel := leftPanelStyle.Width(m.leftWidth).Height(panelHeight).Render(left.String())
	rightPanel := rightPanelStyle.Width(m.rightWidth).Height(panelHeight).Render(right.String())
	main := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	footer := lipgloss.NewStyle().Foreground(darkGrey).Render(
		lipgloss.NewStyle().Foreground(cyan).Render("↑↓") + " Navigate  " +
			lipgloss.NewStyle().Foreground(cyan).Render("⏎") + " Select  " +
			lipgloss.NewStyle().Foreground(cyan).Render("Esc") + " Back  " +
			lipgloss.NewStyle().Foreground(cyan).Render("[ ]") + " Scroll  " +
			lipgloss.NewStyle().Foreground(cyan).Render("c") + " Clear  " +
			lipgloss.NewStyle().Foreground(cyan).Render("q") + " Quit")

	return lipgloss.JoinVertical(lipgloss.Left, main, footer)
}

type CommandRunner func(action string, output io.Writer) error

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
