# Kernel Building

Build Linux kernels for ARM64, ARM, and RISC-V.

---

## Prerequisites

- Workspace initialized: `elmos init`
- Dependencies checked: `elmos doctor`
- Architecture set: `elmos arch arm64`

---

## Build Workflow

### 1. Set Architecture

```bash
elmos arch arm64    # or: arm, riscv
elmos arch show     # Show current
```

### 2. Configure Kernel

```bash
# Default config for architecture
elmos kernel config defconfig

# Interactive menu
elmos kernel config menuconfig

# Minimal config
elmos kernel config tinyconfig
```

**Valid config types:**

| Type               | Description          |
| ------------------ | -------------------- |
| `defconfig`        | Architecture default |
| `tinyconfig`       | Minimal kernel       |
| `menuconfig`       | Interactive menu     |
| `kvm_guest.config` | KVM optimized        |
| `oldconfig`        | Update existing      |
| `olddefconfig`     | Update with defaults |

### 3. Build

```bash
# Default targets (Image, dtbs, modules)
elmos kernel build

# Specific targets
elmos kernel build Image
elmos kernel build vmlinux

# Custom parallelism
elmos kernel build -j 8
```

**Valid build targets:**

| Target    | Description                   |
| --------- | ----------------------------- |
| `Image`   | Kernel image (arm64, riscv)   |
| `zImage`  | Compressed image (arm)        |
| `dtbs`    | Device tree blobs             |
| `modules` | Kernel modules                |
| `vmlinux` | Uncompressed kernel (for GDB) |

### 4. Verify Build

```bash
elmos status
```

Output:

```
Workspace Status:
  Kernel: ✓ Configured, ✓ Built
  Architecture: arm64
  Image: linux/arch/arm64/boot/Image
```

---

## BuildOptions (Developer Reference)

```cpp
// src/domain/builder/kernel.hpp
struct BuildOptions {
    int jobs = 0;                      // Parallel jobs (-j)
    std::vector<std::string> targets;  // Build targets
};
```

---

## Environment Variables

ELMOS automatically sets when building:

| Variable        | Value                  |
| --------------- | ---------------------- |
| `ARCH`          | Target architecture    |
| `CROSS_COMPILE` | Toolchain prefix       |
| `PATH`          | Prepends toolchain bin |

---

## Clean Build

```bash
elmos kernel clean    # make distclean
```

---

## Patches

Apply kernel patches:

```bash
# List available
elmos patch list

# Apply
elmos patch apply v6.18/generic/fix-copy-range
```

---

## Troubleshooting

| Issue           | Solution                                  |
| --------------- | ----------------------------------------- |
| "No toolchain"  | Run `elmos doctor`                        |
| Config errors   | Run `elmos kernel clean` then reconfigure |
| Build hangs     | Check disk space, reduce `-j`             |
| Missing headers | Run `elmos doctor --fix`                  |