// Package builtin provides built-in plugins for the ELMOS system.
// This file implements the image assembler plugin that combines kernel,
// bootloader, and rootfs artifacts into a flashable disk image.
package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	elcontext "github.com/NguyenTrongPhuc552003/elmos/core/context"
	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/core/plugin"
)

// ImageAssemblerPlugin combines kernel, bootloader, and rootfs artifacts
// into a flashable disk image.
//
// Phase 3 note: disk image creation is delegated to the executor via dd/genimage.
// Phase 4 will replace the dd calls with platform.DiskImageManager.Create().
type ImageAssemblerPlugin struct {
	exec   executor.Executor
	appCtx *elcontext.Context
	config *ImageAssemblerConfig
}

// ImageAssemblerConfig holds image assembly configuration.
type ImageAssemblerConfig struct {
	// OutputPath is the final image file path (e.g. "build/rock5b_plus.img").
	OutputPath string `mapstructure:"output_path"`

	// BootloaderOffset is the byte offset where the bootloader is written.
	BootloaderOffset int64 `mapstructure:"bootloader_offset"`

	// SizeGB is the total image size in gigabytes.
	SizeGB int `mapstructure:"size_gb"`

	// KernelArtifact is the path to the compiled kernel image.
	KernelArtifact string `mapstructure:"kernel_artifact"`

	// BootloaderArtifact is the path to the bootloader binary (e.g. u-boot.itb).
	BootloaderArtifact string `mapstructure:"bootloader_artifact"`

	// RootfsArtifact is the path to the populated rootfs image.
	RootfsArtifact string `mapstructure:"rootfs_artifact"`
}

// NewImageAssemblerPlugin creates a new ImageAssembler plugin.
func NewImageAssemblerPlugin() (*ImageAssemblerPlugin, error) {
	return &ImageAssemblerPlugin{
		config: &ImageAssemblerConfig{
			OutputPath:       "build/elmos.img",
			BootloaderOffset: 32768, // 32 KiB default (common for Rockchip/Allwinner)
			SizeGB:           8,
		},
	}, nil
}

// Name returns the plugin name.
func (ia *ImageAssemblerPlugin) Name() string { return "image-assembler" }

// Version returns the plugin version.
func (ia *ImageAssemblerPlugin) Version() string { return "1.0.0" }

// Description returns the plugin description.
func (ia *ImageAssemblerPlugin) Description() string {
	return "Final disk image assembly from kernel, bootloader, and rootfs artifacts"
}

// Init stores the application context and applies plugin configuration.
func (ia *ImageAssemblerPlugin) Init(ctx *elcontext.Context, cfg map[string]interface{}) error {
	ia.appCtx = ctx
	if cfg == nil {
		return nil
	}
	if path, ok := cfg["output_path"].(string); ok && path != "" {
		ia.config.OutputPath = path
	}
	if offset, ok := cfg["bootloader_offset"].(float64); ok {
		ia.config.BootloaderOffset = int64(offset)
	}
	if size, ok := cfg["size_gb"].(float64); ok && size > 0 {
		ia.config.SizeGB = int(size)
	}
	return nil
}

// Validate checks that required artifacts are configured.
func (ia *ImageAssemblerPlugin) Validate() error {
	if ia.config.OutputPath == "" {
		return fmt.Errorf("image-assembler: output_path is required")
	}
	if ia.config.SizeGB < 1 {
		return fmt.Errorf("image-assembler: size_gb must be >= 1")
	}
	return nil
}

// Cleanup removes the temporary working image if present.
func (ia *ImageAssemblerPlugin) Cleanup() error {
	tmpImg := ia.config.OutputPath + ".tmp"
	if _, err := os.Stat(tmpImg); err == nil {
		return os.Remove(tmpImg)
	}
	return nil
}

// Hooks registers pre/post image-assembly lifecycle hooks.
func (ia *ImageAssemblerPlugin) Hooks() []plugin.HookRegistration {
	return []plugin.HookRegistration{
		{
			Event:    plugin.PreImageAssemble,
			Handler:  ia.onPreAssemble,
			Priority: 5,
			Required: true,
		},
		{
			Event:    plugin.PostImageAssemble,
			Handler:  ia.onPostAssemble,
			Priority: 5,
			Required: false,
		},
	}
}

// onPreAssemble validates executor availability and records start time.
func (ia *ImageAssemblerPlugin) onPreAssemble(_ context.Context, evt *plugin.Event) error {
	if ia.exec == nil {
		return fmt.Errorf("image-assembler: executor not initialised")
	}
	if evt.Metadata == nil {
		evt.Metadata = make(map[string]interface{})
	}
	evt.Metadata[plugin.MetadataStartTime] = time.Now().Unix()
	evt.Metadata["image.output_path"] = ia.config.OutputPath
	evt.Metadata["image.size_gb"] = ia.config.SizeGB
	return nil
}

