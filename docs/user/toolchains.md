# Toolchain Management

ELMOS integrates [crosstool-ng](https://crosstool-ng.github.io/) for building native cross-compilers.

## Overview

Toolchains enable cross-compilation for target architectures. ELMOS supports pre-configured targets with optimized settings.

## Supported Targets

| Target                           | Architecture | Description             |
| -------------------------------- | ------------ | ----------------------- |
| `aarch64-unknown-linux-gnu`      | ARM64        | 64-bit ARM              |
| `arm-cortex_a15-linux-gnueabihf` | ARM          | 32-bit ARM (Cortex-A15) |
| `riscv64-unknown-linux-gnu`      | RISC-V       | 64-bit RISC-V           |

## Commands

### Install crosstool-ng

```bash
./build/bin/elmos toolchains install
```

Clones and builds crosstool-ng to `~/.elmos/toolchains/crosstool-ng/`.

### List Targets

```bash
./build/bin/elmos toolchains list
```

Shows available configurations.

### Select Target

```bash
./build/bin/elmos toolchains select <target>
```

Example: `./build/bin/elmos toolchains select riscv64-unknown-linux-gnu`

### Build Toolchain

```bash
./build/bin/elmos toolchains build
```

Builds the selected toolchain (~30–60 min). Installs to `~/.elmos/toolchains/x-tools/`.

### Check Status

```bash
./build/bin/elmos toolchains status
```

Verifies installation and shows installed toolchains.

### Show Environment

```bash
./build/bin/elmos toolchains env
```

Displays `CROSS_COMPILE`, `PATH`, etc.

### Customize Config

```bash
./build/bin/elmos toolchains menuconfig
```

Interactive configuration for advanced users (requires crosstool-ng).

### Clean Artifacts

```bash
./build/bin/elmos toolchains clean
```

Removes build artifacts.

## Automatic Detection

Kernel, module, and app builds auto-detect installed toolchains based on the selected architecture (`./build/bin/elmos arch set <arch>`).

## Custom Toolchains

For custom targets, modify configs in `assets/toolchains/configs/` and rebuild.

## Troubleshooting

- Build fails: Ensure all deps installed (`./build/bin/elmos doctor`)
- Slow builds: Use more cores with `CT_PARALLEL_JOBS` env var
- Conflicts: Clean and rebuild