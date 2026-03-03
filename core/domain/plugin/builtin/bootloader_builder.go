// Package builtin provides built-in plugins for the ELMOS system.
package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	elcontext "github.com/NguyenTrongPhuc552003/elmos/core/context"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/bsp"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/core/plugin"
)

// BootloaderBuilderPlugin orchestrates U-Boot builds.
// It handles source fetching, patching, firmware blob staging, and assembly.
type BootloaderBuilderPlugin struct {
	exec        executor.Executor
	firmwareMgr *bsp.FirmwareMgr
	config      *BootloaderBuilderConfig
	ctx         *elcontext.Context
}

// BootloaderBuilderConfig holds configuration for the bootloader builder.
type BootloaderBuilderConfig struct {
	// UBootRepo is the U-Boot git repository URL.
	UBootRepo string `mapstructure:"uboot_repo"`

	// UBootVersion is the git tag or branch to use.
	UBootVersion string `mapstructure:"uboot_version"`

	// SoCIdentifier is the SoC name for firmware blob lookup (e.g. "rk3588").
	SoCIdentifier string `mapstructure:"soc"`

	// Defconfig is the board defconfig name.
	Defconfig string `mapstructure:"defconfig"`

	// OutputBinary is the expected output binary name.
	OutputBinary string `mapstructure:"output_binary"`

	// FirmwareCacheDir is the directory where firmware blobs are cached.
	FirmwareCacheDir string `mapstructure:"firmware_cache_dir"`

	// CrossCompile is the cross-compilation prefix (e.g. "aarch64-linux-gnu-").
	CrossCompile string `mapstructure:"cross_compile"`

	// Jobs is the number of parallel make jobs.
	Jobs int `mapstructure:"jobs"`

	// PatchDir is the directory containing U-Boot patches to apply.
	PatchDir string `mapstructure:"patch_dir"`

	// BuildDir is where U-Boot source is checked out and built.
	BuildDir string `mapstructure:"build_dir"`
}

// NewBootloaderBuilderPlugin creates a new BootloaderBuilder plugin instance.
func NewBootloaderBuilderPlugin() (*BootloaderBuilderPlugin, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home dir: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".elmos", "bsp-cache")
	firmwareMgr := bsp.NewFirmwareMgr(cacheDir)

	return &BootloaderBuilderPlugin{
		firmwareMgr: firmwareMgr,
		config: &BootloaderBuilderConfig{
			UBootRepo:    "https://github.com/u-boot/u-boot.git",
			UBootVersion: "v2024.01",
			Jobs:         8,
			OutputBinary: "u-boot.itb",
			CrossCompile: "aarch64-linux-gnu-",
		},
	}, nil
}

// Name returns the plugin name.
func (bb *BootloaderBuilderPlugin) Name() string { return "bootloader-builder" }

// Version returns the plugin version.
func (bb *BootloaderBuilderPlugin) Version() string { return "2.0.0-alpha" }

// Description returns the plugin description.
func (bb *BootloaderBuilderPlugin) Description() string {
	return "U-Boot bootloader build orchestration with firmware blob staging"
}

// Init initializes the plugin with application context and config.
func (bb *BootloaderBuilderPlugin) Init(ctx *elcontext.Context, cfg map[string]interface{}) error {
	bb.ctx = ctx

	// Extract executor from context if available
	// (will be set during app initialization in later phases)

	// Apply configuration overrides
	if cfg != nil {
		if repo, ok := cfg["uboot_repo"].(string); ok {
			bb.config.UBootRepo = repo
		}
		if version, ok := cfg["uboot_version"].(string); ok {
			bb.config.UBootVersion = version
		}
		if soc, ok := cfg["soc"].(string); ok {
			bb.config.SoCIdentifier = soc
		}
		if defconfig, ok := cfg["defconfig"].(string); ok {
			bb.config.Defconfig = defconfig
		}
		if jobs, ok := cfg["jobs"].(float64); ok && jobs > 0 {
			bb.config.Jobs = int(jobs)
		}
		if crossCompile, ok := cfg["cross_compile"].(string); ok {
			bb.config.CrossCompile = crossCompile
		}
		if buildDir, ok := cfg["build_dir"].(string); ok {
			bb.config.BuildDir = buildDir
		}
	}

	return nil
}

