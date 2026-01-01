// Package cmd implements the Cobra CLI commands for elmos.
package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/NguyenTrongPhuc552003/elmos/internal/tui"
)

// uiCmd - Interactive TUI
var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch interactive menu (TUI)",
	Long: `Launch an interactive menu-driven interface for ELMOS.

The TUI provides a split-pane view:
  - Left panel: Menu navigation
  - Right panel: Command output and descriptions

Navigation:
  ↑/k, ↓/j  Move selection
  Enter     Execute selected action
  Tab       Toggle category expand/collapse
  c         Clear output panel
  q         Quit`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}

// runTUI runs the interactive TUI
func runTUI() error {
	m := tui.NewMenuModel()
	m.SetCommandRunner(executeAction)

	p := tea.NewProgram(m, tea.WithAltScreen())

	_, err := p.Run()
	return err
}

// executeAction runs the action and returns captured output
func executeAction(choice string) (string, error) {
	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	// Run in goroutine to capture output
	done := make(chan error)
	go func() {
		defer close(done)
		done <- runAction(choice)
	}()

	// Wait for completion
	err := <-done

	// Restore stdout/stderr
	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	io.Copy(&buf, r)

	return buf.String(), err
}

// runAction dispatches to the appropriate command handler
func runAction(choice string) error {
	printStep("Executing: %s", choice)

	switch choice {
	case "Doctor (Check Environment)":
		return runDoctor()
	case "Init Workspace":
		if err := runImageMount(); err != nil {
			return err
		}
		return runRepoCheck()
	case "Configure (Arch, Jobs...)":
		return RunConfigShow()
	case "Kernel Config (defconfig)":
		return runKernelConfig("defconfig")
	case "Kernel Menuconfig (UI)":
		return runKernelConfig("menuconfig")
	case "Build Kernel":
		return runBuild(ctx.Config.Build.Jobs, []string{"Image", "dtbs", "modules"})
	case "Build Modules":
		return runModuleBuild("")
	case "Build Apps":
		return runAppsBuild("")
	case "Run QEMU":
		return runQEMU(false, false)
	case "Run QEMU (Debug Mode)":
		return runQEMU(true, false)
	default:
		return fmt.Errorf("unknown selection: %s", choice)
	}
}
