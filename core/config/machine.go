// Package config provides configuration management for elmos, including machine definitions.
// This file defines the structure for board-specific configurations (machine definitions).
package config

import "fmt"

// MachineDefinition represents a complete board/machine configuration.
// It defines how to build, configure, and run a specific embedded board.
type MachineDefinition struct {
	// Metadata
	Name         string `mapstructure:"name" yaml:"name"`
	Manufacturer string `mapstructure:"manufacturer" yaml:"manufacturer"`
	Description  string `mapstructure:"description" yaml:"description"`

	// Classification tags for machine discovery
	Tags []string `mapstructure:"tags" yaml:"tags"`

	// Kernel configuration for this machine
	Kernel MachineKernelConfig `mapstructure:"kernel" yaml:"kernel"`

	// Bootloader configuration for this machine
	Bootloader MachineBootloaderConfig `mapstructure:"bootloader" yaml:"bootloader"`

	// QEMU emulation configuration (for testing without hardware)
	QEMU MachineQEMUConfig `mapstructure:"qemu" yaml:"qemu"`

	// Rootfs customization for this machine
	Rootfs MachineRootfsConfig `mapstructure:"rootfs" yaml:"rootfs"`
}

// MachineKernelConfig holds kernel-specific settings for a machine
type MachineKernelConfig struct {
	// ARCH value for kernel build (e.g., "arm64", "arm", "riscv")
	Arch string `mapstructure:"arch" yaml:"arch"`

	// Defconfig to use for this machine
	// Path relative to kernel/arch/{arch}/configs/
	Defconfig string `mapstructure:"defconfig" yaml:"defconfig"`

	// Device tree binaries to build and include
	DeviceTrees []string `mapstructure:"device_trees" yaml:"device_trees"`

	// Additional kernel config options to set (as CONFIG_* strings)
	// Applied after defconfig
	ConfigOptions []string `mapstructure:"config_options" yaml:"config_options"`

	// Patches to apply to kernel source before build
	Patches []string `mapstructure:"patches" yaml:"patches"`
}

// MachineBootloaderConfig holds bootloader-specific settings for a machine
type MachineBootloaderConfig struct {
	// Bootloader type: "u-boot" or "barebox" (Phase 2+)
	Type string `mapstructure:"type" yaml:"type"`

	// Repository URL for bootloader source
	Repo string `mapstructure:"repo" yaml:"repo"`

	// Version/tag of bootloader to use
	Version string `mapstructure:"version" yaml:"version"`

	// Defconfig for this machine
	Defconfig string `mapstructure:"defconfig" yaml:"defconfig"`

	// Output binary name after build
	Binary string `mapstructure:"binary" yaml:"binary"`

	// SoC-specific firmware blobs required (DDR init, miniloader, etc.)
	Firmware []FirmwareBlob `mapstructure:"firmware" yaml:"firmware"`

	// Patches to apply to bootloader source
	Patches []string `mapstructure:"patches" yaml:"patches"`

	// Flashing instructions
	Flash FlashConfig `mapstructure:"flash" yaml:"flash"`
}

// FirmwareBlob represents a required firmware binary for bootloader
type FirmwareBlob struct {
	// Human-readable name (e.g., "DDR initialization blob")
	Name string `mapstructure:"name" yaml:"name"`

	// Source identifier - can be:
	// - URL (https://...)
	// - Registry path (bsp-registry:rk3588/ddr.bin)
	// - Local path (firmware/rk3588_ddr.bin)
	Src string `mapstructure:"src" yaml:"src"`

	// Output binary name/path in bootloader build
	BlobName string `mapstructure:"blob_name" yaml:"blob_name"`

	// SHA256 checksum for validation
	Checksum string `mapstructure:"checksum" yaml:"checksum"`

	// Whether this blob is strictly required (build fails if missing)
	Required bool `mapstructure:"required" yaml:"required"`
}

