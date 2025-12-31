# ELMOS – Embedded Linux on MacOS

[![Build Status](https://img.shields.io/badge/build-v6.18%20ARM64-green)](https://github.com/NguyenTrongPhuc552003/elmos) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Native builds for the Linux kernel (v6.18+) on macOS, targeting RISC-V, ARM64, and more. No Docker, no VMs—just Clang/LLVM, Homebrew, and targeted patches for host tool compatibility. This repo captures our best-effort workarounds for macOS's Unix-like (but non-Linux) environment, enabling clean, performant builds without compromises.

Inspired by [Seiya's tutorial](https://seiya.me/blog/building-linux-on-macos-natively) (which fixed older kernels like v6.17), we extended it for v6.18's new challenges, like the `copy_file_range()` incompatibility.

## Why This Exists

Building the Linux kernel on macOS hits several walls:

- **Old tools**: macOS ships GNU Make 3.81 (kernel needs ≥4.0), BSD `sed` (breaks VDSO offsets), and Clang without Linux headers like `elf.h`/`byteswap.h`.
- **Syscall mismatches**: v6.18 introduced `copy_file_range()` in `usr/gen_init_cpio.c` for faster initramfs generation on reflink filesystems (Btrfs/XFS). This Linux-only syscall doesn't exist on macOS.
- **Header conflicts**: macOS's `<sys/types.h>` defines `uuid_t` differently, breaking `scripts/mod/file2alias.c`.
- **Host vs. target**: Kernel host tools (compiled on macOS) need Linux-like env, but macOS is Darwin-based—hence shims, macros, and includes.

Our workarounds:

- **Patch `gen_init_cpio`**: Replace `copy_file_range()` with `copyfile(COPYFILE_DATA)` on macOS.
- **Custom headers**: Minimal `elf.h`/`byteswap.h` shims using Clang builtins. `asm/` symlinks to kernel uapi/asm-generic.
- **Modern CLI**: Go-based `elmos` command replaces shell scripts with styled output and configuration management.

## Quick Start

### 1. Prerequisites (macOS Sequoia/Tahoe+ recommended)

```bash
brew tap messense/macos-cross-toolchains
brew install llvm lld gnu-sed make libelf git qemu fakeroot e2fsprogs coreutils go-task
xcode-select --install  # For SDK headers
```

### 2. Build ELMOS CLI

```bash
git clone https://github.com/NguyenTrongPhuc552003/elmos.git
cd elmos
task build
```

### 3. Check Environment

```bash
./elmos doctor  # Checks deps and taps
```

### 4. Initialize Workspace

```bash
./elmos init  # Creates/mounts 20GB case-sensitive sparseimage, clones kernel
```

### 5. Configure & Build

```bash
./elmos config set arch arm64     # Or: riscv, arm
./elmos kernel config             # Default: defconfig
./elmos build                     # Build Image, dtbs, modules
```

### 6. Create RootFS & Run

```bash
./elmos rootfs create             # Debian rootfs in ext4 disk image
./elmos qemu run                  # Boot in QEMU
./elmos qemu debug                # With GDB stub (port 1234)
```

## Interactive TUI

Run `elmos ui` for a menuconfig-style interface:

```
┌────────────────────────────────────────────────────┐
│  ELMOS - Embedded Linux on MacOS                   │
├────────────────────────────────────────────────────┤
│  ▼ Setup                                           │
│      Doctor (Check Environment)              [✓]   │
│      Init Workspace                          [○]   │
│      Configure (Arch, Jobs...)                     │
│  ▼ Build                                           │
│      Build Kernel                                  │
│      Build Modules                                 │
├────────────────────────────────────────────────────┤
│  ↑↓: Navigate  Enter: Select  q: Quit  ?: Help     │
└────────────────────────────────────────────────────┘
```

## Kernel Modules

```bash
# Prepare kernel headers (IMPORTANT)
./elmos build modules_prepare

# Build a module
./elmos module build hello-world

# Build all modules
./elmos module build

# Check status
./elmos module status
```

## Key Workarounds Explained

### 1. The v6.18 `copy_file_range()` Incompatibility

- **What broke**: v6.18 optimized `gen_init_cpio` with `copy_file_range()` for faster initramfs. This Linux-only syscall doesn't exist on macOS.
- **Our fix**: Patch replaces it with `copyfile(COPYFILE_DATA)` on `__APPLE__`. Keeps Linux path intact.
- **Apply**: `./elmos patch apply patches/v6.18/0001-usr-gen_init_cpio-Replace-linux-kernel-syscall-with-.patch`

### 2. HOSTCFLAGS Breakdown

The CLI automatically sets these for macOS compatibility:

- `-I${MACOS_HEADERS}`: Custom shims for missing Linux headers (`elf.h`, `byteswap.h`)
- `-I${LIBELF_INCLUDE}`: Links libelf for ELF parsing in host tools
- `-D_UUID_T -D__GETHOSTUUID_H`: Suppresses `uuid_t` conflicts
- `-D_DARWIN_C_SOURCE`: Unlocks macOS 10.15+ APIs
- `-D_FILE_OFFSET_BITS=64`: Enables 64-bit file offsets

### 3. Kernel Module Headers on macOS

macOS has no `linux-headers` package. The CLI handles this:

```bash
./elmos build modules_prepare  # Generates all necessary headers
./elmos module build my-driver # Now works
```

## Build System (Task)

Uses [Task](https://taskfile.dev) instead of Make:

```bash
task --list     # Show all targets
task build      # Build elmos binary
task clean      # Clean artifacts
task deps       # Download dependencies
task test       # Run tests
task release    # Multi-platform builds
```

## Repo Structure

```bash
.
├── apps/           # Userspace applications
├── cmd/            # CLI commands (Go)
├── internal/       # Core packages
│   ├── core/       # Config, context
│   └── tui/        # Interactive menu (Bubbletea)
├── libraries/      # Shims: byteswap.h, elf.h, asm/
├── modules/        # Sample kernel modules
├── patches/        # Versioned patches (v6.18/)
├── pkg/            # Public packages
├── tools/          # debootstrap
├── Taskfile.yml    # Build automation
└── elmos.yaml      # Configuration (optional)
```

## Troubleshooting

### Common Problems

- **"gmake not found"**: `brew install make` → use `gmake`
- **UUID conflicts**: Ensure patch applied; check `scripts/mod/file2alias.c`
- **QEMU test fails**: Ensure `Image.gz` compressed: `gzip arch/riscv/boot/Image`

### Missing `asm/*.h` Headers (older v6.* tags)

Some older v6.x tags fail with `asm/types.h: No such file or directory`. The `libraries/asm/` directory provides shims:

- `libraries/asm/types.h` — minimal shim defining `__u8/__s16/__u32/__u64`
- `libraries/asm/posix_types.h` — minimal shim for `__kernel_*` POSIX typedefs

**Support policy**: This project targets Linux v6.*. Older tags may require more extensive shims.

## Credits & Inspiration

- **Original Tutorial**: [Building Linux on macOS Natively](https://seiya.me/blog/building-linux-on-macos-natively) by Seiya Suzuki
- **Upstream**: [Clang Built Linux](https://clangbuiltlinux.github.io/) for LLVM guidance
- **Author**: Phuc Nguyen ([@NguyenTrongPhuc552003](https://github.com/NguyenTrongPhuc552003))

## License

MIT — fork, extend, build freely.
