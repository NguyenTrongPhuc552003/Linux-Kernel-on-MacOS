# ELMOS Documentation

<p align="center">
  <strong>Embedded Linux SDK</strong><br>
  A complete embedded Linux development toolkit — smarter, faster alternative to Buildroot.
</p>

---

## Features

| Feature                  | Description                                  |
| ------------------------ | -------------------------------------------- |
| 🔧 **Native Toolchains**  | Build cross-compilers for ARM64, ARM, RISC-V |
| 🐧 **Kernel Automation**  | Clone, configure, build Linux kernels        |
| 🖥️ **Interactive TUI**    | Rich terminal interface (FTXUI)              |
| 🚀 **QEMU Integration**   | Boot and debug with GDB                      |
| 📦 **Module Development** | Cross-compile kernel modules and apps        |
| 🔌 **Plugin System**      | 13 lifecycle hooks for extensibility         |

---

## Quick Start

```bash
# Build from source (C++23, CMake)
cmake --preset default
cmake --build build --parallel

# Initialize workspace
./build/bin/elmos init my_project
./build/bin/elmos doctor

# Build kernel
./build/bin/elmos kernel clone
./build/bin/elmos kernel config defconfig
./build/bin/elmos kernel build

# Run in QEMU
./build/bin/elmos qemu run
```

---

## Documentation

### [User Guide](user/index.md)

For users installing and using ELMOS:

- [Installation](user/installation.md) - Setup prerequisites
- [Getting Started](user/getting-started.md) - First kernel build
- [Kernel Building](user/kernel-building.md) - Build configurations
- [QEMU Integration](user/qemu-integration.md) - Running and debugging
- [Troubleshooting](user/troubleshooting.md) - Common issues

### [Developer Guide](developer/index.md)

For contributors:

- [Architecture](developer/architecture.md) - System design
- [Diagrams](developer/diagrams.md) - Visual architecture
- [Build System](developer/build-system.md) - Task automation
- [Code Patterns](developer/code-patterns.md) - Go idioms
- [Contributing](developer/contributing.md) - Guidelines

---

## CLI Overview

<!-- elmos_tui.png -->
![elmos_tui](images/elmos_tui.png)

---

## Support

- [GitHub Issues](https://github.com/NguyenTrongPhuc552003/elmos/issues) - Bug reports
- [Discussions](https://github.com/NguyenTrongPhuc552003/elmos/discussions) - Questions

---

*MIT Licensed. Inspired by [Seiya's tutorial](https://seiya.me/blog/building-linux-on-macos-natively).*
