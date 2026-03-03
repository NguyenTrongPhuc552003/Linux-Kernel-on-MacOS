// Package config provides configuration management for elmos.
package config

// Default values for configuration.
const (
	// DefaultImageSize is the default sparse image size.
	DefaultImageSize = "40G"
	// DefaultVolumeName is the default volume name for the disk image.
	DefaultVolumeName = "elmos"
	// MinimumImageSize is the minimum recommended image size in GB.
	MinimumImageSize = 40
	// DefaultArch is the default target architecture.
	DefaultArch = "arm64"
	// DefaultCrossPrefix is the default cross-compile prefix.
	DefaultCrossPrefix = "llvm-"
	// DefaultMemory is the default QEMU memory allocation.
	DefaultMemory = "2G"
	// DefaultGDBPort is the default GDB debugging port.
	DefaultGDBPort = 1234
	// DefaultSSHPort is the default SSH forwarding port.
	DefaultSSHPort = 2222
	// DefaultDebianMirror is the default Debian package mirror.
	DefaultDebianMirror = "http://deb.debian.org/debian"
	// DefaultGlibcVersion is the glibc version used for downloading elf.h.
	DefaultGlibcVersion = "2.42"
)

// RequiredPackage represents a Homebrew package dependency.
type RequiredPackage struct {
	Name        string
	Description string
	Category    string
	Required    bool
}

// RequiredPackages lists all Homebrew dependencies for elmos.
var RequiredPackages = []RequiredPackage{
	// Core build tools
	{"llvm", "LLVM/Clang toolchain", "Build Tools", true},
	{"lld", "LLVM linker", "Build Tools", true},
	{"gnu-sed", "GNU sed (kernel requires it)", "Build Tools", true},
	{"make", "GNU make 4.0+", "Build Tools", true},
	{"libelf", "ELF library", "Build Tools", true},
	{"git", "Git version control", "Build Tools", true},
	{"qemu", "QEMU emulator", "Virtualization", true},
	{"fakeroot", "Fake root for packaging", "Build Tools", true},
	{"e2fsprogs", "ext4 filesystem tools", "Build Tools", true},
	{"wget", "File downloader", "Build Tools", false},
	{"coreutils", "GNU core utilities", "Build Tools", true},
	{"go", "Go programming language", "Build Tools", true},
	{"go-task", "Go task runner", "Build Tools", true},
	// Crosstool-ng dependencies (optional, for building custom toolchains)
	{"binutils", "GNU binary utilities (objcopy)", "Toolchain Dependencies", false},
	{"gcc", "GNU Compiler Collection (for ct-ng builds)", "Toolchain Dependencies", false},
	{"gmp", "GNU Multiple Precision library", "Toolchain Dependencies", false},
	{"mpfr", "GNU MPFR library", "Toolchain Dependencies", false},
	{"libmpc", "GNU MPC library", "Toolchain Dependencies", false},
	{"isl", "Integer Set Library", "Toolchain Dependencies", false},
	{"texinfo", "GNU documentation system", "Toolchain Dependencies", false},
	{"bison", "Parser generator", "Toolchain Dependencies", false},
	{"gawk", "GNU AWK", "Toolchain Dependencies", false},
	{"autoconf", "Autoconf for ct-ng bootstrap", "Toolchain Dependencies", false},
	{"automake", "Automake for ct-ng bootstrap", "Toolchain Dependencies", false},
	{"libtool", "GNU Libtool", "Toolchain Dependencies", false},
	{"ncurses", "Terminal UI library (menuconfig)", "Toolchain Dependencies", false},
	{"xz", "XZ compression", "Toolchain Dependencies", false},
}

// RequiredHeaders lists header files that should exist in assets/libraries/.
var RequiredHeaders = []string{
	"elf.h",
	"byteswap.h",
}

// ValidBuildTargets lists valid kernel build targets.
var ValidBuildTargets = map[string]bool{
	"Image":           true,
	"zImage":          true, // ARM32
	"dtbs":            true,
	"modules":         true,
	"modules_prepare": true,
	"all":             true,
	"vmlinux":         true,
}

// KernelConfigTypes lists valid kernel configuration types.
var KernelConfigTypes = []string{
	"defconfig",
	"tinyconfig",
	"kvm_guest.config",
	"menuconfig",
	"xconfig",
	"nconfig",
	"oldconfig",
	"olddefconfig",
	"allnoconfig",
	"allyesconfig",
	"allmodconfig",
	"localmodconfig",
	"localyesconfig",
}

// DefaultMachine returns the default machine definition for rock5b_plus.
// This is the example machine used for testing and as a reference.
func DefaultMachine() *MachineDefinition {
	return &MachineDefinition{
		Name:         "rock5b_plus",
		Manufacturer: "Radxa",
		Description:  "Radxa ROCK 5B+ with Rockchip RK3588",
		Tags: []string{
			"arm64", "armv8", "rockchip", "rk3588", "production-ready",
		},
		Kernel: MachineKernelConfig{
			Arch:      "arm64",
			Defconfig: "rock5b_defconfig",
			DeviceTrees: []string{
				"rk3588-rock-5b.dtb",
			},
		},
		Bootloader: MachineBootloaderConfig{
			Type:      "u-boot",
			Repo:      "https://github.com/u-boot/u-boot.git",
			Version:   "v2024.01",
			Defconfig: "rock5b_plus_defconfig",
			Binary:    "u-boot.itb",
			Firmware: []FirmwareBlob{
				{
					Name:     "DDR Initialization",
					Src:      "rk3588_ddr.bin",
					BlobName: "rk3588_ddr.bin",
					Required: true,
				},
				{
					Name:     "Miniloader",
					Src:      "rk3588_miniloader.elf",
					BlobName: "rk3588_miniloader.elf",
					Required: true,
				},
			},
			Flash: FlashConfig{
				Method:        "rkdevtool",
				OffsetMb:      0,
				DevicePattern: "/dev/sd*",
			},
		},
		QEMU: MachineQEMUConfig{
			System:  "qemu-system-aarch64",
			Machine: "virt",
			CPU:     "cortex-a72",
			Memory:  "2G",
			Network: "user",
			Console: "ttyAMA0",
			SMP:     2,
		},
		Rootfs: MachineRootfsConfig{
			ExtraPackages: []string{
				"u-boot-tools",
				"openssh-server",
			},
			KernelModules: []string{
				"rk_crypto",
			},
		},
	}
}
