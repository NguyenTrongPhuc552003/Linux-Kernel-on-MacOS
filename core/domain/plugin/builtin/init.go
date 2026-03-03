// Package builtin provides built-in plugins for the ELMOS system.
// This file registers the builtin plugin factories to avoid circular imports.
package builtin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/NguyenTrongPhuc552003/elmos/core/plugin"
)

// init registers all builtin plugin factories with the plugin system
func init() {
	// Register KernelBuilder plugin factory
	plugin.RegisterBuiltinFactory("kernel-builder", func() (plugin.Plugin, error) {
		return NewKernelBuilderPlugin()
	})

	// Register BspManager plugin factory
	plugin.RegisterBuiltinFactory("bsp-manager", func() (plugin.Plugin, error) {
		return NewBspManagerPlugin()
	})

	// Register BootloaderBuilder plugin factory
	plugin.RegisterBuiltinFactory("bootloader-builder", func() (plugin.Plugin, error) {
		return NewBootloaderBuilderPlugin()
	})

	// Register CachingBackend plugin factory
	plugin.RegisterBuiltinFactory("caching-backend", func() (plugin.Plugin, error) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("caching-backend: cannot determine home dir: %w", err)
		}
		cacheDir := filepath.Join(homeDir, ".elmos", "build-cache")
		return NewCachingBackendPlugin(cacheDir)
	})

	// Register RootfsBuilder plugin factory
	plugin.RegisterBuiltinFactory("rootfs-builder", func() (plugin.Plugin, error) {
		return NewRootfsBuilderPlugin()
	})

	// Register ImageAssembler plugin factory
	plugin.RegisterBuiltinFactory("image-assembler", func() (plugin.Plugin, error) {
		return NewImageAssemblerPlugin()
	})
}
