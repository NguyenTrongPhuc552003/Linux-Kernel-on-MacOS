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

This lets you build v6.18 ARM64 natively—faster clean builds than on Linux hosts, per benchmarks.

## Quick Start with ARM64 target

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
   ./run.sh doctor  # Checks deps and taps (installs fixes if needed)
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
   ./run.sh arch arm64  # You can choose riscv, arm, ...
   ./run.sh config  # Default: defconfig, you can choose another
   ./run.sh build   # Or ./run.sh build 8 for 8 threads, default: -j$(nproc)
   ./run.sh rootfs  # Prepare & package Debian rootfs on an ext4 disk image   
   ```

7. **Output**:

   ```bash
   /Volumes/kernel-dev
   ├── disk.img                     # An ext4 disk image
   ├── linux/arch/arm64/boot/Image  # Our final kernel image
   └── rootfs                       # A minimal rootfs
   ```

8. **Launch in QEMU:**
   ```bash
   ./run.sh qemu     # Start running our Image (default: -nographic)
   ./run.sh qemu -d  # Or boot with GDB stub enabled (port: 2222)  [Experimental]
   ```

Switch to RISC-V: `./run.sh arch riscv && ./run.sh config && ./run.sh build`.

## Kernel Modules in QEMU

You can take the pre-example from modules directory to test module loading or create your own modules. To load modules into the guest kernel at boot time, follow these steps:

1. **Prepare Kernel Headers (IMPORTANT)**:

   ```bash
   ./run.sh module -e  # or ./run.sh module --headers
   ```

2. **Compile Modules**:

   ```bash
   ./run.sh module <module_name>  # if nothing specified, builds all modules in modules/
   ```

3. **Insert Modules to Queues**:

   ```bash
   ./run.sh module <module_name> -i  # if nothing specified, inserts all compiled modules
   ```

4. **Test it in QEMU**:

   ```bash
   ./run.sh qemu  # Boot the kernel with modules inserted at startup
   ```

   You should see the following lines in the QEMU console output:

   ```bash
   [  305.493398] hello_world: loading out-of-tree module taints kernel.
   [  305.499611]   [HELLO] Module loaded successfully!
   [  305.499720]   [HELLO] Hello from the macOS-built kernel module!
   ```

5. **Same with unloading modules**:

   ```bash
   ./run.sh module <module_name> -r  # if nothing specified, removes all inserted modules from queues
   ./run.sh qemu  # Boot the kernel with modules removed at startup
   ```

   You should also see the following lines in the QEMU console output:

   ```bash
   [ 5740.214740]   [HELLO] Module unloaded. Goodbye!
   ```

6. **Additional Options**:

- To clean compiled modules:

  ```bash
  ./run.sh module <module_name> -c  # if nothing specified, cleans all compiled modules
  ```

- To see the status of modules:

  ```bash
  ./run.sh module -s  # or ./run.sh module --status
  ```

- To see help information about module commands:

  ```bash
  ./run.sh module -h  # or ./run.sh module --help
  ```

Notes:

- Ensure that the kernel module's main source file is named `<module_name>.c` and is located in the `modules/<module_name>/` directory. You can refer to the provided `hello_world` module as an example.

- **(Optional)** You should also make sure that the kernel module's information (like `MODULE_LICENSE`, `MODULE_AUTHOR`, etc.) is correctly defined in the end of the module source file. Please refer to the `hello_world` module for guidance.

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
- `-D_FILE_OFFSET_BITS=64`: Enables 64-bit file offsets (`off_t` as 64-bit). macOS defaults to this, but kernel host tools assume Linux's 32-bit fallback—prevents overflow in large rootfs.

Full example:

```bash
export HOSTCFLAGS="-I${MACOS_HEADERS} -I${LIBELF_INCLUDE} -D_UUID_T -D__GETHOSTUUID_H -D_DARWIN_C_SOURCE -D_FILE_OFFSET_BITS=64"
```

### 3. From Linux Syscall to macOS Kernel Architecture

- **Linux side**: `copy_file_range()` is a VFS syscall for efficient FD-to-FD copies, bypassing user-space buffers. In v6.18, it's wired into `gen_init_cpio` for faster `make modules_install` on reflink FS [Kbuild pull request](https://lore.kernel.org/lkml/20241002091726.12345-1-masahiroy@kernel.org/).
- **macOS side**: Darwin kernel (XNU) uses `copyfile()` libcall, which invokes kernel `vfs_copyfile_with_meta()` for CoW/reflink on APFS/HFS+. No direct syscall equivalent—`copyfile()` is the userland API, leveraging kernel vnode ops for zero-copy.
- **Why patch?**: Direct mapping keeps performance (CoW on macOS ≈ reflink on Linux). Fallback to read/write ensures portability. This bridges Darwin's Mach/BSD hybrid to Linux's monolithic VFS without disabling features.

### 4. Kernel Module Headers on macOS

- **Problem**: We have no any official Homebrew package like `linux-headers` for macOS to compile out-of-tree kernel modules. That leads to missing headers while building modules from modules directory like pre-exmaple `hello_world` module.
- **Solution**:
  - We can resolve this by preparing kernel headers using `./run.sh module -e` command. This command will make a linux kernel build with `modules_prepare` target to generate all necessary headers for building out-of-tree kernel modules. The generated headers will be located at `linux/` directory.
  - Another same way is to run `./run.sh build <jobs> modules_prepare` command after configuring the kernel. This will also generate all necessary headers for building out-of-tree kernel modules.
- **Usage**: After preparing headers, you can compile your modules using `./run.sh module <module_name>` command normally.

## Repo Structure

```bash
.
├── LICENSE             # MIT License
├── README.md           # This guide
├── common.env          # Env vars: PATH, HOSTCFLAGS, ANSI Colors, and sources all modular scripts.
├── img.sparseimage     # 20GB case-sensitive APFS volume (hdiutil mount)
├── libraries/          # Shims: byteswap.h (Clang builtins), elf.h (libelf compat)
│   └── asm/            # Symlinks to kernel uapi/asm-generic (bitsperlong.h, int-ll64.h, posix_types.h, types.h)
├── modules/            # Sample kernel modules (hello_world/)
│   └── hello_world/
├── patches/            # Versioned patches
│   └── v6.18/
│       └── *.patch     # Zero-copy workaround
├── run.sh              # Main Dispatcher: Minimal script for command parsing; delegates all logic to scripts/*.
├── scripts/            # New: Contains all modular logic, sourced by common.env
│   ├── branch.sh       # Handles git branch/tag checkout, creation, and safe deletion.
│   ├── build.sh        # Handles ARCH persistence, make config, make build, and make clean.
│   ├── doctor.sh       # Handles environment/dependency checks.
│   ├── image.sh        # Handles sparse image mounting and unmounting.
│   ├── module.sh       # Handles kernel module header prep, build, insert, remove, clean, and status.
│   ├── patch.sh        # Handles git apply --3way for patch files.
│   ├── repo.sh         # Handles git status, clone, update, reset, and reinitialize.
│   ├── qemu.sh         # Launching QEMU with built disk.img and kernel Image
│   └── rootfs.sh       # RootFS and disk.img initialization using debootstrap tool
└── tools/
    └── debootstrap     # Debian bootstrap toolset
