package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/NguyenTrongPhuc552003/elmos/core/config"
	"github.com/NguyenTrongPhuc552003/elmos/core/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// BuildInit creates the init command for workspace initialization.
func BuildInit(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "init [workspace_name] [size]",
		Short: "Initialize workspace (mount volume)",
		Long: `Initialize workspace and mount volume.

Arguments:
  workspace_name  Optional name for the workspace volume (default: "elmos")
  size           Optional volume size (default: "40G", minimum: 40G)

Examples:
  elmos init                    # Create workspace with 40GB
  elmos init my_workspace       # Create named workspace with 40GB
  elmos init my_workspace 50G   # Create named workspace with 50GB`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(ctx, cmd, args)
		},
	}
}

// runInit executes the init command logic.
// CC is kept ≤ 5 by delegating argument parsing and config resolution to helpers.
func runInit(ctx *Context, cmd *cobra.Command, args []string) error {
	workspaceName, volumeSize, err := resolveInitArgs(ctx, args)
	if err != nil {
		return err
	}

	if err := loadOrCreateWorkspaceConfig(ctx, workspaceName, volumeSize); err != nil {
		return err
	}

	if err := ensureWorkspaceVolume(ctx, cmd); err != nil {
		return err
	}

	if err := initializeWorkspace(ctx, ctx.Config.Image.MountPoint); err != nil {
		ctx.Printer.Warn("Workspace structure initialization failed: %v", err)
		// Don't fail — volume is mounted; structure setup is convenience-only.
	}

	ctx.Printer.Success("Workspace initialized! Volume mounted at %s", ctx.Config.Image.MountPoint)
	ctx.Printer.Info("Next step: cd %s", workspaceName)
	return nil
}

// resolveInitArgs parses CLI args into workspace name and volume size, applying
// config defaults and validating the size when explicitly provided.
func resolveInitArgs(ctx *Context, args []string) (name, size string, err error) {
	name = ctx.Config.Image.VolumeName
	if name == "" {
		name = config.DefaultVolumeName
	}
	size = ctx.Config.Image.Size
	if size == "" {
		size = config.DefaultImageSize
	}
	if len(args) > 0 {
		name = args[0]
	}
	if len(args) > 1 {
		size = args[1]
		if err = validateVolumeSize(size, ctx.Printer); err != nil {
			return "", "", err
		}
	}
	return name, size, nil
}

// loadOrCreateWorkspaceConfig auto-discovers an existing workspace config at
// $PWD/<name>/<name>.yaml and loads it, or creates a fresh config when none exists.
func loadOrCreateWorkspaceConfig(ctx *Context, workspaceName, volumeSize string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	existingConfig := filepath.Join(cwd, workspaceName, workspaceName+".yaml")
	if _, statErr := os.Stat(existingConfig); statErr == nil {
		// Existing workspace found — reload config so we remount instead of overwriting.
		ctx.Printer.Step("Found existing workspace config: %s", existingConfig)
		if loaded, loadErr := config.Load(existingConfig); loadErr == nil {
			*ctx.Config = *loaded
		}
		return nil
	}
	// No existing workspace — create fresh config.
	return updateInitConfig(ctx, workspaceName, volumeSize)
}

// ensureWorkspaceVolume creates and mounts the disk image if needed.
func ensureWorkspaceVolume(ctx *Context, cmd *cobra.Command) error {
	plat := ctx.AppContext.Platform

	// Create disk image if it doesn't exist
	if !ctx.FS.Exists(ctx.Config.Image.Path) {
		ctx.Printer.Step("Creating sparse disk image...")
		sizeGB := parseSizeToGB(ctx.Config.Image.Size)
		if err := plat.DiskImage().Create(cmd.Context(), ctx.Config.Image.Path, sizeGB); err != nil {
			return fmt.Errorf("failed to create disk image: %w", err)
		}
		ctx.Printer.Success("Disk image created!")
	}

	// Mount volume if not already mounted
	if !ctx.AppContext.IsMounted() {
		ctx.Printer.Step("Mounting volume...")
		mp, err := plat.DiskImage().Mount(cmd.Context(), ctx.Config.Image.Path)
		if err != nil {
			return fmt.Errorf("failed to mount: %w", err)
		}
		// Update mount point in config in case it differs (e.g. suffix on macOS)
		if mp != "" {
			ctx.Config.Image.MountPoint = mp
		}
	}
	return nil
}

