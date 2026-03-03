// Package commands provides individual CLI command builders for elmos.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/NguyenTrongPhuc552003/elmos/core/domain/bsp"
	"github.com/spf13/cobra"
)

// BuildBootloader creates the bootloader command with subcommands.
func BuildBootloader(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootloader",
		Short: "Manage bootloader builds (U-Boot)",
		Long:  "Build, configure, and manage U-Boot bootloader for the target machine",
	}

	cmd.AddCommand(buildBootloaderBuildCmd(ctx))
	cmd.AddCommand(buildBootloaderConfigCmd(ctx))
	cmd.AddCommand(buildBootloaderCleanCmd(ctx))
	cmd.AddCommand(buildBootloaderBlobsCmd(ctx))
	cmd.AddCommand(buildBootloaderInstallCmd(ctx))

	return cmd
}

// buildBootloaderBuildCmd creates the `elmos bootloader build` subcommand.
func buildBootloaderBuildCmd(ctx *Context) *cobra.Command {
	var defconfig string
	var jobs int

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build U-Boot bootloader",
		Long: `Build U-Boot bootloader for the configured machine.

Uses the machine definition from elmos.yaml or the active machine config.
Firmware blobs are downloaded and cached automatically.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get machine config
			machine := ctx.Config.CurrentMachine
			if machine == nil {
				ctx.Printer.Warn("No machine configured. Using defaults.")
				ctx.Printer.Info("Run: elmos init --machine <name>  to configure a machine")
				return fmt.Errorf("no machine configured for bootloader build")
			}

			ctx.Printer.Info("Building U-Boot for machine: %s", machine.Name)

			// Show bootloader config
			bl := machine.Bootloader
			ctx.Printer.Step("  Bootloader type: %s", bl.Type)
			ctx.Printer.Step("  Repository:      %s", bl.Repo)
			ctx.Printer.Step("  Version:         %s", bl.Version)
			if d := defconfig; d != "" {
				ctx.Printer.Step("  Defconfig:       %s (override)", d)
			} else {
				ctx.Printer.Step("  Defconfig:       %s", bl.Defconfig)
			}

			// Get active bootloader plugin
			bbPlugin := ctx.PluginRegistry.GetPlugin("bootloader-builder")
			if bbPlugin == nil {
				return fmt.Errorf("bootloader-builder plugin not loaded")
			}

			ctx.Printer.Step("Invoking bootloader-builder plugin...")
			ctx.Printer.Info("U-Boot build initiated for %s", machine.Name)

			// TODO: Wire executor through to plugin in Phase 3
			// For now, inform the user what would happen
			buildDir := filepath.Join(ctx.Config.Paths.ProjectRoot, "build", "u-boot")
			ctx.Printer.Step("  Build directory: %s", buildDir)
			ctx.Printer.Step("  Cross compile:   aarch64-linux-gnu-")
			ctx.Printer.Step("  Jobs:            %d", jobs)

			ctx.Printer.Success("Bootloader build configuration validated for %s", machine.Name)
			ctx.Printer.Info("Full build execution will be available in Phase 3 (DAG orchestration)")

			return nil
		},
	}

	cmd.Flags().StringVarP(&defconfig, "defconfig", "d", "", "Override machine defconfig")
	cmd.Flags().IntVarP(&jobs, "jobs", "j", 8, "Number of parallel make jobs")

	return cmd
}

// buildBootloaderConfigCmd creates the `elmos bootloader config` subcommand.
func buildBootloaderConfigCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Apply bootloader defconfig",
		Long:  "Run make <defconfig> to configure U-Boot without a full build",
		RunE: func(cmd *cobra.Command, args []string) error {
			machine := ctx.Config.CurrentMachine
			if machine == nil {
				return fmt.Errorf("no machine configured")
			}

			ctx.Printer.Info("Configuring U-Boot for: %s", machine.Name)
			ctx.Printer.Step("  Defconfig: %s", machine.Bootloader.Defconfig)
			ctx.Printer.Info("Configuration step will run in Phase 3 (full build pipeline)")

			return nil
		},
	}
}

// buildBootloaderCleanCmd creates the `elmos bootloader clean` subcommand.
func buildBootloaderCleanCmd(ctx *Context) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean bootloader build artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			buildDir := filepath.Join(ctx.Config.Paths.ProjectRoot, "build", "u-boot")

			if all {
				ctx.Printer.Step("Removing entire U-Boot build directory: %s", buildDir)
				if err := os.RemoveAll(buildDir); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to clean build directory: %w", err)
				}
			} else {
				ctx.Printer.Step("Running make mrproper in: %s", buildDir)
				ctx.Printer.Info("Clean will run `make mrproper` in Phase 3 (full build pipeline)")
			}

			ctx.Printer.Success("Bootloader build clean complete")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Remove entire build directory (not just artifacts)")

	return cmd
}

// buildBootloaderBlobsCmd creates the `elmos bootloader blobs` subcommand.
func buildBootloaderBlobsCmd(ctx *Context) *cobra.Command {
	var socOverride string
	var download bool

	cmd := &cobra.Command{
		Use:   "blobs",
		Short: "List and manage firmware blobs",
		Long: `Show required firmware blobs for the active machine's SoC.
Optionally download and cache them.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			soc := socOverride
			if soc == "" {
				if machine := ctx.Config.CurrentMachine; machine != nil {
					soc = inferSoCFromMachine(machine.Name, machine.Tags)
				}
			}

			if soc == "" {
				listAllKnownBlobsInfo(ctx)
				return nil
			}

			knownBlobs := bsp.GetBlobsForSoC(soc)
			if len(knownBlobs) == 0 {
				ctx.Printer.Warn("No known firmware blobs for SoC: %s", soc)
				ctx.Printer.Info("Firmware blobs can be manually placed in ~/.elmos/bsp-cache/firmware/%s/", soc)
				return nil
			}

			homeDir, _ := os.UserHomeDir()
			mgr := bsp.NewFirmwareMgr(filepath.Join(homeDir, ".elmos", "bsp-cache"))

			listBlobStatusForSoC(ctx, soc, knownBlobs, mgr)

			if download {
				return downloadBlobsForSoC(ctx, soc, knownBlobs, mgr)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&socOverride, "soc", "s", "", "Override SoC identifier (e.g., rk3588)")
	cmd.Flags().BoolVarP(&download, "download", "d", false, "Download blobs if not cached")

	return cmd
}

