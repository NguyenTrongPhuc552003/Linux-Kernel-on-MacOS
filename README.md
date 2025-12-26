# macOS Native Linux Kernel Builds

[![Build Status](https://img.shields.io/badge/build-v6.18%20RISC--V-green)](https://github.com/NguyenTrongPhuc552003/Linux-Kernel-on-MacOS) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Native builds for the Linux kernel (v6.18+) on macOS, targeting RISC-V, ARM64, and more. No Docker, no VMs—just Clang/LLVM, Homebrew, and targeted patches for host tool compatibility.

This project uses a **Hybrid Python/Bash Architecture** to orchestrate the build process, combining the robustness of Python for logic and state management with the raw power of Bash for low-level compilation.

## Why This Exists

Building the Linux kernel on macOS hits several walls:
- **Old tools**: macOS ships GNU Make 3.81 (kernel needs ≥4.0), BSD `sed`, and Clang without Linux headers.
- **Syscall mismatches**: v6.18 introduced `copy_file_range()` which breaks on macOS.
- **Case-sensitivity**: The Linux source tree requires a case-sensitive filesystem, while macOS is case-insensitive by default.

This project provides a CLI tool (`km`) that automates:
1.  **Disk Management**: Creates and mounts a Case-Sensitive APFS sparse image.
2.  **Toolchain Injection**: Wraps Homebrew tools (`gmake`, `llvm`) to look like native Linux tools.
3.  **Patch Management**: Automatically detects kernel versions and applies macOS-specific fixes.
4.  **Multi-Arch Support**: seamless switching between `riscv`, `arm64`, and other architectures via the Strategy Pattern.

---

## Quick Start

### 1. Prerequisites
You need **Python 3** and **Homebrew**. Install the required build tools:
```bash
brew install make llvm qemu e2fsprogs wget python3
```

### 2. Setup

Clone the repository:

```bash
git clone https://github.com/NguyenTrongPhuc552003/Linux-Kernel-on-MacOS.git kernel-dev
cd kernel-dev
```

(Optional) Setup a virtual environment for development tools:

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

### 3. Build & Boot Workflow

Use the `km` (Kernel Manager) CLI for all operations.

```bash
# 1. Mount the Case-Sensitive Workspace (Required)
./bin/km mount

# 2. Initialize Repo (Checkout a version, e.g., v6.12)
./bin/km repo branch v6.12

# 3. Apply macOS Compatibility Patches (Auto-detects version)
./bin/km patch apply auto

# 4. Generate Config & Build
./bin/km config defconfig
./bin/km build -j$(sysctl -n hw.ncpu)

# 5. Generate Root Filesystem (Debian Bullseye via debootstrap)
# Note: Requires sudo
./bin/km rootfs

# 6. Boot in QEMU
./bin/km qemu
```

---

## Usage Reference

The `km` tool replaces the old `run.sh`.

### Core Commands

| Command         | Arguments         | Description                                                           |
| --------------- | ----------------- | --------------------------------------------------------------------- |
| **`km build`**  | `-j<N> -a <arch>` | Compiles the kernel (Image, dtbs, modules). Auto-detects parallelism. |
| **`km config`** | `<target>`        | Runs `make <target>` (e.g., `defconfig`, `menuconfig`).               |
| **`km rootfs`** | `--arch <arch>`   | Generates a Debian rootfs using `debootstrap`.                        |
| **`km qemu`**   | `--debug --nogui` | Boots the kernel in QEMU. `-d` halts for GDB.                         |

### Utility Commands

| Command         | Arguments                   | Description                                          |
| --------------- | --------------------------- | ---------------------------------------------------- |
| **`km doctor`** |                             | Checks environment health (tools, headers, paths).   |
| **`km patch`**  | `list`, `apply`             | Manages macOS compatibility patches.                 |
| **`km repo`**   | `status`, `update`, `reset` | Manages the git repository state inside the mount.   |
| **`km module`** | `name -i -r`                | Manages external modules (queue for install/remove). |
| **`km mount`**  |                             | Mounts the APFS image to `/Volumes/kernel-dev`.      |

---

## Architecture

This project uses a **Hybrid Architecture**:

* **Python (`Python/`)**: Acts as the "Brain". It handles argument parsing (`argparse`), configuration management (Singleton), and Strategy selection (RISC-V vs ARM64).
* **Bash Services (`Python/Services/`)**: Acts as the "Muscle". These scripts (migrated from the old `scripts/` folder) perform the heavy execution (Make, Git, HDIUtil).
* **State (`var/state/`)**: Persistent storage for configuration (`build.cfg`) and runtime data.

```bash
.
├── bin/
│   └── km                # Symlink to Python/main.sh (Entry Point)
├── Python/
│   ├── commands/         # OOP Command Implementations (Build, Repo, etc.)
│   ├── managers/         # State Managers (ModuleState, Config)
│   ├── strategies/       # Arch Logic (RISC-V vs ARM64)
│   ├── Services/         # Low-level Bash Implementations
│   │   ├── KernelService.sh
│   │   ├── RootFSService.sh
│   │   └── ...
│   └── main.sh           # Bootstrapper
├── libraries/            # macOS Headers (elf.h, byteswap.h)
├── patches/              # Version-specific kernel patches
└── var/
    └── state/            # Runtime state (img.sparseimage, build.cfg)
```

## Credits & Inspiration

* **Original Tutorial**: [Building Linux on macOS Natively](https://seiya.me/blog/building-linux-on-macos-natively) by Seiya Suzuki.
* **Upstream**: [Clang Built Linux](https://clangbuiltlinux.github.io/) for LLVM guidance.