// parseSizeToGB parses a size string like "40G" or "2T" into gigabytes.
// Returns config.MinimumImageSize on parse failure.
func parseSizeToGB(size string) int {
	var n int
	var unit string
	if _, err := fmt.Sscanf(size, "%d%s", &n, &unit); err != nil {
		return config.MinimumImageSize
	}
	switch unit {
	case "T", "t":
		return n * 1024
	case "M", "m":
		return n / 1024
	default: // G, g
		return n
	}
}

// updateInitConfig updates the configuration and saves it to $PWD/<name>/<name>.yaml.
// The local workspace directory ($PWD/<name>/) holds the disk image, config, and
// workspace-local copies of examples, assets, and patches so every path resolves
// inside the workspace rather than pointing at the source repository.
func updateInitConfig(ctx *Context, workspaceName, volumeSize string) error {
	ctx.Config.Image.VolumeName = workspaceName
	ctx.Config.Image.Size = volumeSize

	// Determine the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Local workspace directory: $PWD/<name>/
	localDir := filepath.Join(cwd, workspaceName)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory %s: %w", localDir, err)
	}

	// Disk image lives inside the local workspace directory
	ext := ".sparseimage"
	if runtime.GOOS != "darwin" {
		ext = ".img"
	}
	ctx.Config.Image.Path = filepath.Join(localDir, workspaceName+ext)

	// Mount point is determined by the platform layer
	mountRoot := ctx.AppContext.Platform.Paths().WorkspaceRoot(workspaceName)
	ctx.Config.Image.MountPoint = mountRoot

	// ProjectRoot for subsequent commands (after user cds into the workspace)
	ctx.Config.Paths.ProjectRoot = localDir
	ctx.Config.Paths.ToolchainsDir = filepath.Join(mountRoot, "toolchains")

	// All resource paths must resolve inside the workspace directory.
	// Create sub-directories so the user can add custom modules/apps/patches
	// without modifying the source repository.
	ctx.Config.Paths.ModulesDir = filepath.Join(localDir, "examples", "modules")
	ctx.Config.Paths.AppsDir = filepath.Join(localDir, "examples", "apps")
	ctx.Config.Paths.LibrariesDir = filepath.Join(localDir, "assets", "libraries")
	ctx.Config.Paths.PatchesDir = filepath.Join(localDir, "patches")

	// Create the workspace-local resource directories so they exist on disk.
	for _, sub := range []string{
		filepath.Join("examples", "modules"),
		filepath.Join("examples", "apps"),
		filepath.Join("assets", "libraries"),
		"patches",
	} {
		if mkErr := os.MkdirAll(filepath.Join(localDir, sub), 0755); mkErr != nil {
			return fmt.Errorf("failed to create %s: %w", sub, mkErr)
		}
	}

	// Mount-relative paths are computed when config is next loaded.
	ctx.Config.Paths.KernelDir = ""
	ctx.Config.Paths.RootfsDir = ""
	ctx.Config.Paths.DiskImage = ""

	// Save config inside the local workspace directory, named after the workspace.
	// e.g. hello/hello.yaml  (not the generic elmos.yaml)
	configPath := filepath.Join(localDir, workspaceName+".yaml")
	if err := ctx.Config.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// validateVolumeSize checks if the provided size meets minimum requirements.
func validateVolumeSize(size string, printer *ui.Printer) error {
	// Parse size string (e.g., "40G", "50G", "1T")
	var numericValue int
	var unit string

	if _, err := fmt.Sscanf(size, "%d%s", &numericValue, &unit); err != nil {
		return fmt.Errorf("invalid size format: %s (expected format: 40G, 50G, etc.)", size)
	}

	// Convert to GB for comparison
	sizeInGB := numericValue
	switch unit {
	case "G", "g":
		// Already in GB
	case "T", "t":
		sizeInGB = numericValue * 1024
	case "M", "m":
		sizeInGB = numericValue / 1024
	default:
		return fmt.Errorf("invalid size unit: %s (use G for gigabytes or T for terabytes)", unit)
	}

	// Validate minimum size
	if sizeInGB < config.MinimumImageSize {
		printer.Warn("⚠️  Volume size %s is less than the recommended minimum of %dG", size, config.MinimumImageSize)
		printer.Warn("   This may cause issues with toolchain builds and kernel compilation")
		printer.Warn("   Consider using at least %dG for optimal performance", config.MinimumImageSize)
	}

	return nil
}

