// Package emulator provides QEMU emulation orchestration for elmos.
package emulator

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	elconfig "github.com/NguyenTrongPhuc552003/elmos/internal/config"
	elcontext "github.com/NguyenTrongPhuc552003/elmos/internal/context"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/executor"
	"github.com/NguyenTrongPhuc552003/elmos/internal/infra/filesystem"
)

// RunOptions contains options for running QEMU.
type RunOptions struct {
	Debug     bool // Enable GDB stub
	Graphical bool // Use graphical display instead of serial console
}

// QEMURunner orchestrates QEMU execution.
type QEMURunner struct {
	exec executor.Executor
	fs   filesystem.FileSystem
	cfg  *elconfig.Config
	ctx  *elcontext.Context
}

// NewQEMURunner creates a new QEMURunner with the given dependencies.
func NewQEMURunner(exec executor.Executor, fs filesystem.FileSystem, cfg *elconfig.Config, ctx *elcontext.Context) *QEMURunner {
	return &QEMURunner{
		exec: exec,
		fs:   fs,
		cfg:  cfg,
		ctx:  ctx,
	}
}

// Run starts QEMU with the built kernel.
func (q *QEMURunner) Run(ctx context.Context, opts RunOptions) error {
	archCfg := q.cfg.GetArchConfig()
	if archCfg == nil {
		return fmt.Errorf("unsupported architecture for QEMU: %s", q.cfg.Build.Arch)
	}

	// Check QEMU binary
	if _, err := q.exec.LookPath(archCfg.QEMUBinary); err != nil {
		return fmt.Errorf("QEMU not found: %s (run 'brew install qemu')", archCfg.QEMUBinary)
	}

	// Check kernel image
	kernelImage := q.ctx.GetKernelImage()
	if !q.fs.Exists(kernelImage) {
		return fmt.Errorf("kernel image not found: %s (run 'elmos build')", kernelImage)
	}

	// Check disk image
	if !q.fs.Exists(q.cfg.Paths.DiskImage) {
		return fmt.Errorf("disk image not found: %s (run 'elmos rootfs create')", q.cfg.Paths.DiskImage)
	}

	// Prepare modules sync script
	if err := q.prepareModulesSync(); err != nil {
		// Non-fatal, just warn
		fmt.Printf("Warning: Failed to prepare module sync: %v\n", err)
	}

	// Build QEMU command arguments
	args := q.buildArgs(archCfg, kernelImage, opts)

	// Execute QEMU
	return q.executeQEMU(ctx, archCfg.QEMUBinary, args, opts.Debug)
}

// Debug starts QEMU in debug mode and waits for GDB connection.
func (q *QEMURunner) Debug(ctx context.Context, graphical bool) error {
	return q.Run(ctx, RunOptions{
		Debug:     true,
		Graphical: graphical,
	})
}

// ConnectGDB launches cross-GDB and connects to a running QEMU instance.
func (q *QEMURunner) ConnectGDB() error {
	archCfg := q.cfg.GetArchConfig()
	if archCfg == nil {
		return fmt.Errorf("unsupported architecture for GDB: %s", q.cfg.Build.Arch)
	}

	if archCfg.GDBBinary == "" {
		return fmt.Errorf("GDB not configured for architecture: %s", q.cfg.Build.Arch)
	}

	// Get full path for GDB
	gdbPath, err := q.exec.LookPath(archCfg.GDBBinary)
	if err != nil {
		return fmt.Errorf("cross-GDB not found: %s", archCfg.GDBBinary)
	}

	vmlinux := q.ctx.GetVmlinux()
	if !q.fs.Exists(vmlinux) {
		return fmt.Errorf("vmlinux not found: %s", vmlinux)
	}

	// Build GDB args
	args := []string{
		gdbPath,
		vmlinux,
		"-ex", fmt.Sprintf("target remote localhost:%d", q.cfg.QEMU.GDBPort),
		"-ex", "layout src",
		"-ex", "break start_kernel",
	}

	// Use syscall.Exec to replace current process with GDB
	env := os.Environ()
	return q.exec.Exec(gdbPath, args, env)
}

