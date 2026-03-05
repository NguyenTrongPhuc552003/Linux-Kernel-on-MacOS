# User Guide Overview

This section provides comprehensive guides for using ELMOS to build and develop embedded Linux systems.

## Getting Started

If you're new to ELMOS:

1. [Install ELMOS](installation.md) - Prerequisites and setup
2. [First Kernel Build](getting-started.md) - Step-by-step tutorial
3. [Toolchain Management](toolchains.md) - Install and configure cross-compilers

## Core Workflows

- **Kernel Development**: [Configure and build kernels](kernel-building.md)
- **Module & App Creation**: [Develop kernel modules and userspace apps](modules-and-apps.md)
- **Emulation**: [Run and debug with QEMU](qemu-integration.md)
- **Interactive Mode**: [Use the TUI](tui-guide.md)

## Reference

- [Troubleshooting](troubleshooting.md) - Common issues and solutions
- [FAQ](faq.md) - Frequently asked questions
- [Changelog](changelog.md) - What's new

## Prerequisites

ELMOS supports Linux (primary), macOS, and Windows (experimental):

- CMake 3.25+ and GCC 13+ (or compatible C++23 compiler)
- System development packages (see [Installation](installation.md))
- Basic familiarity with terminal commands

For advanced features, install crosstool-ng toolchains.

## Support

Encounter an issue? Check [Troubleshooting](troubleshooting.md) or open a [GitHub Issue](https://github.com/NguyenTrongPhuc552003/elmos/issues).