// Validate checks that required configuration is present.
func (bb *BootloaderBuilderPlugin) Validate() error {
	if bb.config.UBootRepo == "" {
		return fmt.Errorf("bootloader-builder: uboot_repo is required")
	}
	if bb.config.UBootVersion == "" {
		return fmt.Errorf("bootloader-builder: uboot_version is required")
	}
	if bb.config.Jobs < 1 || bb.config.Jobs > 256 {
		return fmt.Errorf("bootloader-builder: jobs must be between 1 and 256, got %d", bb.config.Jobs)
	}
	return nil
}

// Cleanup releases plugin resources.
func (bb *BootloaderBuilderPlugin) Cleanup() error { return nil }

// Hooks returns lifecycle hook registrations.
func (bb *BootloaderBuilderPlugin) Hooks() []plugin.HookRegistration {
	return []plugin.HookRegistration{
		{
			Event:    plugin.PreBootloaderBuild,
			Handler:  bb.onPreBuild,
			Priority: 8,
			Required: true,
		},
		{
			Event:    plugin.PostBootloaderBuild,
			Handler:  bb.onPostBuild,
			Priority: 5,
			Required: false,
		},
	}
}

// onPreBuild validates that firmware blobs are available before build starts.
func (bb *BootloaderBuilderPlugin) onPreBuild(ctx context.Context, evt *plugin.Event) error {
	if evt.Metadata == nil {
		evt.Metadata = make(map[string]interface{})
	}

	evt.Metadata[plugin.MetadataStartTime] = time.Now().Unix()
	evt.Metadata["bootloader.uboot_version"] = bb.config.UBootVersion

	// Validate SoC blobs are accessible (check registry, not download yet)
	if bb.config.SoCIdentifier != "" {
		knownBlobs := bsp.GetBlobsForSoC(bb.config.SoCIdentifier)
		evt.Metadata["bootloader.firmware_blob_count"] = len(knownBlobs)
	}

	return nil
}

// onPostBuild collects artifact metadata after a successful build.
func (bb *BootloaderBuilderPlugin) onPostBuild(ctx context.Context, evt *plugin.Event) error {
	if evt.Metadata == nil {
		evt.Metadata = make(map[string]interface{})
	}

	evt.Metadata[plugin.MetadataEndTime] = time.Now().Unix()

	if startTime, ok := evt.Metadata[plugin.MetadataStartTime].(int64); ok {
		evt.Metadata[plugin.MetadataDuration] = time.Now().Unix() - startTime
	}

	// Record expected output binary
	evt.Metadata["bootloader.output_binary"] = bb.config.OutputBinary

	return nil
}

// --- Build Methods ---

// Build executes the full U-Boot build pipeline.
func (bb *BootloaderBuilderPlugin) Build(ctx context.Context, buildDir string) error {
	if buildDir == "" {
		buildDir = bb.config.BuildDir
	}
	if buildDir == "" {
		return fmt.Errorf("build directory not configured")
	}

	if bb.exec == nil {
		return fmt.Errorf("executor not initialized; ensure app passes executor to plugin")
	}

	// Step 1: Ensure source is available
	if err := bb.ensureSource(ctx, buildDir); err != nil {
		return fmt.Errorf("U-Boot source preparation failed: %w", err)
	}

	// Step 2: Download firmware blobs
	if _, err := bb.ensureFirmwareBlobs(ctx); err != nil {
		return fmt.Errorf("firmware blob preparation failed: %w", err)
	}

	// Step 3: Apply patches if configured
	if err := bb.applyPatches(ctx, buildDir); err != nil {
		return fmt.Errorf("patch application failed: %w", err)
	}

	// Step 4: Configure with defconfig
	if err := bb.configure(ctx, buildDir); err != nil {
		return fmt.Errorf("U-Boot configuration failed: %w", err)
	}

	// Step 5: Compile
	if err := bb.compile(ctx, buildDir); err != nil {
		return fmt.Errorf("U-Boot compilation failed: %w", err)
	}

	// Step 6: Validate output
	if err := bb.validateOutput(buildDir); err != nil {
		return fmt.Errorf("U-Boot output validation failed: %w", err)
	}

	return nil
}

// FetchBlobs downloads all required firmware blobs for the configured SoC.
func (bb *BootloaderBuilderPlugin) FetchBlobs(ctx context.Context) (map[string]string, error) {
	return bb.ensureFirmwareBlobs(ctx)
}

