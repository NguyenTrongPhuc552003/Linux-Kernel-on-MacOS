package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NguyenTrongPhuc552003/elmos/core/domain/toolchain"
	"github.com/spf13/cobra"
)

// BuildToolchains creates the toolchains command tree for crosstool-ng management.
func BuildToolchains(ctx *Context) *cobra.Command {
	toolchainsCmd := &cobra.Command{
		Use:   "toolchains",
		Short: "Manage cross-compiler toolchains (crosstool-ng)",
		Long: `Manage cross-compiler toolchains using crosstool-ng.

Subcommands allow you to clone crosstool-ng, list available targets,
select a target configuration, build toolchains, and more.

Examples:
  elmos toolchains clone                	  # Clone and install crosstool-ng
  elmos toolchains list                 	  # List target samples for active arch
  elmos toolchains riscv64-unknown-linux-gnu  # Select target
  elmos toolchains build                	  # Build the selected toolchain
  elmos toolchains build -j8            	  # Build with 8 parallel jobs
  elmos toolchains install              	  # Symlink built toolchain to workspace`,
	}

	toolchainsCmd.AddCommand(
		buildToolchainCloneCmd(ctx),
		buildToolchainListCmd(ctx),
		buildToolchainStatusCmd(ctx),
		buildToolchainBuildCmd(ctx),
		buildToolchainInstallCmd(ctx),
		buildToolchainMenuconfigCmd(ctx),
		buildToolchainCleanCmd(ctx),
		buildToolchainEnvCmd(ctx),
	)

	return toolchainsCmd
}

// buildToolchainCloneCmd creates the toolchains clone subcommand.
func buildToolchainCloneCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "clone",
		Short: "Clone and install crosstool-ng from latest git",
		Long:  "Download and install crosstool-ng toolchain builder from upstream git repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ctx.AppContext.EnsureMounted(); err != nil {
				return err
			}
			ctx.Printer.Step("Cloning crosstool-ng...")
			if err := ctx.ToolchainManager.Install(cmd.Context()); err != nil {
				return err
			}
			ctx.Printer.Success("crosstool-ng installed!")
			return nil
		},
	}
}

