// Package tui provides the interactive Text User Interface for elmos.
// This file contains the model initialization, menu structure, and entry point.
package tui

import (
	"io"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// NewModel creates a new TUI Model with default settings.
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

// buildMenuStructure returns the complete menu tree for the TUI.
func buildMenuStructure() []MenuItem {
	return []MenuItem{
		{Label: "Workspace", Desc: "Initialize and manage workspace", Children: []MenuItem{
			{Label: "Initialize", Desc: "Create image & mount", Action: "init:workspace", Command: "elmos init", Args: []string{"init"}},
			{Label: "Status", Desc: "Show workspace status", Action: "workspace:status", Command: "elmos status", Args: []string{"status"}},
			{Label: "Exit", Desc: "Unmount & cleanup", Action: "workspace:exit", Command: "elmos exit", Args: []string{"exit"}},
		}},
		{Label: "Arch", Desc: "Set target architecture", Children: []MenuItem{
			{Label: "Show", Desc: "Show current config", Action: "arch:show", Command: "elmos arch show", Args: []string{"arch", "show"}},
			{Label: "Set", Desc: "Set architecture", Action: "arch:set", Command: "elmos arch <target>", NeedsInput: true, InputPrompt: "Architecture (arm64/arm/riscv):", InputPlaceholder: "arm64"},
		}},
		{Label: "Kernel", Desc: "Configure and build Linux kernel", Children: []MenuItem{
			{Label: "Status", Desc: "Show kernel status", Action: "kernel:status", Command: "elmos kernel status", Args: []string{"kernel", "status"}},
			{Label: "Clone", Desc: "Download source", Action: "kernel:clone", Command: "elmos kernel clone", Args: []string{"kernel", "clone"}},
			{Label: "Pull", Desc: "Update source", Action: "kernel:pull", Command: "elmos kernel pull", Args: []string{"kernel", "pull"}},
			{Label: "Switch", Desc: "Checkout ref", Action: "kernel:switch", Command: "elmos kernel switch <ref>", NeedsInput: true, InputPrompt: "Branch/Tag (? for list):", InputPlaceholder: "v6.7"},
			{Label: "Reset", Desc: "Reclone source", Action: "kernel:reset", Command: "elmos kernel reset", Args: []string{"kernel", "reset"}},
			{Label: "Config", Desc: "Configure kernel", Action: "kernel:config", Command: "elmos kernel config <type>", NeedsInput: true, InputPrompt: "Config (defconfig/tinyconfig/menuconfig):", InputPlaceholder: "defconfig"},
			{Label: "Build", Desc: "Compile kernel", Action: "kernel:build", Command: "elmos kernel build", Args: []string{"kernel", "build"}},
			{Label: "Install", Desc: "Link build artifacts", Action: "kernel:install", Command: "elmos kernel install", Args: []string{"kernel", "install"}},
			{Label: "Clean", Desc: "Remove artifacts", Action: "kernel:clean", Command: "elmos kernel clean", Args: []string{"kernel", "clean"}},
		}},
		{Label: "Modules", Desc: "Manage kernel modules", Children: []MenuItem{
			{Label: "List", Desc: "Show modules", Action: "module:list", Command: "elmos module list", Args: []string{"module", "list"}},
			{Label: "Build", Desc: "Build module(s)", Action: "module:build", Command: "elmos module build [name]", NeedsInput: true, InputPrompt: "Module (blank=all):", InputPlaceholder: ""},
			{Label: "New", Desc: "Create module", Action: "module:new", Command: "elmos module new <name>", NeedsInput: true, InputPrompt: "Module name:", InputPlaceholder: "hello_world"},
			{Label: "Clean", Desc: "Remove binaries", Action: "module:clean", Command: "elmos module clean", Args: []string{"module", "clean"}},
		}},
		{Label: "Apps", Desc: "Manage userspace apps", Children: []MenuItem{
			{Label: "List", Desc: "Show apps", Action: "app:list", Command: "elmos app list", Args: []string{"app", "list"}},
			{Label: "Build", Desc: "Build app(s)", Action: "app:build", Command: "elmos app build [name]", NeedsInput: true, InputPrompt: "App (blank=all):", InputPlaceholder: ""},
			{Label: "New", Desc: "Create app", Action: "app:new", Command: "elmos app new <name>", NeedsInput: true, InputPrompt: "App name:", InputPlaceholder: "hello_app"},
			{Label: "Clean", Desc: "Remove binaries", Action: "app:clean", Command: "elmos app clean", Args: []string{"app", "clean"}},
		}},
		{Label: "QEMU", Desc: "Run kernel in emulator", Children: []MenuItem{
			{Label: "Run", Desc: "Boot kernel", Action: "qemu:run", Command: "elmos qemu run", Interactive: true, Args: []string{"qemu", "run"}},
			{Label: "Debug", Desc: "With GDB server", Action: "qemu:debug", Command: "elmos qemu debug", Interactive: true, Args: []string{"qemu", "debug"}},
		}},
		{Label: "GDB", Desc: "Connect debugger", Action: "gdb:connect", Command: "elmos gdb", Args: []string{"gdb"}},
		{Label: "RootFS", Desc: "Manage root filesystem", Children: []MenuItem{
			{Label: "Status", Desc: "Show rootfs status", Action: "rootfs:status", Command: "elmos rootfs status", Args: []string{"rootfs", "status"}},
			{Label: "Build", Desc: "Build rootfs", Action: "rootfs:build", Command: "elmos rootfs build -s <size>", NeedsInput: true, InputPrompt: "Size (e.g. 5G):", InputPlaceholder: "5G"},
			{Label: "Clean", Desc: "Remove rootfs", Action: "rootfs:clean", Command: "elmos rootfs clean", Args: []string{"rootfs", "clean"}},
		}},
		{Label: "Bootloader", Desc: "Build and install U-Boot", Children: []MenuItem{
			{Label: "Build", Desc: "Build bootloader", Action: "bootloader:build", Command: "elmos bootloader build", Args: []string{"bootloader", "build"}},
			{Label: "Install", Desc: "Link bootloader artifacts", Action: "bootloader:install", Command: "elmos bootloader install", Args: []string{"bootloader", "install"}},
			{Label: "Blobs", Desc: "List/download firmware blobs", Action: "bootloader:blobs", Command: "elmos bootloader blobs", Args: []string{"bootloader", "blobs"}},
		}},
		{Label: "Doctor", Desc: "Check environment", Action: "doctor:check", Command: "elmos doctor", Args: []string{"doctor"}},
		{Label: "Toolchains", Desc: "Manage cross-compiler toolchains", Children: []MenuItem{
			{Label: "Status", Desc: "Show installed toolchains", Action: "toolchain:status", Command: "elmos toolchains status", Args: []string{"toolchains", "status"}},
			{Label: "Clone", Desc: "Install crosstool-ng", Action: "toolchain:clone", Command: "elmos toolchains clone", Args: []string{"toolchains", "clone"}},
			{Label: "List", Desc: "List available targets", Action: "toolchain:list", Command: "elmos toolchains list", Args: []string{"toolchains", "list"}},
			{Label: "Select", Desc: "Select toolchain target", Action: "toolchain:select", Command: "elmos toolchains <target>", NeedsInput: true, InputPrompt: "Target (e.g. riscv64-unknown-linux-gnu):", InputPlaceholder: "riscv64-unknown-linux-gnu"},
			{Label: "Build", Desc: "Build selected toolchain", Action: "toolchain:build", Command: "elmos toolchains build", Args: []string{"toolchains", "build"}},
			{Label: "Install", Desc: "Link selected toolchain", Action: "toolchain:install", Command: "elmos toolchains install", Args: []string{"toolchains", "install"}},
			{Label: "Env", Desc: "Show env variables", Action: "toolchain:env", Command: "elmos toolchains env", Args: []string{"toolchains", "env"}},
			{Label: "Menuconfig", Desc: "Configure toolchain", Action: "toolchain:menuconfig", Command: "elmos toolchains menuconfig", Interactive: true, Args: []string{"toolchains", "menuconfig"}},
			{Label: "Clean", Desc: "Clean toolchain build", Action: "toolchain:clean", Command: "elmos toolchains clean", Args: []string{"toolchains", "clean"}},
		}},
	}
}

// CommandRunner is a function type for running commands.
type CommandRunner func(action string, output io.Writer) error

// Run starts the TUI application.
func Run() error {
	// Ensure TERM is set for WSL2 and minimal environments where it may be absent.
	// Without TERM, bubbletea cannot send the ANSI sequences needed for alt-screen.
	// Best-effort: failure to set TERM is non-fatal — bubbletea may degrade gracefully.
	if os.Getenv("TERM") == "" {
		_ = os.Setenv("TERM", "xterm-256color")
	}

	m := NewModel()
	// tea.WithAltScreen() switches to the alternate screen buffer (cleaner UX).
	// tea.WithMouseCellMotion() enables cell-level mouse events for scrolling.
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}

// maxInt returns the larger of two integers.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
