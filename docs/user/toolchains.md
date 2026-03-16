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

### Clone crosstool-ng

```bash
elmos toolchain clone
```

Clones and builds crosstool-ng into the workspace toolchains directory.

### List Targets

```bash
elmos toolchain list
```

Shows available configurations.

### Pick Target

```bash
elmos toolchain <target>
```

Example: `elmos toolchain riscv64-unknown-linux-gnu`

### Build Toolchain

```bash
elmos toolchain build
```

Builds the selected toolchain (~30–60 min). Installs to `<workspace>/toolchains/x-tools/`.

### Check Status

```bash
elmos toolchain status
```

Verifies installation and shows installed toolchains.

### Show Info

```bash
elmos toolchain show
```

Displays current toolchain architecture, bin dir, and installed status.

### Customize Config

```bash
elmos toolchain menuconfig
```

Interactive configuration for advanced users (requires crosstool-ng).

### Clean Artifacts

```bash
elmos toolchain clean
```

Removes build artifacts.

## Automatic Detection

Kernel, module, and app builds auto-detect installed toolchains based on the selected architecture (`./build/bin/elmos arch <arch>`).

## Custom Toolchains

For custom targets, modify configs in `assets/toolchains/configs/` and rebuild.

## Troubleshooting

- Build fails: Ensure all deps installed (`./build/bin/elmos doctor`)
- Slow builds: Use more cores with `CT_PARALLEL_JOBS` env var
- Conflicts: Clean and rebuild