// buildToolchainInstallCmd creates the toolchains install subcommand for symlinking.
func buildToolchainInstallCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Create symbolic links to built toolchain in workspace",
		Long: `Create symbolic links from the built toolchain to the current workspace.

This makes the cross-compiler accessible for kernel and module builds.
The toolchain must be built first using 'elmos toolchains build'.

Examples:
  elmos toolchains install           # Install to current workspace
  elmos toolchains install --force   # Force reinstall (recreate links)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runToolchainInstall(ctx)
		},
	}
}

func runToolchainInstall(ctx *Context) error {
	if err := ctx.AppContext.EnsureMounted(); err != nil {
		return err
	}

	toolchains, err := ctx.ToolchainManager.GetInstalledToolchains()
	if err != nil {
		return err
	}
	if len(toolchains) == 0 {
		ctx.Printer.Warn("No toolchains built yet")
		ctx.Printer.Info("  Run: elmos toolchains build")
		return nil
	}

	targetToolchain := selectInstalledToolchain(toolchains, ctx.Config.Build.Arch)
	if targetToolchain == nil {
		printAvailableToolchains(ctx, toolchains)
		return nil
	}

	return linkCurrentToolchain(ctx, targetToolchain.Target)
}

func selectInstalledToolchain(toolchains []toolchain.ToolchainInfo, arch string) *toolchain.ToolchainInfo {
	for i := range toolchains {
		tc := toolchains[i]
		if !tc.Installed {
			continue
		}
		if strings.Contains(tc.Target, arch) || (arch == "arm64" && strings.Contains(tc.Target, "aarch64")) {
			return &toolchains[i]
		}
	}
	return nil
}

func printAvailableToolchains(ctx *Context, toolchains []toolchain.ToolchainInfo) {
	arch := ctx.Config.Build.Arch
	ctx.Printer.Warn("No built toolchain found for architecture: %s", arch)
	ctx.Printer.Info("  Available toolchains:")
	for _, tc := range toolchains {
		if tc.Installed {
			ctx.Printer.Info("    • %s", tc.Target)
		}
	}
}

func linkCurrentToolchain(ctx *Context, target string) error {
	workspaceToolchainDir := filepath.Join(ctx.Config.Paths.ToolchainsDir, "current")
	sourceDir := filepath.Join(ctx.ToolchainManager.Paths().XTools, target)

	ctx.Printer.Step("Creating symlink: %s → %s", workspaceToolchainDir, sourceDir)

	if ctx.FS.Exists(workspaceToolchainDir) {
		if err := os.Remove(workspaceToolchainDir); err != nil && !os.IsNotExist(err) {
			ctx.Printer.Warn("Failed to remove existing link: %v", err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(workspaceToolchainDir), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	if err := os.Symlink(sourceDir, workspaceToolchainDir); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	ctx.Printer.Success("Toolchain installed! (%s)", target)
	ctx.Printer.Info("  Toolchain available at: %s", workspaceToolchainDir)
	return nil
}

// buildToolchainListCmd creates the toolchains list subcommand.
func buildToolchainListCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available toolchain samples for active architecture",
		RunE: func(cmd *cobra.Command, args []string) error {
			samples, err := ctx.ToolchainManager.ListSamples(cmd.Context())
			if err != nil {
				return err
			}
			if len(samples) == 0 {
				ctx.Printer.Info("No samples found")
				return nil
			}

			arch := ctx.Config.Build.Arch
			filtered := filterSamplesByArch(samples, arch)
			if len(filtered) == 0 {
				ctx.Printer.Warn("No toolchain samples matched architecture: %s", arch)
				return nil
			}

			ctx.Printer.Info("Toolchain samples for %s:", arch)
			for _, sample := range filtered {
				ctx.Printer.Print("  %s", sample)
			}
			return nil
		},
	}
}

func filterSamplesByArch(samples []string, arch string) []string {
	var filtered []string
	for _, sample := range samples {
		if isSampleRelevantForArch(sample, arch) {
			filtered = append(filtered, sample)
		}
	}
	return filtered
}

func isSampleRelevantForArch(sample, arch string) bool {
	sample = strings.ToLower(sample)
	arch = strings.ToLower(arch)

	switch arch {
	case "arm64":
		return strings.Contains(sample, "aarch64") || strings.Contains(sample, "arm64")
	case "arm":
		return strings.Contains(sample, "arm") && !strings.Contains(sample, "aarch64")
	case "riscv":
		return strings.Contains(sample, "riscv")
	default:
		return strings.Contains(sample, arch)
	}
}

// buildToolchainStatusCmd creates the toolchains status subcommand.
func buildToolchainStatusCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show installed toolchains status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showToolchainStatus(ctx)
		},
	}
}

// showToolchainStatus displays the toolchain installation status.
func showToolchainStatus(ctx *Context) error {
	if !ctx.ToolchainManager.IsInstalled() {
		ctx.Printer.Warn("crosstool-ng not installed")
		ctx.Printer.Print("  Run: elmos toolchains clone")
		return nil
	}
	ctx.Printer.Success("crosstool-ng installed at %s", ctx.ToolchainManager.Paths().CrosstoolNG)

	toolchains, err := ctx.ToolchainManager.GetInstalledToolchains()
	if err != nil {
		return err
	}
	if len(toolchains) == 0 {
		ctx.Printer.Info("No toolchains built yet")
		return nil
	}
	ctx.Printer.Print("")
	ctx.Printer.Print("Installed toolchains:")
	for _, tc := range toolchains {
		status := "✓"
		if !tc.Installed {
			status = "○"
		}
		ctx.Printer.Print("  %s %s", status, tc.Target)
	}
	return nil
}

// buildToolchainBuildCmd creates the toolchains build subcommand.
func buildToolchainBuildCmd(ctx *Context) *cobra.Command {
	var buildJobs int
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the currently configured toolchain",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ctx.AppContext.EnsureMounted(); err != nil {
				return err
			}
			ctx.Printer.Step("Building toolchain...")
			if err := ctx.ToolchainManager.Build(cmd.Context(), buildJobs); err != nil {
				return err
			}
			ctx.Printer.Success("Toolchain built!")
			return nil
		},
	}
	cmd.Flags().IntVarP(&buildJobs, "jobs", "j", 0, "Number of parallel jobs (default: CPU count)")
	return cmd
}

// buildToolchainMenuconfigCmd creates the toolchains menuconfig subcommand.
func buildToolchainMenuconfigCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "menuconfig",
		Short: "Open interactive configuration menu",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ctx.ToolchainManager.Menuconfig(cmd.Context())
		},
	}
}

// buildToolchainCleanCmd creates the toolchains clean subcommand.
func buildToolchainCleanCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Clean build artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx.Printer.Step("Cleaning build artifacts...")
			if err := ctx.ToolchainManager.Clean(cmd.Context()); err != nil {
				return err
			}
			ctx.Printer.Success("Cleaned!")
			return nil
		},
	}
}

// buildToolchainEnvCmd creates the toolchains env subcommand.
func buildToolchainEnvCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "env",
		Short: "Print environment variables for shell integration",
		RunE: func(cmd *cobra.Command, args []string) error {
			paths := ctx.ToolchainManager.Paths()
			fmt.Printf("export PATH=\"%s/bin:$PATH\"\n", paths.CrosstoolNG)

			toolchains, _ := ctx.ToolchainManager.GetInstalledToolchains()
			for _, tc := range toolchains {
				if tc.Installed {
					fmt.Printf("export PATH=\"%s/bin:$PATH\"\n", tc.Path)
				}
			}
			return nil
		},
	}
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