// initializeWorkspace creates the v2.0 workspace directory structure and plugins.yml.
// It is idempotent: if plugins.yml already exists the workspace is considered
// fully initialized and the function returns immediately without overwriting
// any user-modified files.
func initializeWorkspace(ctx *Context, rootPath string) error {
	// Create workspace manager first so we can query paths.
	wsManager := config.NewWorkspaceManager(rootPath)

	// Skip if workspace was already initialized (plugins.yml present).
	// This prevents overwriting user-customised plugin configurations on
	// every subsequent `elmos init` (re-mount) invocation.
	if ctx.FS.Exists(wsManager.GetPluginConfigPath()) {
		ctx.Printer.Step("  ✓ Workspace already initialized, skipping")
		return nil
	}

	ctx.Printer.Step("Initializing v2.0 workspace structure...")

	// Initialize directory structure
	if err := wsManager.Initialize(); err != nil {
		return fmt.Errorf("failed to create workspace directories: %w", err)
	}
	ctx.Printer.Step("  ✓ Created directory structure")

	// Generate default plugins.yml
	defaultPluginsConfig := map[string]interface{}{
		"plugins": map[string]interface{}{
			"kernel-builder": map[string]interface{}{
				"type":    "builtin",
				"enabled": true,
			},
			"bsp-manager": map[string]interface{}{
				"type":    "builtin",
				"enabled": true,
				"config": map[string]interface{}{
					"registries": []map[string]string{
						{
							"name": "official",
							"url":  "https://github.com/elmos-sdk/bsp-registry",
						},
					},
				},
			},
		},
	}

	// Marshal plugins config to YAML
	pluginsYAML, err := yaml.Marshal(defaultPluginsConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal plugins config: %w", err)
	}

	// Write plugins.yml
	pluginsConfigPath := wsManager.GetPluginConfigPath()
	if err := ctx.FS.WriteFile(pluginsConfigPath, pluginsYAML, 0644); err != nil {
		return fmt.Errorf("failed to write plugins.yml: %w", err)
	}
	ctx.Printer.Step("  ✓ Generated plugins.yml")

	ctx.Printer.Success("Workspace initialized successfully!")
	return nil
}

// BuildExit creates the exit command for unmounting the workspace.
func BuildExit(ctx *Context) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "exit",
		Short: "Exit workspace (unmount volume)",
		RunE: func(cmd *cobra.Command, args []string) error {
			plat := ctx.AppContext.Platform
			name := ctx.Config.Image.VolumeName

			// 1. Ask the platform layer for the actual mount status.
			//    This handles macOS (hdiutil info) and Linux (/proc/mounts)
			//    authoritatively rather than relying solely on config state.
			mounted, actualMP, err := plat.DiskImage().IsMounted(cmd.Context(), name)
			if err != nil {
				// If the check itself fails, fall back to config mount point.
				ctx.Printer.Warn("Could not query mount status: %v", err)
				mounted = false
			}

			if !mounted {
				ctx.Printer.Info("Volume %q is not mounted", name)
				return nil
			}

			// Use the platform-reported mount point (more reliable than config).
			mountPoint := actualMP
			if mountPoint == "" {
				mountPoint = ctx.Config.Image.MountPoint
			}

			ctx.Printer.Step("Unmounting volume at %s...", mountPoint)
			_ = force // TODO(platform): wire force-unmount
			if err := plat.DiskImage().Unmount(cmd.Context(), mountPoint); err != nil {
				return fmt.Errorf("failed to unmount: %w", err)
			}
			ctx.Printer.Success("Volume unmounted")
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force unmount (needed if resource is busy)")
	return cmd
}
