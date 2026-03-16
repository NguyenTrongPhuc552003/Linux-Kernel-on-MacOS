# How ELMOS's QEMU Command Works From Scratch (Legacy Golang Implementation)

Here is a full breakdown of how the QEMU command is assembled inside ELMOS — layer by layer — and then how you'd adapt it for a U-Boot-based boot flow for each supported architecture.

---

## 1. The Entry Point: `elmos qemu run`

The CLI command `elmos qemu run` is defined in `core/app/commands/qemu.go`. It calls `ctx.QEMURunner.Run()` after ensuring the workspace volume is mounted. [1](#0-0) 

The `QEMURunner` struct lives in the `core/domain/emulator` package and holds four dependencies: an executor, a filesystem, the global config, and the build context. [2](#0-1) 

---

## 2. Pre-flight Checks (inside `Run()`)

Before any command is assembled, `Run()` performs three validations:

1. **Architecture support** — `q.cfg.GetArchConfig()` looks up the `Architectures` map; returns `nil` for unsupported arches. [3](#0-2) 

2. **QEMU binary** — verifies the correct `qemu-system-*` binary is on `PATH`. [4](#0-3) 

3. **Kernel image and disk image** — confirms both files exist before attempting to launch. [5](#0-4) 

---

## 3. Architecture-Specific Parameters (the `ArchConfig` Map)

All per-architecture QEMU settings come from the `Architectures` map in `core/config/arch.go`. This is the single source of truth for binary name, machine type, CPU model, BIOS flag, and console device. [6](#0-5) 

| Field         | arm64                 | arm                | riscv                     |
| ------------- | --------------------- | ------------------ | ------------------------- |
| `QEMUBinary`  | `qemu-system-aarch64` | `qemu-system-arm`  | `qemu-system-riscv64`     |
| `QEMUMachine` | `virt`                | `virt,highmem=off` | `virt`                    |
| `QEMUCPU`     | `cortex-a72`          | `cortex-a15`       | `rv64`                    |
| `QEMUBios`    | *(empty)*             | *(empty)*          | `-bios default` (OpenSBI) |
| `Console`     | `ttyAMA0`             | `ttyAMA0`          | `ttyS0`                   |
| `KernelImage` | `Image`               | `zImage`           | `Image`                   |

The kernel image path is resolved via `GetKernelImage()`, which joins `KernelDir` + `arch/<KernelArch>/boot/<KernelImage>`. [7](#0-6) 

---

## 4. Building the Full Argument List (`buildArgs`)

The `buildArgs()` method assembles the final QEMU argument slice in this exact order:

### 4a. Core hardware flags [8](#0-7) 

- `-m <memory>` — default **2G** (from `QEMUConfig.Memory`)
- `-smp <ncpus>` — one thread per host CPU (from `QEMUConfig.SMP`)
- `-kernel <kernel_image>` — the compiled Linux `Image` / `zImage`
- `-machine <machine>` — e.g. `virt` or `virt,highmem=off`
- `-cpu <cpu>` — only appended when `QEMUCPU` is non-empty

### 4b. BIOS (RISC-V only) [9](#0-8) 

When `QEMUBios != ""` (only RISC-V), `-bios default` is added, which instructs QEMU to load the built-in **OpenSBI** firmware as the M-mode (machine-mode) firmware.

### 4c. Disk and networking [10](#0-9) 

- `-drive file=<disk.img>,format=raw,if=virtio`
- `-device virtio-net-device,netdev=net0`
- `-netdev user,id=net0,hostfwd=tcp::<SSHPort>-:22` (default SSH port **2222**)

### 4d. 9P virtio module share [11](#0-10) 

A Plan-9 (9p) virtio filesystem share exposes the host modules directory inside the guest at the tag `modules_mount`. The guest `/init` script mounts this at `/mnt/modules`. [12](#0-11) 

### 4e. Display and serial console [13](#0-12) 

- **Non-graphical** (default): `-nographic -serial mon:stdio`
- **Graphical**: `-display cocoa` + virtio-gpu/keyboard/mouse devices

### 4f. Kernel command line (`-append`) [14](#0-13) 

The kernel cmdline is:
```
root=/dev/vda rw init=/init earlycon console=<Console>
```
Where `<Console>` is `ttyAMA0` for ARM/ARM64 and `ttyS0` for RISC-V.

### 4g. Debug flags (optional) [15](#0-14) 

When `Debug: true`, QEMU adds `-s -S` — opening a GDB server on port **1234** (configurable via `QEMUConfig.GDBPort`) and halting until GDB connects.

---

## 5. Execution with Signal Handling

The assembled command is passed to `executeQEMU()`, which uses `ShellExecutor.Run()` in a goroutine and listens for `SIGINT`/`SIGTERM` to cleanly tear down QEMU. [16](#0-15) 

---

## 6. Default Configuration Values

Key QEMU defaults come from `core/config/defaults.go`: [17](#0-16) 

| Default      | Value   |
| ------------ | ------- |
| Memory       | `2G`    |
| GDB Port     | `1234`  |
| SSH Port     | `2222`  |
| Architecture | `arm64` |

---

## 7. The Full Reconstructed ELMOS QEMU Command (No U-Boot)

For **RISC-V**, the command ELMOS generates internally is equivalent to:

```
qemu-system-riscv64
  -m 2G
  -smp <host_ncpus>
  -kernel /Volumes/elmos/linux/arch/riscv/boot/Image
  -machine virt
  -cpu rv64
  -bios default                                          ← OpenSBI M-mode firmware
  -drive file=/Volumes/elmos/disk.img,format=raw,if=virtio
  -device virtio-net-device,netdev=net0
  -netdev user,id=net0,hostfwd=tcp::2222-:22
  -fsdev local,id=moddev,path=<modules_dir>,security_model=none
  -device virtio-9p-pci,fsdev=moddev,mount_tag=modules_mount
  -nographic
  -serial mon:stdio
  -append "root=/dev/vda rw init=/init earlycon console=ttyS0"
```

---

## 8. Manual QEMU Command With U-Boot as External Bootloader

> **Important Note:** ELMOS currently has **no U-Boot support** built in. The following commands are derived purely from the architecture parameters defined in ELMOS and the standard QEMU+U-Boot boot conventions.

### The Key Difference

In the current ELMOS flow, QEMU acts as the bootloader by passing `-kernel <linux_image>` directly. When using **U-Boot**, QEMU hands control to U-Boot first, and U-Boot then loads and boots the Linux kernel from the disk image. This means:
- `-kernel` no longer points to the Linux `Image` — it points to U-Boot (or is replaced by `-bios`)
- `-append` is **removed** — U-Boot uses its own `bootargs` environment variable
- The Linux kernel + DTB must be embedded in the disk image (e.g., on a FAT boot partition) where U-Boot can find them

### RISC-V (riscv64)

RISC-V has a 3-stage firmware hierarchy: **OpenSBI (M-mode) → U-Boot (S-mode) → Linux**. ELMOS's `-bios default` already covers OpenSBI. With U-Boot, you replace `-kernel <linux_image>` with `-kernel u-boot.bin` (U-Boot becomes the S-mode payload): [18](#0-17) 

```bash
qemu-system-riscv64 \
  -m 2G \
  -smp 4 \
  -machine virt \
  -cpu rv64 \
  -bios opensbi-riscv64-virt-fw_jump.bin \   # OpenSBI: replaces '-bios default'
  -kernel u-boot.bin \                        # U-Boot as S-mode payload
  -drive file=disk.img,format=raw,if=virtio \
  -device virtio-net-device,netdev=net0 \
  -netdev user,id=net0,hostfwd=tcp::2222-:22 \
  -fsdev local,id=moddev,path=./modules,security_model=none \
  -device virtio-9p-pci,fsdev=moddev,mount_tag=modules_mount \
  -nographic \
  -serial mon:stdio
  # No -append: U-Boot provides bootargs
```

U-Boot will then execute a boot script that does `virtio scan; load virtio 0:1 $kernel_addr_r Image; booti $kernel_addr_r - $fdtaddr`.

### AARCH64 (arm64)

ARM64 has no built-in BIOS on the `virt` machine. ELMOS leaves `QEMUBios` empty. With U-Boot, supply it as the firmware via `-bios`: [19](#0-18) 

```bash
qemu-system-aarch64 \
  -m 2G \
  -smp 4 \
  -machine virt \
  -cpu cortex-a72 \
  -bios u-boot.bin \     # U-Boot as the sole firmware; replaces '-kernel Image'
  -drive file=disk.img,format=raw,if=virtio \
  -device virtio-net-device,netdev=net0 \
  -netdev user,id=net0,hostfwd=tcp::2222-:22 \
  -fsdev local,id=moddev,path=./modules,security_model=none \
  -device virtio-9p-pci,fsdev=moddev,mount_tag=modules_mount \
  -nographic \
  -serial mon:stdio
  # No -append: U-Boot provides bootargs
```

### ARM 32-bit

ARM 32-bit uses `virt,highmem=off` and `cortex-a15`. U-Boot for the `virt` machine is passed via `-kernel` (not `-bios`) since the ARM virt machine does not expose a firmware slot: [20](#0-19) 

```bash
qemu-system-arm \
  -m 2G \
  -smp 4 \
  -machine virt,highmem=off \
  -cpu cortex-a15 \
  -kernel u-boot \       # U-Boot ELF/binary at the kernel slot; no -bios
  -drive file=disk.img,format=raw,if=virtio \
  -device virtio-net-device,netdev=net0 \
  -netdev user,id=net0,hostfwd=tcp::2222-:22 \
  -fsdev local,id=moddev,path=./modules,security_model=none \
  -device virtio-9p-pci,fsdev=moddev,mount_tag=modules_mount \
  -nographic \
  -serial mon:stdio
  # No -append: U-Boot provides bootargs
```

---

## Architecture Summary Diagram

```mermaid
flowchart TD
    "elmos qemu run" --> "QEMURunner.Run()"
    "QEMURunner.Run()" --> "GetArchConfig(arch)"
    "GetArchConfig(arch)" --> "Architectures map (arch.go)"
    "Architectures map (arch.go)" --> "arm64: qemu-system-aarch64"
    "Architectures map (arch.go)" --> "arm: qemu-system-arm"
    "Architectures map (arch.go)" --> "riscv: qemu-system-riscv64"
    "QEMURunner.Run()" --> "buildArgs(archCfg, kernelImage, opts)"
    "buildArgs(archCfg, kernelImage, opts)" --> "-m / -smp / -machine / -cpu"
    "buildArgs(archCfg, kernelImage, opts)" --> "-kernel kernelImage"
    "buildArgs(archCfg, kernelImage, opts)" --> "-bios default (RISC-V only)"
    "buildArgs(archCfg, kernelImage, opts)" --> "-drive disk.img (virtio)"
    "buildArgs(archCfg, kernelImage, opts)" --> "-netdev user + SSH fwd"
    "buildArgs(archCfg, kernelImage, opts)" --> "-fsdev 9p modules share"
    "buildArgs(archCfg, kernelImage, opts)" --> "-append root=/dev/vda..."
    "buildArgs(archCfg, kernelImage, opts)" --> "executeQEMU()"
    "executeQEMU()" --> "ShellExecutor.Run(qemu-system-*, args)"
```

---

## Notes

1. **No U-Boot Support in ELMOS Currently**: The `QEMUBios` field in `ArchConfig` is the only BIOS-related hook, and it's hardcoded to add `-bios default` (OpenSBI) for RISC-V only. There is no field or flag in ELMOS for specifying a U-Boot binary path. [9](#0-8) 

2. **RISC-V is the only arch needing two firmware stages**: RISC-V requires OpenSBI in M-mode before U-Boot (or Linux) can run in S-mode. The `-bios default` in ELMOS points to QEMU's bundled OpenSBI. ARM/ARM64 `virt` machines can boot directly from a U-Boot binary.

3. **The disk image must contain the kernel for U-Boot**: In the U-Boot flow, `-append` is removed because U-Boot reads `bootargs` from its own environment. The Linux `Image`/`zImage` and DTB must be placed on a FAT or ext4 partition inside `disk.img` that U-Boot can access via its `virtio` or `mmc` driver commands.

4. **9P module share and SSH forwarding are retained**: These are ELMOS infrastructure features not related to the bootloader and work identically with U-Boot. [21](#0-20) 

5. **GDB debug mode**: When using `elmos qemu debug`, the `-s -S` flags pause QEMU at CPU reset. With U-Boot, GDB would stop at the U-Boot entry point, not `start_kernel` — you'd need to `continue` past U-Boot's initialization. [15](#0-14)

### Citations

**File:** core/app/commands/qemu.go (L17-27)
```go
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run kernel",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ctx.AppContext.EnsureMounted(); err != nil {
				return err
			}
			ctx.Printer.Step("Starting QEMU...")
			return ctx.QEMURunner.Run(cmd.Context(), emulator.RunOptions{Graphical: graphical})
		},
	}
```

**File:** core/domain/emulator/qemu.go (L20-35)
```go
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
```

**File:** core/domain/emulator/qemu.go (L39-43)
```go
	archCfg := q.cfg.GetArchConfig()
	if archCfg == nil {
		return fmt.Errorf("unsupported architecture for QEMU: %s", q.cfg.Build.Arch)
	}

```

**File:** core/domain/emulator/qemu.go (L44-47)
```go
	// Check QEMU binary
	if _, err := q.exec.LookPath(archCfg.QEMUBinary); err != nil {
		return fmt.Errorf("QEMU not found: %s (run 'brew install qemu')", archCfg.QEMUBinary)
	}
```

**File:** core/domain/emulator/qemu.go (L49-58)
```go
	// Check kernel image
	kernelImage := q.ctx.GetKernelImage()
	if !q.fs.Exists(kernelImage) {
		return fmt.Errorf("kernel image not found: %s (run 'elmos build')", kernelImage)
	}

	// Check disk image
	if !q.fs.Exists(q.cfg.Paths.DiskImage) {
		return fmt.Errorf("disk image not found: %s (run 'elmos rootfs create')", q.cfg.Paths.DiskImage)
	}
```

**File:** core/domain/emulator/qemu.go (L119-128)
```go
	args := []string{
		"-m", q.cfg.QEMU.Memory,
		"-smp", fmt.Sprintf("%d", q.cfg.QEMU.SMP),
		"-kernel", kernelImage,
		"-machine", archCfg.QEMUMachine,
	}

	if archCfg.QEMUCPU != "" {
		args = append(args, "-cpu", archCfg.QEMUCPU)
	}
```

**File:** core/domain/emulator/qemu.go (L130-132)
```go
	if archCfg.QEMUBios != "" {
		args = append(args, "-bios", "default")
	}
```

**File:** core/domain/emulator/qemu.go (L134-145)
```go
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
```

**File:** core/domain/emulator/qemu.go (L148-167)
```go
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
```

**File:** core/domain/emulator/qemu.go (L169-174)
```go
	// Debug flags
	if opts.Debug {
		args = append(args, "-s", "-S")
	}

	return args
```

**File:** core/domain/emulator/qemu.go (L177-213)
```go
// executeQEMU runs the QEMU binary with signal handling.
func (q *QEMURunner) executeQEMU(ctx context.Context, binary string, args []string) error {
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
```

**File:** core/config/arch.go (L29-72)
```go
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
		ToolchainPkg:   "",
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
		GCCBinary:      "arm-cortex_a15-linux-gnueabihf-gcc",
		GDBBinary:      "arm-cortex_a15-linux-gnueabihf-gdb",
		ToolchainPkg:   "",
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
		GCCBinary:      "riscv64-unknown-linux-gnu-gcc",
		GDBBinary:      "riscv64-unknown-linux-gnu-gdb",
		ToolchainPkg:   "", // Optional, uses LLVM
	},
}
```

**File:** core/context/context.go (L137-143)
```go
func (ctx *Context) GetKernelImage() string {
	archCfg := ctx.Config.GetArchConfig()
	if archCfg == nil {
		return ""
	}
	return filepath.Join(ctx.Config.Paths.KernelDir, "arch", archCfg.KernelArch, "boot", archCfg.KernelImage)
}
```

**File:** assets/templates/init/init.sh.tmpl (L26-27)
```text
mkdir -p /mnt/modules
mount -t 9p -o trans=virtio modules_mount /mnt/modules 2>/dev/null || true
```

**File:** core/config/defaults.go (L16-24)
```go
	// DefaultMemory is the default QEMU memory allocation.
	DefaultMemory = "2G"
	// DefaultGDBPort is the default GDB debugging port.
	DefaultGDBPort = 1234
	// DefaultSSHPort is the default SSH forwarding port.
	DefaultSSHPort = 2222
	// DefaultDebianMirror is the default Debian package mirror.
	DefaultDebianMirror = "http://deb.debian.org/debian"
	// DefaultGlibcVersion is the glibc version used for downloading elf.h.
```
