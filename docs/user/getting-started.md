# Getting Started

This tutorial walks through building your first Linux kernel with ELMOS.

## Prerequisites

Ensure ELMOS is [installed](installation.md) and a workspace initialized:

```bash
./build/bin/elmos init my_kernel
```

All subsequent commands automatically target the active workspace.

## Step 1: Select Architecture

Choose a target architecture (e.g., ARM64):

```bash
./build/bin/elmos arch arm64
```

Available architectures: `arm64`, `arm`, `riscv`

## Step 2: Clone Kernel Source

Clone the Linux kernel repository:

```bash
./build/bin/elmos kernel clone
```

## Step 3: Configure Kernel

Generate a default config:

```bash
./build/bin/elmos kernel config defconfig
```

For custom config, use menuconfig:

```bash
./build/bin/elmos kernel config menuconfig
```

## Step 4: Build Kernel

Build the kernel with the detected toolchain:

```bash
./build/bin/elmos kernel build
```

This may take 10-30 minutes depending on hardware.

## Step 5: Create RootFS

Create a Debian-based root filesystem:

```bash
./build/bin/elmos rootfs build
```

## Step 6: Run in QEMU

Boot the kernel in QEMU:

```bash
./build/bin/elmos qemu run
```

You should see the Linux boot process. Login with `root` (no password).

## Step 7: Debug (Optional)

For debugging, run with GDB stub:

```bash
./build/bin/elmos qemu debug
```

Connect GDB in another terminal:

```bash
gdb-multiarch vmlinux
(gdb) target remote :1234
```

## Next Steps

- [Develop modules](modules-and-apps.md)
- [Customize toolchains](toolchains.md)
- [Use the TUI](tui-guide.md) for interactive workflows

## Troubleshooting

If builds fail, check [Troubleshooting](troubleshooting.md) or run `./build/elmos doctor`.