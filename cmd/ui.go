// Package cmd implements the Cobra CLI commands for elmos.
package cmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/NguyenTrongPhuc552003/elmos/internal/tui"
)

// uiCmd - Interactive TUI
var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch interactive menu (TUI)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}

func runTUI() error {
	m := tui.NewMenuModel()
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	menu, ok := finalModel.(tui.MenuModel)
	if !ok || menu.Choice() == "" {
		return nil
	}

	// Dispatch based on selection
	choice := menu.Choice()
	// Strip "1. " prefix
	if idx := strings.Index(choice, ". "); idx != -1 {
		choice = choice[idx+2:]
	}

	printStep("Executing: %s", choice)

	switch choice {
	case "Doctor (Check Environment)":
		return runDoctor()
	case "Init Workspace":
		// Init command logic
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