// FlashConfig describes how to flash the final image to hardware
type FlashConfig struct {
	// Flashing method: "rkdevtool", "fastboot", "tftp", "dd", etc.
	Method string `mapstructure:"method" yaml:"method"`

	// Offset in bytes/sectors where bootloader starts
	OffsetMb int `mapstructure:"offset_mb" yaml:"offset_mb"`

	// Device pattern for auto-detection (e.g., "/dev/sd*" for USB drives on Linux)
	DevicePattern string `mapstructure:"device_pattern" yaml:"device_pattern"`

	// Optional command to run before flashing
	PreFlashCmd string `mapstructure:"pre_flash_cmd" yaml:"pre_flash_cmd"`

	// Optional command to run after flashing
	PostFlashCmd string `mapstructure:"post_flash_cmd" yaml:"post_flash_cmd"`
}

// MachineQEMUConfig holds QEMU emulation settings for this machine
type MachineQEMUConfig struct {
	// QEMU system binary (e.g., "qemu-system-aarch64", "qemu-system-arm")
	System string `mapstructure:"system" yaml:"system"`

	// QEMU machine type (e.g., "virt" for ARM64, "virt,highmem=off" for ARM)
	Machine string `mapstructure:"machine" yaml:"machine"`

	// CPU model (e.g., "cortex-a72", "cortex-a15")
	CPU string `mapstructure:"cpu" yaml:"cpu"`

	// Memory allocation (e.g., "2G")
	Memory string `mapstructure:"memory" yaml:"memory"`

	// Network type (e.g., "user", "tap", "none")
	Network string `mapstructure:"network" yaml:"network"`

	// Serial console device (e.g., "ttyAMA0", "ttyS0")
	Console string `mapstructure:"console" yaml:"console"`

	// Number of CPUs to emulate
	SMP int `mapstructure:"smp" yaml:"smp"`

	// Additional QEMU launch arguments
	ExtraArgs []string `mapstructure:"extra_args" yaml:"extra_args"`
}

// MachineRootfsConfig holds rootfs customization for this machine
type MachineRootfsConfig struct {
	// Additional packages to install in rootfs (beyond default)
	ExtraPackages []string `mapstructure:"extra_packages" yaml:"extra_packages"`

	// Kernel modules to include in rootfs
	KernelModules []string `mapstructure:"kernel_modules" yaml:"kernel_modules"`

	// Custom post-build script to run in rootfs
	PostBuildScript string `mapstructure:"post_build_script" yaml:"post_build_script"`

	// Device nodes to create in rootfs
	Devices []DeviceNode `mapstructure:"devices" yaml:"devices"`
}

// DeviceNode represents a device that should exist in rootfs
type DeviceNode struct {
	// Path in rootfs (e.g., "/dev/custom")
	Path string `mapstructure:"path" yaml:"path"`

	// Device type: "char", "block"
	Type string `mapstructure:"type" yaml:"type"`

	// Major device number
	Major int `mapstructure:"major" yaml:"major"`

	// Minor device number
	Minor int `mapstructure:"minor" yaml:"minor"`

	// File permissions (e.g., 0666)
	Permissions string `mapstructure:"permissions" yaml:"permissions"`
}

// Validate checks if a machine definition is valid and complete.
// Returns helpful error messages for missing required fields.
func (m *MachineDefinition) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("machine definition: name is required")
	}

	if m.Kernel.Arch == "" {
		return fmt.Errorf("machine definition %s: kernel.arch is required", m.Name)
	}

	if !IsValidArch(m.Kernel.Arch) {
		return fmt.Errorf("machine definition %s: unsupported architecture %s", m.Name, m.Kernel.Arch)
	}

	if m.Bootloader.Type == "" {
		return fmt.Errorf("machine definition %s: bootloader.type is required", m.Name)
	}

	if m.Bootloader.Type != "u-boot" && m.Bootloader.Type != "barebox" {
		return fmt.Errorf("machine definition %s: unsupported bootloader type %s (expected u-boot or barebox)",
			m.Name, m.Bootloader.Type)
	}

	if m.QEMU.System == "" {
		return fmt.Errorf("machine definition %s: qemu.system is required", m.Name)
	}

	return nil
}

// String returns a human-readable string representation of the machine
func (m *MachineDefinition) String() string {
	return fmt.Sprintf("%s (%s) by %s", m.Name, m.Kernel.Arch, m.Manufacturer)
}

// GetArchConfig returns the architecture config for this machine
func (m *MachineDefinition) GetArchConfig() *ArchConfig {
	return GetArchConfig(m.Kernel.Arch)
}