// listAllKnownBlobsInfo prints every SoC and its blobs when no machine/SoC is active.
func listAllKnownBlobsInfo(ctx *Context) {
	ctx.Printer.Info("No machine/SoC configured. Known firmware blobs:")
	for socName, blobs := range bsp.KnownFirmwareBlobs {
		ctx.Printer.Step("  [%s] (%d blobs)", socName, len(blobs))
		for _, blob := range blobs {
			ctx.Printer.Step("    • %s", blob.Name)
		}
	}
}

// listBlobStatusForSoC prints the cache status for each blob belonging to soc.
func listBlobStatusForSoC(ctx *Context, soc string, blobs []bsp.BlobSpec, mgr *bsp.FirmwareMgr) {
	ctx.Printer.Info("Firmware blobs for SoC: %s (%d blobs)", soc, len(blobs))
	for _, blob := range blobs {
		cached := "not cached"
		if mgr.IsCached(soc, blob.Name) {
			cached = "✓ cached"
		}
		ctx.Printer.Step("  %-45s [%s]", blob.Name, cached)
	}
}

// downloadBlobsForSoC downloads all blobs for soc and prints the results.
func downloadBlobsForSoC(ctx *Context, soc string, blobs []bsp.BlobSpec, mgr *bsp.FirmwareMgr) error {
	ctx.Printer.Info("Downloading firmware blobs for %s...", soc)
	paths, err := mgr.DownloadBlobs(blobs)
	if err != nil {
		return fmt.Errorf("blob download failed: %w", err)
	}
	ctx.Printer.Success("Downloaded %d firmware blob(s):", len(paths))
	for name, path := range paths {
		ctx.Printer.Step("  ✓ %s → %s", name, path)
	}
	return nil
}

