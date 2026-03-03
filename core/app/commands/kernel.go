package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/NguyenTrongPhuc552003/elmos/core/domain/builder"
)

// BuildKernel creates the kernel command tree for kernel management.
func BuildKernel(ctx *Context) *cobra.Command {
	kernelCmd := &cobra.Command{
		Use:   "kernel",
		Short: "Kernel configuration commands",
	}

	kernelCmd.AddCommand(
		buildKernelConfigCmd(ctx),
		buildKernelCleanCmd(ctx),
		buildKernelCloneCmd(ctx),
		buildKernelStatusCmd(ctx),
		buildKernelResetCmd(ctx),
		buildKernelSwitchCmd(ctx),
		buildKernelPullCmd(ctx),
		buildKernelBuildCmd(ctx),
		buildKernelInstallCmd(ctx),
	)

	return kernelCmd
}

// buildKernelConfigCmd creates the kernel config subcommand.
func buildKernelConfigCmd(ctx *Context) *cobra.Command {
	var enableOpts []string
	cmd := &cobra.Command{
		Use:   "config [type]",
		Short: "Configure the kernel",
		Long: `Configure the kernel with a config target or enable specific options.

Examples:
  elmos kernel config              # Run defconfig
  elmos kernel config menuconfig   # Interactive config
  elmos kernel config -E NETFILTER # Enable CONFIG_NETFILTER`,
		RunE: RunEWithContext(ctx, func(cmd *cobra.Command, args []string) error {
			// Handle --enable options
			if len(enableOpts) > 0 {
				return enableKernelConfigs(ctx, cmd, enableOpts)
			}

			configType := "defconfig"
			if len(args) > 0 {
				configType = args[0]
			}
			ctx.Printer.Step("Running kernel %s...", configType)
			if err := ctx.KernelBuilder.Configure(cmd.Context(), configType); err != nil {
				return err
			}
			ctx.Printer.Success("Kernel configured!")
			return nil
		}),
	}
	cmd.Flags().StringArrayVarP(&enableOpts, "enable", "E", nil, "Enable kernel config option (e.g., NETFILTER)")
	return cmd
}

// enableKernelConfigs enables specific kernel config options and runs oldconfig.
func enableKernelConfigs(ctx *Context, cmd *cobra.Command, opts []string) error {
	configPath := ctx.Config.Paths.KernelDir + "/.config"
	scriptsConfig := ctx.Config.Paths.KernelDir + "/scripts/config"

	// Enable each option
	for _, opt := range opts {
		// Add CONFIG_ prefix if not present
		configName := opt
		if !strings.HasPrefix(opt, "CONFIG_") {
			configName = "CONFIG_" + opt
		}
		ctx.Printer.Step("Enabling %s...", configName)
		if err := ctx.Exec.Run(cmd.Context(), scriptsConfig, "--file", configPath, "--enable", configName); err != nil {
			return fmt.Errorf("failed to enable %s: %w", configName, err)
		}
	}

	// Run oldconfig to resolve dependencies
	ctx.Printer.Step("Updating config...")
	if err := ctx.KernelBuilder.Configure(cmd.Context(), "olddefconfig"); err != nil {
		return err
	}

	ctx.Printer.Success("Config updated! Run 'elmos kernel build' to apply changes.")
	return nil
}

// buildKernelCleanCmd creates the kernel clean subcommand.
func buildKernelCleanCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Clean kernel build artifacts",
		RunE: RunEWithContext(ctx, func(cmd *cobra.Command, args []string) error {
			ctx.Printer.Step("Cleaning...")
			if err := ctx.KernelBuilder.Clean(cmd.Context()); err != nil {
				return err
			}
			ctx.Printer.Success("Cleaned!")
			return nil
		}),
	}
}

// buildKernelCloneCmd creates the kernel clone subcommand.
func buildKernelCloneCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "clone [git-url]",
		Short: "Clone the Linux kernel source",
		RunE: RunEWithContext(ctx, func(cmd *cobra.Command, args []string) error {
			if ctx.AppContext.KernelExists() {
				ctx.Printer.Info("Kernel source already exists at %s", ctx.Config.Paths.KernelDir)
				return nil
			}
			url := "https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git"
			if len(args) > 0 {
				url = args[0]
			}
			ctx.Printer.Step("Cloning kernel from %s...", url)
			if err := ctx.Exec.Run(cmd.Context(), "git", "clone", url, ctx.Config.Paths.KernelDir); err != nil {
				return fmt.Errorf("failed to clone: %w", err)
			}
			ctx.Printer.Success("Kernel cloned to %s", ctx.Config.Paths.KernelDir)
			return nil
		}),
	}
}

