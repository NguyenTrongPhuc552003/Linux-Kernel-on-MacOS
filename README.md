# ELMOS – Embedded Linux SDK

[![Build Status](https://img.shields.io/badge/build-v6.18%20ARM64-green)](https://github.com/NguyenTrongPhuc552003/elmos) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) [![Docs](https://img.shields.io/badge/docs-gh--pages-blue)](https://nguyentrongphuc552003.github.io/elmos/)

A complete embedded Linux SDK — a smarter, faster alternative to Buildroot. Build kernels, cross-compile with native toolchains, develop kernel modules and userspace apps. ELMOS provides an integrated development environment with interactive TUI, automatic toolchain management (crosstool-ng), and seamless QEMU integration. Targeting RISC-V, ARM64, ARM, and more.

Built with **C++23** (GCC 13+), CMake, and modern dependency injection patterns.

**[Read the Full Documentation](https://elmos.vercel.app/)**

## Features

- **Cross-Compiler Toolchain Management**: Build and manage crosstool-ng toolchains for ARM64, ARM, and RISC-V
- **Interactive TUI**: Rich terminal interface (FTXUI) with categorized command access
- **Environment Doctor**: Comprehensive dependency and toolchain health checks
- **Kernel Build Automation**: Configure, build, and test Linux kernels with integrated toolchain support
- **Module & App Development**: Build kernel modules and userspace apps with automatic cross-compilation
- **Plugin Architecture**: Extensible hook-based plugin system with 13 lifecycle events
- **QEMU Integration**: Boot and debug kernels with built-in GDB support
- **Board Support Packages**: Machine-specific configurations for popular SBCs

## Documentation

Comprehensive documentation is hosted on GitHub Pages:

- **[User Guide](https://elmos.vercel.app/user/)**: Usage, kernel building, toolchains.
- **[Developer Guide](https://elmos.vercel.app/developer/)**: Architecture, APIs, code patterns.
- **[Diagrams](https://elmos.vercel.app/developer/diagrams/)**: Component, Sequence, and Class diagrams.

## Quick Start

### 1. Prerequisites

**Ubuntu/Debian:**
```bash
sudo apt install build-essential cmake ninja-build pkg-config \
    libcli11-dev libyaml-cpp-dev nlohmann-json3-dev libspdlog-dev \
    libssl-dev catch2 libcpp-httplib-dev \
    git qemu-system debootstrap
```

**macOS (Homebrew):**
```bash
brew install cmake ninja pkg-config cli11 yaml-cpp nlohmann-json spdlog openssl catch2 cpp-httplib
brew install qemu git
```

### 2. Build

```bash
git clone https://github.com/NguyenTrongPhuc552003/elmos.git
cd elmos

# Configure + Build (system packages + FetchContent for inja/ftxui)
cmake --preset default
cmake --build build --parallel

# Or use Taskfile
task build
```

### 3. Initialize Workspace

```bash
./build/bin/elmos init my_project    # Create workspace structure
./build/bin/elmos doctor             # Verify environment
```

### 4. Install Toolchain

```bash
./build/bin/elmos toolchains install    # Install crosstool-ng
./build/bin/elmos toolchains list       # List available targets
./build/bin/elmos arch set arm64        # Set target architecture
./build/bin/elmos toolchains build      # Build the toolchain (~30-60 min)
./build/bin/elmos toolchains status     # Verify installation
```

### 5. Build Kernel & Run

```bash
./build/bin/elmos kernel clone              # Clone kernel source
./build/bin/elmos kernel config defconfig   # Or: menuconfig, tinyconfig
./build/bin/elmos kernel build              # Build with detected toolchain
./build/bin/elmos rootfs build              # Debian rootfs (debootstrap)
./build/bin/elmos qemu run                  # Boot in QEMU
./build/bin/elmos qemu debug                # With GDB stub (port 1234)
```

## Interactive TUI

Launch with `./build/bin/elmos tui` for a rich interactive interface with categorized menus for all operations.

## Command Reference

### Core Commands

| Command                 | Description                                        |
| ----------------------- | -------------------------------------------------- |
| `elmos init <name>`     | Initialize workspace directory structure           |
| `elmos doctor`          | Check environment health (tools, deps, toolchains) |
| `elmos status`          | Show workspace status                              |
| `elmos version`         | Display version and build info                     |
| `elmos tui`             | Launch interactive terminal UI                     |
| `elmos arch show`       | Show current target architecture                   |
| `elmos arch set <arch>` | Set target architecture (arm64/arm/riscv)          |

### Build Commands

| Command                      | Description                                        |
| ---------------------------- | -------------------------------------------------- |
| `elmos kernel clone [url]`   | Clone Linux kernel source                          |
| `elmos kernel config <type>` | Configure kernel (defconfig/menuconfig/tinyconfig) |
| `elmos kernel build [-j N]`  | Build the kernel                                   |
| `elmos kernel clean`         | Clean build artifacts                              |
| `elmos kernel status`        | Show kernel source status                          |
| `elmos kernel pull`          | Update kernel source                               |
| `elmos kernel switch <ref>`  | Switch branch/tag                                  |
| `elmos kernel reset`         | Reclone kernel from scratch                        |
| `elmos kernel install`       | Install kernel artifacts                           |
| `elmos rootfs build`         | Build Debian rootfs via debootstrap                |
| `elmos rootfs clean`         | Clean rootfs                                       |
| `elmos rootfs status`        | Show rootfs status                                 |
| `elmos module new <name>`    | Scaffold a new kernel module                       |
| `elmos module build [name]`  | Build kernel modules                               |
| `elmos module list`          | List available modules                             |
| `elmos module clean`         | Clean module build artifacts                       |
| `elmos app new <name>`       | Scaffold a new userspace application               |
| `elmos app build [name]`     | Build userspace apps                               |
| `elmos app list`             | List available apps                                |
| `elmos app clean`            | Clean app build artifacts                          |
| `elmos patch apply`          | Apply kernel patches from config                   |

### Toolchain Commands

| Command                            | Description                          |
| ---------------------------------- | ------------------------------------ |
| `elmos toolchains install`         | Install crosstool-ng                 |
| `elmos toolchains clone`           | Alias for install                    |
| `elmos toolchains list`            | List available target configurations |
| `elmos toolchains select <target>` | Select a toolchain target for builds |
| `elmos toolchains build`           | Build the selected toolchain         |
| `elmos toolchains status`          | Show installed toolchains            |
| `elmos toolchains env`             | Display environment variables        |
| `elmos toolchains menuconfig`      | Interactive toolchain configuration  |
| `elmos toolchains clean`           | Clean toolchain build artifacts      |

### Runtime & Config Commands

| Command                     | Description                            |
| --------------------------- | -------------------------------------- |
| `elmos qemu run`            | Boot kernel in QEMU                    |
| `elmos qemu debug`          | Boot with GDB server (port 1234)       |
| `elmos plugins list`        | List loaded plugins                    |
| `elmos bsp list`            | List available board support packages  |
| `elmos bsp fetch <machine>` | Fetch BSP configuration                |
| `elmos bootloader build`    | Build U-Boot (Phase 3)                 |
| `elmos bootloader install`  | Install bootloader artifacts (Phase 3) |

**Pre-configured toolchain targets:**
- `aarch64-unknown-linux-gnu` (ARM64)
- `arm-cortex_a15-linux-gnueabihf` (ARM 32-bit)
- `riscv64-unknown-linux-gnu` (RISC-V 64-bit)

## Build System

Uses [CMake](https://cmake.org/) with presets and optional [Task](https://taskfile.dev):

```bash
# CMake presets
cmake --preset default          # System packages + FetchContent
cmake --preset vcpkg            # vcpkg for all dependencies (requires VCPKG_ROOT)
cmake --preset release          # Optimized release build

# Taskfile commands
task --list                     # Show all targets
task build                      # Build elmos binary
task clean                      # Clean all artifacts
task test                       # Run tests
task release                    # Release build
task docs                       # Build documentation
task elmos:doctor               # Run environment check
task elmos:status               # Show workspace status
```

## Architecture

```
src/
├── main.cpp                    # Entry point only; no logic
├── app/                        # CLI wiring, CLI11 commands, App container
│   └── commands/               # Individual command registrations
├── config/                     # Config structs + YAML loader
├── context/                    # Shared runtime state
├── domain/                     # All business logic
│   ├── bsp/                    # Board support package registry
│   ├── builder/                # Kernel/module/app build orchestration
│   ├── doctor/                 # Environment health checks
│   ├── emulator/               # QEMU integration
│   ├── orchestrator/           # DAG pipeline + fingerprinter
│   ├── patch/                  # Patch application
│   ├── plugin/builtin/         # Compiled-in plugins
│   ├── rootfs/                 # Rootfs creation
│   └── toolchain/              # Cross-compilation toolchain management
├── infra/                      # I/O adapters (no domain logic)
│   ├── executor/               # Shell command execution
│   ├── filesystem/             # File I/O abstractions
│   ├── homebrew/               # Homebrew resolver (macOS)
│   └── platform/               # OS abstraction layer
├── plugin/                     # Plugin registry + hook executor
└── ui/                         # Printer, help formatter, TUI
include/elmos/                  # Public headers (common.hpp, forward.hpp)
cmake/                          # CMake modules (Platform, Version, EmbedResources)
```

### Design Principles

- **Pointer-based dependency injection** — domain services receive raw pointers, `App` owns everything via `std::unique_ptr`
- **Result types** — `Result<T>` / `VoidResult` via `std::expected`, no exceptions in domain/infra
- **No raw system calls** — all commands go through `infra::executor::Executor`
- **Platform abstraction** — OS-specific paths through platform layer, never hardcoded
- **Plugin hooks** — 13 lifecycle events with priority-ordered execution

## Dependencies

| Package       | Purpose              | Install                   |
| ------------- | -------------------- | ------------------------- |
| CLI11         | Command-line parsing | apt: `libcli11-dev`       |
| yaml-cpp      | YAML config loading  | apt: `libyaml-cpp-dev`    |
| nlohmann/json | JSON handling        | apt: `nlohmann-json3-dev` |
| spdlog        | Logging              | apt: `libspdlog-dev`      |
| OpenSSL       | SHA256 checksums     | apt: `libssl-dev`         |
| cpp-httplib   | HTTP client          | apt: `libcpp-httplib-dev` |
| Catch2        | Testing framework    | apt: `catch2`             |
| inja          | Template engine      | FetchContent (v3.4.0)     |
| FTXUI         | Terminal UI          | FetchContent (v5.0.0)     |

## Troubleshooting

| Issue                 | Solution                                       |
| --------------------- | ---------------------------------------------- |
| CMake < 3.25          | `pip3 install cmake` or build from source      |
| Missing packages      | Run `elmos doctor` to check deps               |
| Toolchain build fails | Check `elmos doctor` for missing deps          |
| vcpkg SSL errors      | Use `cmake --preset default` (system packages) |
| TUI not rendering     | Ensure terminal supports ANSI escape codes     |

## Credits

- **Author**: Phuc Nguyen ([@NguyenTrongPhuc552003](https://github.com/NguyenTrongPhuc552003))
- **Upstream**: [Clang Built Linux](https://clangbuiltlinux.github.io/) for LLVM guidance
- **Inspired by**: [Building Linux on macOS Natively](https://seiya.me/blog/building-linux-on-macos-natively) by Seiya Suzuki

## License

MIT — fork, extend, build freely.
