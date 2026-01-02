// Package config provides configuration management for elmos.
package config

// ArchConfig holds architecture-specific settings for building and emulation.
type ArchConfig struct {
	// Name is the architecture name (e.g., "arm64", "arm", "riscv").
	Name string
	// KernelArch is the kernel ARCH= value.
	KernelArch string
	// KernelImage is the output image name (e.g., "Image", "zImage").
	KernelImage string
	// DefaultTargets are the default build targets for this architecture.
	DefaultTargets []string

	// QEMU settings
	QEMUBinary  string // e.g., "qemu-system-aarch64"
	QEMUMachine string // e.g., "virt"
	QEMUCPU     string // e.g., "cortex-a72"
	QEMUBios    string // e.g., "-bios default" for RISC-V
	Console     string // e.g., "ttyAMA0"

	// Cross-compilation settings
	GCCBinary    string // e.g., "aarch64-unknown-linux-gnu-gcc"
	GDBBinary    string // e.g., "aarch64-unknown-linux-gnu-gdb"
	ToolchainPkg string // Homebrew package for the toolchain
}

// Architectures contains all supported architecture configurations.
var Architectures = map[string]*ArchConfig{
	"arm64": {
		Name:           "arm64",
		KernelArch:     "arm64",
		KernelImage:    "Image",
		DefaultTargets: []string{"Image", "dtbs", "modules"},
		QEMUBinary:     "qemu-system-aarch64",
		QEMUMachine:    "virt",
		QEMUCPU:        "cortex-a72",
		QEMUBios:       "",
		Console:        "ttyAMA0",
		GCCBinary:      "aarch64-unknown-linux-gnu-gcc",
		GDBBinary:      "aarch64-unknown-linux-gnu-gdb",
		ToolchainPkg:   "messense/macos-cross-toolchains/aarch64-unknown-linux-gnu",
	},
	"arm": {
		Name:           "arm",
		KernelArch:     "arm",
		KernelImage:    "zImage",
		DefaultTargets: []string{"zImage", "dtbs", "modules"},
		QEMUBinary:     "qemu-system-arm",
		QEMUMachine:    "virt,highmem=off",
		QEMUCPU:        "cortex-a15",
		QEMUBios:       "",
		Console:        "ttyAMA0",
		GCCBinary:      "arm-unknown-linux-gnueabihf-gcc",
		GDBBinary:      "arm-unknown-linux-gnueabihf-gdb",
		ToolchainPkg:   "messense/macos-cross-toolchains/arm-unknown-linux-gnueabihf",
	},
	"riscv": {
		Name:           "riscv",
		KernelArch:     "riscv",
		KernelImage:    "Image",
		DefaultTargets: []string{"Image", "dtbs", "modules"},
		QEMUBinary:     "qemu-system-riscv64",
		QEMUMachine:    "virt",
		QEMUCPU:        "rv64",
		QEMUBios:       "-bios default",
		Console:        "ttyS0",
		GCCBinary:      "riscv64-elf-gcc",
		GDBBinary:      "riscv64-elf-gdb",
		ToolchainPkg:   "", // Optional, uses LLVM
	},
}

// GetArchConfig returns the configuration for the specified architecture.
// Returns nil if the architecture is not supported.
func GetArchConfig(arch string) *ArchConfig {
	return Architectures[arch]
}

// SupportedArchitectures returns a list of supported architecture names.
func SupportedArchitectures() []string {
	archs := make([]string, 0, len(Architectures))
	for name := range Architectures {
		archs = append(archs, name)
	}
	return archs
}

// IsValidArch checks if the given architecture is supported.
func IsValidArch(arch string) bool {
	_, ok := Architectures[arch]
	return ok
}