// buildKernelStatusCmd creates the kernel status subcommand.
func buildKernelStatusCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show kernel source status",
		RunE: RunEWithContext(ctx, func(cmd *cobra.Command, args []string) error {
			if !ctx.AppContext.KernelExists() {
				ctx.Printer.Info("Kernel source not found at %s", ctx.Config.Paths.KernelDir)
				ctx.Printer.Print("  Run 'elmos kernel clone' to download kernel source")
				return nil
			}
			ctx.Printer.Success("Kernel source found at %s", ctx.Config.Paths.KernelDir)
			ctx.Printer.Print("")
			printKernelGitInfo(ctx, cmd)
			printKernelBuildStatus(ctx)
			return nil
		}),
	}
}

// buildKernelResetCmd creates the kernel reset subcommand.
func buildKernelResetCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset kernel source (reclone completely)",
		RunE: RunEWithContext(ctx, func(cmd *cobra.Command, args []string) error {
			if ctx.AppContext.KernelExists() {
				ctx.Printer.Step("Removing existing kernel source...")
				if err := os.RemoveAll(ctx.Config.Paths.KernelDir); err != nil {
					return fmt.Errorf("failed to remove kernel: %w", err)
				}
			}
			url := "https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git"
			ctx.Printer.Step("Cloning kernel from %s...", url)
			if err := ctx.Exec.Run(cmd.Context(), "git", "clone", url, ctx.Config.Paths.KernelDir); err != nil {
				return fmt.Errorf("failed to clone: %w", err)
			}
			ctx.Printer.Success("Kernel reset complete!")
			return nil
		}),
	}
}

// buildKernelSwitchCmd creates the kernel switch subcommand.
func buildKernelSwitchCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "switch [ref]",
		Short: "List or switch branch/tag (auto-detects)",
		Long: `List all branches and tags, or switch to a specific ref.
Automatically detects whether the ref is a branch or tag.

Examples:
  elmos kernel switch           # List all refs
  elmos kernel switch master    # Switch to branch
  elmos kernel switch v6.7      # Switch to tag`,
		RunE: RunEWithContext(ctx, func(cmd *cobra.Command, args []string) error {
			if !ctx.AppContext.KernelExists() {
				ctx.Printer.Info("Kernel source not found. Run 'elmos kernel clone' first.")
				return nil
			}
			if len(args) == 0 {
				return listKernelRefs(ctx, cmd)
			}
			return switchKernelRef(ctx, cmd, args[0])
		}),
	}
}

// buildKernelPullCmd creates the kernel pull subcommand.
func buildKernelPullCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Update kernel source",
		RunE: RunEWithContext(ctx, func(cmd *cobra.Command, args []string) error {
			if !ctx.AppContext.KernelExists() {
				ctx.Printer.Info("Kernel source not found. Run 'elmos kernel clone' first.")
				return nil
			}
			ctx.Printer.Step("Updating kernel source...")
			if err := ctx.Exec.Run(cmd.Context(), "git", "-C", ctx.Config.Paths.KernelDir, "pull"); err != nil {
				return fmt.Errorf("failed to update: %w", err)
			}
			ctx.Printer.Success("Kernel updated!")
			return nil
		}),
	}
}

// buildKernelBuildCmd creates the kernel build subcommand.
func buildKernelBuildCmd(ctx *Context) *cobra.Command {
	var jobs int
	cmd := &cobra.Command{
		Use:   "build [targets...]",
		Short: "Build the Linux kernel",
		RunE: RunEWithContext(ctx, func(cmd *cobra.Command, args []string) error {
			targets := args
			if len(targets) == 0 {
				targets = ctx.KernelBuilder.GetDefaultTargets()
			}
			ctx.Printer.Step("Building kernel for %s...", ctx.Config.Build.Arch)
			if err := ctx.KernelBuilder.Build(cmd.Context(), builder.BuildOptions{Jobs: jobs, Targets: targets}); err != nil {
				return err
			}
			ctx.Printer.Success("Build complete!")
			return nil
		}),
	}
	cmd.Flags().IntVarP(&jobs, "jobs", "j", 0, "Number of parallel build jobs")
	return cmd
}

