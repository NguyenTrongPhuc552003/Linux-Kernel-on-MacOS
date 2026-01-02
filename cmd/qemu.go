// Package cmd implements the Cobra CLI commands for elmos.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// qemuCmd - QEMU management
var qemuCmd = &cobra.Command{
	Use:   "qemu",
	Short: "Run and debug kernel in QEMU",
	Long:  `Launch QEMU to run or debug the built Linux kernel.`,
}

var qemuRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run kernel in QEMU",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ctx.EnsureMounted(); err != nil {
			return err
		}
		debug, _ := cmd.Flags().GetBool("debug")
		verbose, _ := cmd.Flags().GetBool("verbose")
		return runQEMU(debug, verbose)
	},
}

var qemuDebugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Run kernel with GDB stub enabled",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ctx.EnsureMounted(); err != nil {
			return err
		}
		verbose, _ := cmd.Flags().GetBool("verbose")
		return runQEMU(true, verbose)
	},
}

var qemuGDBCmd = &cobra.Command{
	Use:   "gdb",
	Short: "Connect cross-GDB to QEMU",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQEMUGDB()
	},
}

func init() {
	qemuCmd.AddCommand(qemuRunCmd)
	qemuCmd.AddCommand(qemuDebugCmd)
	qemuCmd.AddCommand(qemuGDBCmd)

	qemuRunCmd.Flags().BoolP("debug", "d", false, "Enable GDB stub")
	qemuRunCmd.Flags().BoolP("verbose", "g", false, "Graphical mode (window)")
	qemuDebugCmd.Flags().BoolP("verbose", "g", false, "Graphical mode (window)")
}

// QEMUConfig holds architecture-specific QEMU settings
type QEMUArchConfig struct {
	Binary  string
	Machine string
	CPU     string
	Console string
	BIOS    string
}

var qemuArchConfigs = map[string]QEMUArchConfig{
	"riscv": {
		Binary:  "qemu-system-riscv64",
		Machine: "virt",
		CPU:     "rv64",
		Console: "ttyS0",
		BIOS:    "-bios default",
	},
	"arm64": {
		Binary:  "qemu-system-aarch64",
		Machine: "virt",
		CPU:     "cortex-a72",
		Console: "ttyAMA0",
		BIOS:    "",
	},
	"arm": {
		Binary:  "qemu-system-arm",
		Machine: "virt,highmem=off",
		CPU:     "cortex-a15",
		Console: "ttyAMA0",
		BIOS:    "",
	},
}