// buildArgs constructs the QEMU command line arguments.
func (q *QEMURunner) buildArgs(archCfg *elconfig.ArchConfig, kernelImage string, opts RunOptions) []string {
	args := []string{
		"-m", q.cfg.QEMU.Memory,
		"-smp", fmt.Sprintf("%d", q.cfg.QEMU.SMP),
		"-kernel", kernelImage,
		"-machine", archCfg.QEMUMachine,
	}

	if archCfg.QEMUCPU != "" {
		args = append(args, "-cpu", archCfg.QEMUCPU)
	}

	if archCfg.QEMUBios != "" {
		args = append(args, "-bios", "default")
	}

	// Disk and networking
	args = append(args,
		"-drive", fmt.Sprintf("file=%s,format=raw,if=virtio", q.cfg.Paths.DiskImage),
		"-device", "virtio-net-device,netdev=net0",
		"-netdev", fmt.Sprintf("user,id=net0,hostfwd=tcp::%d-:22", q.cfg.QEMU.SSHPort),
	)

	// 9p share for modules
	args = append(args,
		"-fsdev", fmt.Sprintf("local,id=moddev,path=%s,security_model=none", q.cfg.Paths.ModulesDir),
		"-device", "virtio-9p-pci,fsdev=moddev,mount_tag=modules_mount",
	)

	// Boot parameters
	appendStr := "root=/dev/vda rw init=/init earlycon"

	// Display mode
	if opts.Graphical {
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
	if opts.Debug {
		args = append(args, "-s", "-S")
	}

	return args
}

// executeQEMU runs the QEMU binary with signal handling.
func (q *QEMURunner) executeQEMU(ctx context.Context, binary string, args []string, debug bool) error {
	// For now, we need to use the shell executor's direct run
	// since we need interactive stdin/stdout
	shellExec, ok := q.exec.(*executor.ShellExecutor)
	if !ok {
		return fmt.Errorf("executor does not support interactive mode")
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Create the command manually for interactive execution
	cmd := &struct {
		binary string
		args   []string
	}{binary, args}

	// Execute in separate goroutine
	done := make(chan error)
	go func() {
		done <- shellExec.Run(ctx, cmd.binary, cmd.args...)
	}()

	// Wait for either completion or signal
	select {
	case err := <-done:
		return err
	case <-sigChan:
		fmt.Println("\nReceived interrupt, stopping QEMU...")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// prepareModulesSync creates the guest sync script in the modules directory.
func (q *QEMURunner) prepareModulesSync() error {
	syncPath := filepath.Join(q.cfg.Paths.ModulesDir, "guesync.sh")

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

	return q.fs.WriteFile(syncPath, []byte(content), 0755)
}

// CheckDebugConfig verifies that the kernel has debugging enabled.
func (q *QEMURunner) CheckDebugConfig() error {
	if !q.ctx.HasConfig() {
		return fmt.Errorf("kernel config not found")
	}

	configFile := filepath.Join(q.cfg.Paths.KernelDir, ".config")
	content, err := q.fs.ReadFile(configFile)
	if err != nil {
		return err
	}

	configStr := string(content)
	hasDebugKernel := contains(configStr, "CONFIG_DEBUG_KERNEL=y")
	hasDWARF := contains(configStr, "CONFIG_DEBUG_INFO_DWARF_TOOLCHAIN_DEFAULT=y") ||
		contains(configStr, "CONFIG_DEBUG_INFO_DWARF5=y")

	if !hasDebugKernel || !hasDWARF {
		return fmt.Errorf("kernel debugging not enabled in .config (need CONFIG_DEBUG_KERNEL and DWARF info)")
	}

	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