// buildKernelInstallCmd creates the kernel install subcommand.
func buildKernelInstallCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Create symbolic links to built kernel artifacts",
		Long: `Create symbolic links from built kernel artifacts to the workspace.

Creates links for:
  - Kernel image (Image, zImage, bzImage, vmlinuz)
  - Device tree blobs (.dtb files)
  - Kernel config (.config)
  - System.map
  - vmlinux (for debugging)

The kernel must be built first using 'elmos kernel build'.

Examples:
  elmos kernel install           # Install to current workspace
  elmos kernel install --force   # Force reinstall (recreate links)`,
		RunE: RunEWithContext(ctx, func(cmd *cobra.Command, args []string) error {
			return runKernelInstall(ctx)
		}),
	}
}

func runKernelInstall(ctx *Context) error {
	kernelDir := ctx.Config.Paths.KernelDir
	if !ctx.AppContext.KernelExists() {
		ctx.Printer.Warn("Kernel source not found at %s", kernelDir)
		ctx.Printer.Info("  Run: elmos kernel clone")
		return nil
	}

	kernelImage := ctx.AppContext.GetKernelImage()
	if !ctx.FS.Exists(kernelImage) {
		ctx.Printer.Warn("Kernel not built yet")
		ctx.Printer.Info("  Run: elmos kernel build")
		return nil
	}

	installDir := filepath.Join(ctx.Config.Image.MountPoint, "kernel")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	ctx.Printer.Step("Installing kernel artifacts to: %s", installDir)
	if err := installKernelCoreArtifacts(ctx, kernelDir, kernelImage, installDir); err != nil {
		return err
	}
	if err := installKernelDTBs(ctx, kernelDir, installDir); err != nil {
		return err
	}

	ctx.Printer.Success("Kernel artifacts installed!")
	ctx.Printer.Info("  Install directory: %s", installDir)
	return nil
}

func installKernelCoreArtifacts(ctx *Context, kernelDir, kernelImage, installDir string) error {
	if err := createSymlink(ctx, kernelImage, filepath.Join(installDir, filepath.Base(kernelImage))); err != nil {
		return err
	}
	ctx.Printer.Info("  ✓ %s", filepath.Base(kernelImage))

	optional := map[string]string{
		"vmlinux":    "vmlinux",
		"System.map": "System.map",
		".config":    "config",
	}
	for sourceName, targetName := range optional {
		sourcePath := filepath.Join(kernelDir, sourceName)
		if !ctx.FS.Exists(sourcePath) {
			continue
		}
		if err := createSymlink(ctx, sourcePath, filepath.Join(installDir, targetName)); err != nil {
			return err
		}
		ctx.Printer.Info("  ✓ %s", targetName)
	}
	return nil
}

func installKernelDTBs(ctx *Context, kernelDir, installDir string) error {
	archConfig := ctx.Config.GetArchConfig()
	if archConfig == nil {
		return nil
	}

	dtbDir := filepath.Join(kernelDir, "arch", archConfig.KernelArch, "boot", "dts")
	if !ctx.FS.Exists(dtbDir) {
		return nil
	}

	dtbInstallDir := filepath.Join(installDir, "dtbs")
	if err := os.MkdirAll(dtbInstallDir, 0755); err != nil {
		return fmt.Errorf("failed to create DTB directory: %w", err)
	}

	dtbFiles, err := filepath.Glob(filepath.Join(dtbDir, "*.dtb"))
	if err != nil {
		return err
	}
	for _, dtbFile := range dtbFiles {
		dtbName := filepath.Base(dtbFile)
		if err := createSymlink(ctx, dtbFile, filepath.Join(dtbInstallDir, dtbName)); err != nil {
			ctx.Printer.Warn("  Failed to install %s: %v", dtbName, err)
			continue
		}
		ctx.Printer.Info("  ✓ dtbs/%s", dtbName)
	}
	return nil
}

