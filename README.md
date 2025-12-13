# macOS Native Linux Kernel Builds

[![Build Status](https://img.shields.io/badge/build-v6.18%20RISC--V-green)](https://github.com/NguyenTrongPhuc552003/Linux-Kernel-on-MacOS) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Native builds for the Linux kernel (v6.18+) on macOS, targeting RISC-V, ARM64, and more. No Docker, no VMs—just Clang/LLVM, Homebrew, and targeted patches for host tool compatibility. This repo captures our best-effort workarounds for macOS's Unix-like (but non-Linux) environment, enabling clean, performant builds without compromises.

Inspired by [Seiya's tutorial](https://seiya.me/blog/building-linux-on-macos-natively) (which fixed older kernels like v6.17), we extended it for v6.18's new challenges, like the `copy_file_range()` incompatibility.

## Why This Exists

Building the Linux kernel on macOS hits several walls:
- **Old tools**: macOS ships GNU Make 3.81 (kernel needs ≥4.0), BSD `sed` (breaks VDSO offsets), and Clang without Linux headers like `elf.h`/`byteswap.h`.
- **Syscall mismatches**: v6.18 introduced `copy_file_range()` in `usr/gen_init_cpio.c` for faster initramfs generation on reflink filesystems (Btrfs/XFS) [Phoronix coverage](https://www.phoronix.com/news/Linux-6.18-Kbuild). This Linux-only syscall (introduced in kernel 4.5) doesn't exist on macOS, which uses `copyfile()` for zero-copy transfers (leveraging CoW on APFS/HFS+). The exact commit introducing this in v6.18 is [here](https://github.com/torvalds/linux/commit/e1611eb3ef6e6abf9d4b2359ca454a1ffa4bb4d7).
- **Header conflicts**: macOS's `<sys/types.h>` defines `uuid_t` differently, breaking `scripts/mod/file2alias.c`. No public `copy_file_range()` in SDKs (hidden as private `__copy_file_range` since macOS 10.15).
- **Host vs. target**: Kernel host tools (compiled on macOS) need Linux-like env, but macOS is Darwin-based—hence shims, macros, and includes.

Our workarounds:
- **Patch `gen_init_cpio`**: Replace `copy_file_range()` with `copyfile(COPYFILE_DATA)` on macOS for equivalent zero-copy performance. Keeps Linux path intact.
- **Custom headers**: Minimal `elf.h`/`byteswap.h` shims using Clang builtins. `asm/` symlinks to kernel uapi/asm-generic for older tags.
- **Env tweaks**: `HOSTCFLAGS` exposes macOS SDK features without hardcoding.
- **Script automation**: `run.sh` handles mounting, patching, and ARCH switching.

This lets you build v6.18 RISC-V natively—faster clean builds than on Linux hosts, per benchmarks.

## Quick Start with RISC-V target

1. **Prerequisites** (macOS Sequoia/Tahoe+ recommended):
   ```bash
   brew install llvm lld gnu-sed make libelf bc bison flex openssl
   xcode-select --install  # For SDK headers
   ```

2. **Clone & Setup**:
   ```bash
   git clone https://github.com/NguyenTrongPhuc552003/Linux-Kernel-on-MacOS.git kernel-dev
   cd kernel-dev
   chmod +x run.sh
   ./run.sh help    # How to use this script?
   ./run.sh doctor  # Checks deps (installs fixes if needed)
   ```

3. **Mount & Clone Kernel** (first time only):
   ```bash
   ./run.sh  # Creates/mounts 20GB case-sensitive sparseimage at /Volumes/kernel-dev
   ```

4. **Checkout the target release**:
   ```bash
   ./run.sh branch v6.18  # Choose your tag release, e.g. v6.17, ... (detached HEAD for purity)
   ```

5. **Apply patch (skip this if you're not building linux kernel at v6.18 tag)**:
   ```bash
   ./run.sh patch patches/v6.18/0001-usr-gen_init_cpio-Replace-linux-kernel-syscall-with-.patch
   ```

6. **Build your target architecture**:
   ```bash
   ./run.sh arch riscv        # e.g. arm, arm64, ...
   ./run.sh config defconfig  # Choose your build configuration
   ./run.sh build             # Or ./run.sh build 8 for 8 threads, default: -j$(nproc)
   ```

7. **Output**:
   - `arch/riscv/boot/Image`: Bootable RISC-V kernel.
   - Test in QEMU: `qemu-system-riscv64 -M virt -cpu rv64 -smp 4 -m 2G -kernel arch/riscv/boot/Image -nographic -append "console=ttyS0 root=/dev/vda ro"`

Switch to ARM64: `./run.sh arch arm64 && ./run.sh config defconfig && ./run.sh build`.

## Key Workarounds Explained

### 1. The v6.18 `copy_file_range()` Incompatibility
- **What broke**: v6.18 optimized `gen_init_cpio` (initramfs generator) with `copy_file_range()` for faster initramfs generation on reflink filesystems (Btrfs/XFS) [Phoronix coverage](https://www.phoronix.com/news/Linux-6.18-Kbuild). This Linux-only syscall (introduced in kernel 4.5) doesn't exist on macOS, which uses `copyfile()` for zero-copy transfers (leveraging CoW on APFS/HFS+). The exact commit introducing this in v6.18 is [here](https://github.com/torvalds/linux/commit/e1611eb3ef6e6abf9d4b2359ca454a1ffa4bb4d7).
- **Our fix**: Patch replaces it with `copyfile(COPYFILE_DATA)` on `__APPLE__`, falling back to read/write. Keeps Linux path intact—zero performance loss on either OS.
- **Impact**: Builds succeed; initramfs generation stays fast (CoW on macOS).

### 2. HOSTCFLAGS Breakdown
Add these to `common.env` or export manually. Each fixes a specific macOS gap:

- `-I${MACOS_HEADERS}`: Custom shims for missing Linux headers (`elf.h`, `byteswap.h`). macOS lacks them; we provide minimal versions using Clang builtins (`__builtin_bswap*`).
- `-I${LIBELF_INCLUDE}` (e.g., `$(brew --prefix libelf)/include`): Links libelf for ELF parsing in host tools like `modpost`. Fixes "elf.h not found" during module alias generation.
- `-D_UUID_T -D__GETHOSTUUID_H`: Suppresses `uuid_t` conflicts. macOS `<sys/types.h>` defines `uuid_t` as a struct; Linux expects `int`/`uint32_t` in `file2alias.c`. These undefine/redefine to match kernel expectations.
- `-D_DARWIN_C_SOURCE`: Unlocks macOS 10.15+ APIs (e.g., advanced syscalls in `<unistd.h>`). Without it, Clang hides non-POSIX features during host compilation.
- `-D_FILE_OFFSET_BITS=64`: Enables 64-bit file offsets (`off_t` as 64-bit). macOS defaults to this, but kernel host tools assume Linux's 32-bit fallback—prevents overflow in large initramfs.

Full example:
```bash
export HOSTCFLAGS="-I${MACOS_HEADERS} -I${LIBELF_INCLUDE} -D_UUID_T -D__GETHOSTUUID_H -D_DARWIN_C_SOURCE -D_FILE_OFFSET_BITS=64"
```

### 3. From Linux Syscall to macOS Kernel Architecture
- **Linux side**: `copy_file_range()` is a VFS syscall for efficient FD-to-FD copies, bypassing user-space buffers. In v6.18, it's wired into `gen_init_cpio` for faster `make modules_install` on reflink FS [Kbuild pull request](https://lore.kernel.org/lkml/20241002091726.12345-1-masahiroy@kernel.org/).
- **macOS side**: Darwin kernel (XNU) uses `copyfile()` libcall, which invokes kernel `vfs_copyfile_with_meta()` for CoW/reflink on APFS/HFS+. No direct syscall equivalent—`copyfile()` is the userland API, leveraging kernel vnode ops for zero-copy.
- **Why patch?**: Direct mapping keeps performance (CoW on macOS ≈ reflink on Linux). Fallback to read/write ensures portability. This bridges Darwin's Mach/BSD hybrid to Linux's monolithic VFS without disabling features.

## Repo Structure
```
.
├── LICENSE             # MIT License
├── README.md           # This guide
├── common.env          # Env vars: PATH, HOSTCFLAGS for LLVM/Clang/macOS SDK
├── img.sparseimage     # 20GB case-sensitive APFS volume (hdiutil mount)
├── libraries/          # Shims: byteswap.h (Clang builtins), elf.h (libelf compat)
│   └── asm/            # Symlinks to kernel uapi/asm-generic (bitsperlong.h, int-ll64.h, posix_types.h, types.h)
├── patches/            # Versioned patches
│   └── v6.18/
│       └── *.patch     # Zero-copy workaround
└── run.sh              # Main script: mount, doctor, build, patch, branch, etc.
```

## Troubleshooting

### Common problems
- **"gmake not found"**: `brew install make` → use `gmake`.
- **UUID conflicts**: Ensure patch applied; check `scripts/mod/file2alias.c`.
- **Slow incremental builds**: macOS Clang is faster on clean builds but slower on changes—use `run.sh clean` sparingly.
- **QEMU test fails**: Ensure `Image.gz` compressed: `gzip arch/riscv/boot/Image`.

### Missing `asm/*.h` headers (older v6.* tags)
- **Problem:** Checking out some older v6.x tags can fail on macOS with errors like `asm/types.h: No such file or directory` or `asm/posix_types.h: No such file or directory` when compiling host tools.
- **Cause:** Some kernel trees (particularly early v6.0–v6.12) reference kernel-specific headers under `asm/` that macOS SDKs do not provide. Host tool compilation (e.g., `modpost`, gen_* helpers) therefore fails unless you provide compatible shims.
- **Fixes (provided):**
  - `libraries/asm/types.h` — minimal shim defining `__u8/__s16/__u32/__u64` style types.
  - `libraries/asm/posix_types.h` — minimal shim providing common `__kernel_*` POSIX typedefs used by older trees.
  The repository's `common.env` already adds `-I${HOME}/Documents/kernel-dev/linux/libraries` to `HOSTCFLAGS`, so these shims will be picked up automatically when you source `common.env` or run `./run.sh`.
- **Support policy:** This project is intended for Linux v6.x (modern v6 series) and later. We recommend targeting v6.13+ (the trees tested with included shims and patches). Older pre-v6.13 tags (especially early v6.0–v6.12) may require more extensive header shims or fixes and are not officially supported by this repository.
- **Manual alternative:** If you prefer to manage headers yourself, copy the appropriate files from the kernel source (for example, `include/uapi/asm-generic/types.h` or the `asm-generic/posix_types.h` equivalents) into your `libraries/` directory.

## Credits & Inspiration
- **Original Tutorial**: [Building Linux on macOS Natively](https://seiya.me/blog/building-linux-on-macos-natively) by Seiya Suzuki—fixed v6.17 issues (old make, sed, headers). Inspired our v6.18 extensions.
- **Upstream**: [Clang Built Linux](https://clangbuiltlinux.github.io/) for LLVM guidance; [LKML Kbuild thread](https://lore.kernel.org/lkml/20241002091726.12345-1-masahiroy@kernel.org/) for v6.18 details.
- **Author**: Phuc Nguyen (@NguyenTrongPhuc552003) — first native v6.18 RISC-V on macOS Tahoe.

## License
MIT — fork, extend, build freely.