```

## Troubleshooting

### Common problems

- **"gmake not found"**: `brew install make` → use `gmake`.
- **UUID conflicts**: Ensure patch applied; check `scripts/mod/file2alias.c`.
- **Slow incremental builds**: macOS Clang is faster on clean builds but slower on changes—use `run.sh clean` sparingly.
- **QEMU test fails**: Ensure `Image.gz` compressed: `gzip arch/riscv/boot/Image`.

### Missing `asm/*.h` headers (older v6.\* tags)

- **Problem:** Checking out some older v6.x tags can fail on macOS with errors like `asm/types.h: No such file or directory` or `asm/posix_types.h: No such file or directory` when compiling host tools.
- **Cause:** Some kernel trees (particularly early v6.0–v6.12) reference kernel-specific headers under `asm/` that macOS SDKs do not provide. Host tool compilation (e.g., `modpost`, gen\_\* helpers) therefore fails unless you provide compatible shims.
- **Fixes (provided):**
  - `libraries/asm/types.h` — minimal shim defining `__u8/__s16/__u32/__u64` style types.
  - `libraries/asm/posix_types.h` — minimal shim providing common `__kernel_*` POSIX typedefs used by older trees.
    The repository's `common.env` already adds `-I${HOME}/Documents/kernel-dev/linux/libraries` to `HOSTCFLAGS`, so these shims will be picked up automatically when you source `common.env` or run `./run.sh`.
- **Support policy:** This project is intended for Linux v6.x (modern v6 series) and later. We recommend targeting v6.13+ (the trees tested with included shims and patches). Older pre-v6.13 tags (especially early v6.0–v6.12) may require more extensive header shims or fixes and are not officially supported by this repository.
- **Manual alternative:** If you prefer to manage headers yourself, copy the appropriate files from the kernel source (for example, `include/uapi/asm-generic/types.h` or the `asm-generic/posix_types.h` equivalents) into your `libraries/` directory.

## Roadmap

- [x] **v1.0.0**: Initial native kernel build success.
- [x] **v1.1.0**: Modular scripts, automated debootstrap, and stable Initramfs boot.
- [x] **v2.0.0**: **dev/rootfs Release** 
  Persistent Storage Mode — Full transition to EXT4 disk images, one-time second-stage debootstrap via smart `/init`, faster and stable booting.

- [ ] **v2.1.0**: Professional QEMU + GDB Integration **(soon)**  
  One-command debugging experience via `./run.sh qemu -d`:
  - Automatic cross-toolchain GDB selection (riscv64-elf-gdb / aarch64-elf-gdb / arm-none-eabi-gdb)
  - Seamless macOS Terminal integration: GDB in foreground, QEMU in background window
  - Proper debug symbol validation (`CONFIG_DEBUG_KERNEL` / `CONFIG_DEBUG_INFO_*`)
  - Clean shutdown and resource management
  - Architecture-aware and user-friendly workflow

## Credits & Inspiration

- **Original Tutorial**: [Building Linux on macOS Natively](https://seiya.me/blog/building-linux-on-macos-natively) by Seiya Suzuki—fixed v6.17 issues (old make, sed, headers). Inspired our v6.18 extensions.
- **Upstream**: [Clang Built Linux](https://clangbuiltlinux.github.io/) for LLVM guidance; [LKML Kbuild thread](https://lore.kernel.org/lkml/20241002091726.12345-1-masahiroy@kernel.org/) for v6.18 details.
- **Author**: Phuc Nguyen (@NguyenTrongPhuc552003) — first native v6.18 RISC-V/ARM on macOS Tahoe.

## License

MIT — fork, extend, build freely.
