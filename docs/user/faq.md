# FAQ

Frequently asked questions about ELMOS.

## General

**What is ELMOS?**  
An embedded Linux SDK for building kernels, modules, and rootfs with cross-compilation support.

**What platforms are supported?**  
Linux (primary), macOS, and Windows (experimental). The host builds cross-compile for ARM64, ARM, and RISC-V targets.

**Supported architectures?**  
ARM64, ARM, RISC-V (Linux v6.18+).

## Installation

**Do I need Docker?**  
No, ELMOS builds natively using CMake and system packages.

**What compiler is needed?**  
GCC 13+ with C++23 support.

## Toolchains

**Are toolchains required?**  
Optional but recommended. Enables full cross-compilation via crosstool-ng.

**How long to build a toolchain?**  
30–60 min, depending on hardware.

## Kernel Building

**Which kernel versions?**  
v6.18+ recommended. Earlier versions require additional patches.

**Can I use custom configs?**  
Yes, via `elmos kernel config menuconfig`.

## Development

**How to develop modules/apps?**  
Use `./build/bin/elmos module create <name>` or `./build/bin/elmos app create <name>`, then build.

**Cross-compilation?**  
Automatic with detected toolchains from `elmos toolchains status`.

## QEMU

**Networking in QEMU?**  
User-mode; host at 10.0.2.2.

**Debugging?**  
Use `./build/bin/elmos qemu debug` with GDB.

## Contributing

**How to contribute?**  
See [Developer Guide](../developer/contributing.md).

**License?**  
MIT.