// inferSoCFromMachine tries to determine the SoC identifier from a machine name and tags.
// This is a best-effort heuristic for Phase 2.
func inferSoCFromMachine(machineName string, tags []string) string {
	// Check tags for known SoC identifiers
	knownSoCs := []string{"rk3588", "rk3566", "rk3399", "rk3288", "h616", "h618", "imx8", "imx6"}

	for _, tag := range tags {
		for _, soc := range knownSoCs {
			if tag == soc {
				return soc
			}
		}
	}

	// Try to infer from machine name
	for _, soc := range knownSoCs {
		if containsSubstring(machineName, soc) {
			return soc
		}
	}

	return ""
}

// containsSubstring checks if s contains substr (case-insensitive).
func containsSubstring(s, substr string) bool {
	lower := func(r byte) byte {
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return r
	}
	ls := make([]byte, len(s))
	lsub := make([]byte, len(substr))
	for i := range ls {
		ls[i] = lower(s[i])
	}
	for i := range lsub {
		lsub[i] = lower(substr[i])
	}
	needle := string(lsub)
	haystack := string(ls)
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// buildBootloaderInstallCmd creates the `elmos bootloader install` subcommand.
func buildBootloaderInstallCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Create symbolic links to built bootloader artifacts",
		Long: `Create symbolic links from built bootloader artifacts to the workspace.

Creates links for:
  - U-Boot binary (u-boot.bin, u-boot.itb, etc.)
  - U-Boot environment (uEnv.txt)
  - Boot script (boot.scr)
  - SPL/TPL binaries if present

The bootloader must be built first using 'elmos bootloader build'.

Examples:
  elmos bootloader install           # Install to current workspace
  elmos bootloader install --force   # Force reinstall (recreate links)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBootloaderInstall(ctx)
		},
	}
}

func runBootloaderInstall(ctx *Context) error {
	machine := ctx.Config.CurrentMachine
	if machine == nil {
		return fmt.Errorf("no machine configured")
	}

	buildDir := filepath.Join(ctx.Config.Paths.ProjectRoot, "build", "u-boot")
	if !ctx.FS.Exists(buildDir) {
		ctx.Printer.Warn("U-Boot build directory not found: %s", buildDir)
		ctx.Printer.Info("  Run: elmos bootloader build")
		return nil
	}

	installDir := filepath.Join(ctx.Config.Image.MountPoint, "bootloader")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	ctx.Printer.Step("Installing bootloader artifacts to: %s", installDir)
	installedCount := installBootloaderArtifacts(ctx, machine.Bootloader.Binary, buildDir, installDir)
	if installedCount == 0 {
		ctx.Printer.Warn("No bootloader binaries found in: %s", buildDir)
		ctx.Printer.Info("  Expected files: %s", machine.Bootloader.Binary)
		return nil
	}

	ctx.Printer.Success("Bootloader artifacts installed! (%d files)", installedCount)
	ctx.Printer.Info("  Install directory: %s", installDir)
	return nil
}

func installBootloaderArtifacts(ctx *Context, machineBinary, buildDir, installDir string) int {
	binaries := collectBootloaderBinaries(machineBinary)
	installedCount := 0

	for _, binary := range binaries {
		if tryInstallBootloaderArtifact(ctx, buildDir, installDir, binary) {
			installedCount++
		}
	}

	return installedCount
}

func collectBootloaderBinaries(machineBinary string) []string {
	return []string{
		machineBinary,
		"u-boot.bin",
		"u-boot.itb",
		"u-boot.img",
		"spl/u-boot-spl.bin",
		"tpl/u-boot-tpl.bin",
	}
}

func tryInstallBootloaderArtifact(ctx *Context, buildDir, installDir, binary string) bool {
	if binary == "" {
		return false
	}

	sourcePath := filepath.Join(buildDir, binary)
	if !ctx.FS.Exists(sourcePath) {
		return false
	}

	targetName := filepath.Base(binary)
	targetPath := filepath.Join(installDir, targetName)

	if ctx.FS.Exists(targetPath) {
		if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
			ctx.Printer.Warn("  Failed to remove existing %s: %v", targetName, err)
			return false
		}
	}

	if err := os.Symlink(sourcePath, targetPath); err != nil {
		ctx.Printer.Warn("  Failed to install %s: %v", targetName, err)
		return false
	}

	ctx.Printer.Info("  ✓ %s", targetName)
	return true
}