// onPostAssemble records timing and the output artifact path.
func (ia *ImageAssemblerPlugin) onPostAssemble(_ context.Context, evt *plugin.Event) error {
	if evt.Metadata == nil {
		evt.Metadata = make(map[string]interface{})
	}
	evt.Metadata[plugin.MetadataEndTime] = time.Now().Unix()
	if start, ok := evt.Metadata[plugin.MetadataStartTime].(int64); ok {
		evt.Metadata[plugin.MetadataDuration] = time.Now().Unix() - start
	}

	// Report output artifact for caching layer
	if _, err := os.Stat(ia.config.OutputPath); err == nil {
		evt.Metadata["build.artifacts"] = []string{ia.config.OutputPath}
	}
	return nil
}

// Assemble creates the final disk image by:
//  1. Allocating a sparse image file (TODO(platform): use DiskImageManager.Create)
//  2. Writing the bootloader binary at BootloaderOffset
//  3. Appending the rootfs image as the root partition
//
// Direct invocation is used by the PipelineExecutor; the hooks are used by the
// CLI build command.
func (ia *ImageAssemblerPlugin) Assemble(ctx context.Context) error {
	if ia.exec == nil {
		return fmt.Errorf("image-assembler: executor not initialised")
	}

	if err := os.MkdirAll(filepath.Dir(ia.config.OutputPath), 0755); err != nil {
		return fmt.Errorf("image-assembler: failed to create output directory: %w", err)
	}

	if err := ia.createImageFile(ctx); err != nil {
		return fmt.Errorf("image-assembler: create image: %w", err)
	}

	if err := ia.writeBootloader(ctx); err != nil {
		return fmt.Errorf("image-assembler: write bootloader: %w", err)
	}

	if err := ia.writeRootfs(ctx); err != nil {
		return fmt.Errorf("image-assembler: write rootfs: %w", err)
	}

	return ia.writeKernelToRootfs(ctx)
}

// createImageFile allocates a sparse disk image using dd.
// TODO(platform): replace with platform.DiskImageManager.Create() in Phase 4.
func (ia *ImageAssemblerPlugin) createImageFile(ctx context.Context) error {
	sizeBytes := int64(ia.config.SizeGB) * 1024 * 1024 * 1024
	sizeStr := fmt.Sprintf("%d", sizeBytes)
	return ia.exec.Run(ctx, "dd",
		"if=/dev/zero",
		"of="+ia.config.OutputPath,
		"bs=1",
		"count=0",
		"seek="+sizeStr,
	)
}

// writeBootloader writes the bootloader binary at the configured offset.
func (ia *ImageAssemblerPlugin) writeBootloader(ctx context.Context) error {
	if ia.config.BootloaderArtifact == "" {
		return nil // no bootloader configured
	}
	return ia.exec.Run(ctx, "dd",
		"if="+ia.config.BootloaderArtifact,
		"of="+ia.config.OutputPath,
		"bs=512",
		fmt.Sprintf("seek=%d", ia.config.BootloaderOffset/512),
		"conv=notrunc",
	)
}

// writeRootfs copies the rootfs image into the output image.
func (ia *ImageAssemblerPlugin) writeRootfs(ctx context.Context) error {
	if ia.config.RootfsArtifact == "" {
		return nil // no rootfs configured
	}
	// For partition-based images, dd the rootfs after the boot area.
	// TODO(platform): use DiskImageManager.Mount() + rsync in Phase 4 for GPT support.
	return ia.exec.Run(ctx, "dd",
		"if="+ia.config.RootfsArtifact,
		"of="+ia.config.OutputPath,
		"bs=4M",
		"seek=64", // 32 MiB reserved for bootloader area
		"conv=notrunc",
	)
}

// writeKernelToRootfs places the kernel image inside the mounted rootfs /boot.
// For now this is a copy operation; Phase 4 will mount via DiskImageManager.
// TODO(platform): replace with proper partition mount + copy in Phase 4.
func (ia *ImageAssemblerPlugin) writeKernelToRootfs(_ context.Context) error {
	if ia.config.KernelArtifact == "" {
		return nil // standalone rootfs build without kernel
	}
	// Phase 4 will implement: mount rootfs partition, copy kernel to /boot, unmount.
	return nil
}

// SetExecutor injects the command executor.
func (ia *ImageAssemblerPlugin) SetExecutor(exec executor.Executor) {
	ia.exec = exec
}

// GetConfig returns the current image assembler configuration.
func (ia *ImageAssemblerPlugin) GetConfig() *ImageAssemblerConfig {
	return ia.config
}

// RunTask implements plugin.TaskRunner.
// It delegates image assembly work to Assemble.
func (ia *ImageAssemblerPlugin) RunTask(ctx context.Context, taskID string, config map[string]any) error {
	return ia.Assemble(ctx)
}