// createSymlink removes existing file/link and creates a new symlink.
func createSymlink(ctx *Context, source, target string) error {
	// Remove existing file/link if it exists
	if ctx.FS.Exists(target) {
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}

	// Create symlink
	if err := os.Symlink(source, target); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// --- Helper functions to reduce RunE complexity ---

// printKernelGitInfo prints git branch/tag and commit info for status command.
func printKernelGitInfo(ctx *Context, cmd *cobra.Command) {
	ctx.Printer.Step("Git info:")
	branch, err := ctx.Exec.Output(cmd.Context(), "git", "-C", ctx.Config.Paths.KernelDir, "symbolic-ref", "-q", "--short", "HEAD")
	if err == nil {
		ctx.Printer.Print("  Branch: %s", strings.TrimSpace(string(branch)))
	} else {
		tag, err := ctx.Exec.Output(cmd.Context(), "git", "-C", ctx.Config.Paths.KernelDir, "describe", "--tags", "--exact-match")
		if err == nil {
			ctx.Printer.Print("  Tag: %s", strings.TrimSpace(string(tag)))
		} else {
			ctx.Printer.Print("  Branch: <detached>")
		}
	}
	commit, err := ctx.Exec.Output(cmd.Context(), "git", "-C", ctx.Config.Paths.KernelDir, "log", "-1", "--format=%h %s")
	if err == nil {
		ctx.Printer.Print("  Commit: %s", strings.TrimSpace(string(commit)))
	}
}

// printKernelBuildStatus prints kernel config and image status.
func printKernelBuildStatus(ctx *Context) {
	ctx.Printer.Print("")
	ctx.Printer.Step("Build status:")
	if ctx.AppContext.HasConfig() {
		ctx.Printer.Print("  ✓ Kernel configured (.config exists)")
	} else {
		ctx.Printer.Print("  ○ Not configured (run 'elmos kernel config')")
	}
	if ctx.AppContext.HasKernelImage() {
		ctx.Printer.Print("  ✓ Kernel image built")
	} else {
		ctx.Printer.Print("  ○ Kernel not built (run 'elmos build')")
	}
}

// listKernelRefs lists available branches and tags for switch command.
func listKernelRefs(ctx *Context, cmd *cobra.Command) error {
	ctx.Printer.Step("Branches:")
	// Use for-each-ref for structured output
	branches, _ := ctx.Exec.Output(cmd.Context(), "git", "-C", ctx.Config.Paths.KernelDir, "for-each-ref", "--format=%(refname:short)", "refs/heads/")
	for _, b := range strings.Split(string(branches), "\n") {
		if b != "" {
			ctx.Printer.Print("  %s", b)
		}
	}
	ctx.Printer.Print("")
	ctx.Printer.Step("Tags (latest 10):")
	// Optimized tag listing
	tags, _ := ctx.Exec.Output(cmd.Context(), "git", "-C", ctx.Config.Paths.KernelDir, "describe", "--tags", "--abbrev=0")
	// Fallback to list if describe only shows one
	if tags == nil || len(string(tags)) < 2 {
		tagsRaw, _ := ctx.Exec.Output(cmd.Context(), "git", "-C", ctx.Config.Paths.KernelDir, "tag", "-l", "--sort=-v:refname", "v*")
		lines := strings.Split(string(tagsRaw), "\n")
		for i, t := range lines {
			if i >= 10 || t == "" {
				break
			}
			ctx.Printer.Print("  %s", t)
		}
	} else {
		ctx.Printer.Print("  Latest: %s", strings.TrimSpace(string(tags)))
	}
	return nil
}

// switchKernelRef switches to a specific branch or tag.
func switchKernelRef(ctx *Context, cmd *cobra.Command, ref string) error {
	ctx.Printer.Step("Switching to: %s", ref)
	if err := ctx.Exec.Run(cmd.Context(), "git", "-C", ctx.Config.Paths.KernelDir, "checkout", ref); err != nil {
		ctx.Printer.Info("Not found locally, fetching...")
		_ = ctx.Exec.Run(cmd.Context(), "git", "-C", ctx.Config.Paths.KernelDir, "fetch", "--all", "--tags")
		if err := ctx.Exec.Run(cmd.Context(), "git", "-C", ctx.Config.Paths.KernelDir, "checkout", ref); err != nil {
			return fmt.Errorf("failed to switch: %w", err)
		}
	}
	ctx.Printer.Success("Now on: %s", ref)
	return nil
}

// RunEWithContext is a helper to run commands that require the AppContext to be mounted.
func RunEWithContext(ctx *Context, run func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := ctx.AppContext.EnsureMounted(); err != nil {
			return err
		}
		return run(cmd, args)
	}
}