// Clean removes U-Boot build artifacts from the build directory.
func (bb *BootloaderBuilderPlugin) Clean(ctx context.Context, buildDir string) error {
	if bb.exec == nil {
		return fmt.Errorf("executor not initialized")
	}
	return bb.exec.Run(ctx, "make", "-C", buildDir, "mrproper")
}

// Configure runs only the defconfig step without a full build.
func (bb *BootloaderBuilderPlugin) Configure(ctx context.Context, buildDir string) error {
	if buildDir == "" {
		buildDir = bb.config.BuildDir
	}
	return bb.configure(ctx, buildDir)
}

// --- Internal helpers ---

func (bb *BootloaderBuilderPlugin) ensureSource(ctx context.Context, buildDir string) error {
	// Check if already cloned
	if _, err := os.Stat(filepath.Join(buildDir, "Makefile")); err == nil {
		// Already cloned - fetch latest if needed
		return bb.exec.Run(ctx, "git", "-C", buildDir, "fetch", "--tags")
	}

	// Clone U-Boot repo
	if err := os.MkdirAll(filepath.Dir(buildDir), 0755); err != nil {
		return fmt.Errorf("failed to create build parent dir: %w", err)
	}

	return bb.exec.Run(ctx, "git", "clone",
		"--depth=1",
		"--branch", bb.config.UBootVersion,
		bb.config.UBootRepo,
		buildDir,
	)
}

func (bb *BootloaderBuilderPlugin) ensureFirmwareBlobs(ctx context.Context) (map[string]string, error) {
	if bb.config.SoCIdentifier == "" {
		return nil, nil
	}

	knownBlobs := bsp.GetBlobsForSoC(bb.config.SoCIdentifier)
	if len(knownBlobs) == 0 {
		// Not a known SoC - skip blob download
		return nil, nil
	}

	return bb.firmwareMgr.DownloadBlobs(knownBlobs)
}

func (bb *BootloaderBuilderPlugin) applyPatches(ctx context.Context, buildDir string) error {
	if bb.config.PatchDir == "" {
		return nil // No patch dir configured
	}
	entries, err := os.ReadDir(bb.config.PatchDir)
	if os.IsNotExist(err) {
		return nil // No patches - OK
	}
	if err != nil {
		return fmt.Errorf("failed to read patch dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".patch" {
			continue
		}
		patchPath := filepath.Join(bb.config.PatchDir, entry.Name())
		if err := bb.exec.Run(ctx, "git", "-C", buildDir, "apply", patchPath); err != nil {
			return fmt.Errorf("failed to apply patch %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func (bb *BootloaderBuilderPlugin) configure(ctx context.Context, buildDir string) error {
	if bb.config.Defconfig == "" {
		return nil // No defconfig set — skip configuration step
	}
	return bb.exec.Run(ctx, "make",
		"-C", buildDir,
		fmt.Sprintf("CROSS_COMPILE=%s", bb.config.CrossCompile),
		bb.config.Defconfig,
	)
}

func (bb *BootloaderBuilderPlugin) compile(ctx context.Context, buildDir string) error {
	return bb.exec.Run(ctx, "make",
		"-C", buildDir,
		fmt.Sprintf("-j%d", bb.config.Jobs),
		fmt.Sprintf("CROSS_COMPILE=%s", bb.config.CrossCompile),
	)
}

func (bb *BootloaderBuilderPlugin) validateOutput(buildDir string) error {
	outputPath := filepath.Join(buildDir, bb.config.OutputBinary)
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("expected output binary not found: %s", outputPath)
	}
	return nil
}

// SetExecutor injects the command executor.
// Called during app initialization.
func (bb *BootloaderBuilderPlugin) SetExecutor(exec executor.Executor) {
	bb.exec = exec
}

// GetConfig returns the current bootloader builder configuration.
func (bb *BootloaderBuilderPlugin) GetConfig() *BootloaderBuilderConfig {
	return bb.config
}

// RunTask implements plugin.TaskRunner.
// It delegates bootloader build work to the Build method.
func (bb *BootloaderBuilderPlugin) RunTask(ctx context.Context, taskID string, config map[string]any) error {
	buildDir, _ := config["build_dir"].(string)
	if buildDir == "" {
		buildDir = "build/u-boot"
	}
	return bb.Build(ctx, buildDir)
}