func runQEMU(debug, graphical bool) error {
	cfg := ctx.Config

	// Get arch-specific config
	archCfg, ok := qemuArchConfigs[cfg.Build.Arch]
	if !ok {
		return fmt.Errorf("unsupported architecture for QEMU: %s", cfg.Build.Arch)
	}

	// Check QEMU binary
	if _, err := exec.LookPath(archCfg.Binary); err != nil {
		return fmt.Errorf("QEMU not found: %s (run 'brew install qemu')", archCfg.Binary)
	}

	// Check kernel image
	kernelImage := ctx.GetKernelImage()
	if _, err := os.Stat(kernelImage); os.IsNotExist(err) {
		return fmt.Errorf("kernel image not found: %s (run 'elmos build')", kernelImage)
	}

	// Check disk image
	if _, err := os.Stat(cfg.Paths.DiskImage); os.IsNotExist(err) {
		return fmt.Errorf("disk image not found: %s (run 'elmos rootfs create')", cfg.Paths.DiskImage)
	}

	// Prepare modules sync script
	if err := prepareModulesSync(); err != nil {
		printWarn("Failed to prepare module sync: %v", err)
	}

	printStep("Starting QEMU for %s...", cfg.Build.Arch)
	if debug {
		printInfo("GDB stub active on port %d", cfg.QEMU.GDBPort)
	}

	// Build QEMU command
	args := []string{
		"-m", cfg.QEMU.Memory,
		"-smp", fmt.Sprintf("%d", cfg.QEMU.SMP),
		"-kernel", kernelImage,
		"-machine", archCfg.Machine,
	}

	if archCfg.CPU != "" {
		args = append(args, "-cpu", archCfg.CPU)
	}

	if archCfg.BIOS != "" {
		args = append(args, "-bios", "default")
	}

	// Disk and networking
	args = append(args,
		"-drive", fmt.Sprintf("file=%s,format=raw,if=virtio", cfg.Paths.DiskImage),
		"-device", "virtio-net-device,netdev=net0",
		"-netdev", "user,id=net0,hostfwd=tcp::2222-:22",
	)
	// 9p share for modules
	args = append(args,
		"-fsdev", fmt.Sprintf("local,id=moddev,path=%s,security_model=none", cfg.Paths.ModulesDir),
		"-device", "virtio-9p-pci,fsdev=moddev,mount_tag=modules_mount",
	)

	// Boot parameters
	appendStr := "root=/dev/vda rw init=/init earlycon"

	// Display mode
	if graphical {
		// Check required kernel configs
		if err := checkGraphicalConfigs(); err != nil {
			return err
		}
		args = append(args, "-display", "cocoa")

		args = append(args,
			"-device", "virtio-gpu-pci",
			"-device", "virtio-keyboard-pci",
			"-device", "virtio-mouse-pci",
		)
		appendStr += " console=tty0"
	} else {
		args = append(args,
			"-nographic",
			"-serial", "mon:stdio",
		)
		appendStr += fmt.Sprintf(" console=%s", archCfg.Console)
	}

	args = append(args, "-append", appendStr)

	// Debug flags
	if debug {
		args = append(args, "-s", "-S")
	}

	// Execute
	printInfo("Command: %s %v", archCfg.Binary, args)
	cmd := exec.Command(archCfg.Binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Execute in separate goroutine
	done := make(chan error)
	go func() {
		done <- cmd.Run()
	}()

	// Wait for either completion or signal
	select {
	case err := <-done:
		return err
	case <-sigChan:
		printInfo("Received interrupt, stopping QEMU...")
		cmd.Process.Signal(syscall.SIGTERM)
		return nil
	}
}

func prepareModulesSync() error {
	cfg := ctx.Config
	syncPath := fmt.Sprintf("%s/guesync.sh", cfg.Paths.ModulesDir)

	// Create robust sync script that loads all built modules
	content := `#!/bin/sh
# Auto-generated by elmos
echo '  [GUEST] Processing module queues...'

# Load all built modules found in the shared directory
if [ -d "/mnt/modules" ]; then
    echo "  [GUEST] Loading modules from /mnt/modules..."
    find /mnt/modules -name "*.ko" -type f -exec insmod {} \; 2>/dev/null
else
    echo "  [GUEST] Warning: /mnt/modules not found"
fi
`

	return os.WriteFile(syncPath, []byte(content), 0755)
}

func checkGraphicalConfigs() error {
	if !ctx.HasConfig() {
		return nil
	}

	// TODO: Implement stricter checks if needed
	// For now, we rely on the user having run 'elmos kernel config'
	return nil
}

func checkDebugConfig() error {
	if !ctx.HasConfig() {
		return fmt.Errorf("kernel config not found")
	}

	// We could grep the config file here, but for now we'll just warn
	// if we can't easily parse it.
	// Since we are in Go, we could read .config and check lines.

	configFile := filepath.Join(ctx.Config.Paths.KernelDir, ".config")
	content, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	configStr := string(content)
	if !strings.Contains(configStr, "CONFIG_DEBUG_KERNEL=y") &&
		!strings.Contains(configStr, "CONFIG_DEBUG_INFO_DWARF_TOOLCHAIN_DEFAULT=y") &&
		!strings.Contains(configStr, "CONFIG_DEBUG_INFO_DWARF5=y") {
		return fmt.Errorf("kernel debugging not enabled in .config (need CONFIG_DEBUG_KERNEL and DWARF info)")
	}

	return nil
}

func runQEMUGDB() error {
	cfg := ctx.Config

	// Check debug config first
	if err := checkDebugConfig(); err != nil {
		printWarn("Debug config check failed: %v", err)
		printInfo("Tip: Enable 'Kernel hacking' -> 'Kernel debugging' in menuconfig")
	}

	// Determine GDB binary
	gdbBinaries := map[string]string{
		"riscv": "riscv64-elf-gdb",
		"arm64": "aarch64-unknown-linux-gnu-gdb",
		"arm":   "arm-unknown-linux-gnueabihf-gdb",
	}

	gdbBin, ok := gdbBinaries[cfg.Build.Arch]
	if !ok {
		return fmt.Errorf("unsupported architecture for GDB: %s", cfg.Build.Arch)
	}

	// Get full path - syscall.Exec requires absolute path
	gdbPath, err := exec.LookPath(gdbBin)
	if err != nil {
		return fmt.Errorf("cross-GDB not found: %s", gdbBin)
	}

	vmlinux := ctx.GetVmlinux()
	if _, err := os.Stat(vmlinux); os.IsNotExist(err) {
		return fmt.Errorf("vmlinux not found: %s", vmlinux)
	}

	printStep("Connecting to localhost:%d...", cfg.QEMU.GDBPort)
	printInfo("Using GDB: %s", gdbPath)
	printInfo("Ctrl+C will interrupt target, 'quit' to exit GDB")

	// Build GDB args
	args := []string{
		gdbPath,
		vmlinux,
		"-ex", fmt.Sprintf("target remote localhost:%d", cfg.QEMU.GDBPort),
		"-ex", "layout src",
		"-ex", "break start_kernel",
	}

	// Use syscall.Exec to replace current process with GDB
	// This gives GDB full control of the terminal and signal handling
	env := os.Environ()
	return syscall.Exec(gdbPath, args, env)
}
