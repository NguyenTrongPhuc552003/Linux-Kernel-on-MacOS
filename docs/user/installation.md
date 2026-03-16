# Installation

This guide covers installing ELMOS and its prerequisites.

## Prerequisites

### Ubuntu/Debian (recommended)

```bash
sudo apt install build-essential cmake ninja-build pkg-config \
    libcli11-dev libyaml-cpp-dev nlohmann-json3-dev libspdlog-dev \
    libssl-dev catch2 libcpp-httplib-dev \
    git qemu-system debootstrap
```

!!! note
    CMake 3.25+ is required. If your distro ships an older version: `pip3 install cmake`

### macOS (Homebrew)

```bash
brew install cmake ninja pkg-config yaml-cpp nlohmann-json spdlog openssl cpp-httplib git qemu
```

!!! note
    Some packages (CLI11, Catch2) may need to be installed via vcpkg or FetchContent on macOS.

### Fedora/RHEL

```bash
sudo dnf install cmake ninja-build pkgconf-pkg-config gcc-c++ \
    cli11-devel yaml-cpp-devel json-devel spdlog-devel openssl-devel \
    catch2-devel cpp-httplib-devel git qemu
```

## Build ELMOS

Clone the repository and build the binary:

```bash
git clone https://github.com/NguyenTrongPhuc552003/elmos.git
cd elmos
cmake --preset default
cmake --build build --parallel
```

The binary is produced at `build/bin/elmos`.

## Initialize Workspace

Create a disk image, mount a dedicated volume, and scaffold the workspace:

```bash
./build/bin/elmos init my_project
```

This creates a case-sensitive volume (`.sparseimage` on macOS, ext4 sparse file on Linux)
and registers it as the active workspace. No `cd` required — all commands target the active workspace.

To manage multiple workspaces:

```bash
./build/bin/elmos init project_a project_b   # Create multiple at once
./build/bin/elmos pick project_b             # Switch active workspace
./build/bin/elmos pick                       # List all workspaces
```

## Verify Setup

Run the environment doctor to check dependencies:

```bash
./build/bin/elmos doctor
```

If issues arise, see [Troubleshooting](../user/troubleshooting.md).

## Optional: Install Toolchains

For full cross-compilation, install crosstool-ng toolchains:

```bash
./build/bin/elmos toolchains install    # Install crosstool-ng
./build/bin/elmos toolchains list       # List targets
./build/bin/elmos arch arm64            # Select architecture
./build/bin/elmos toolchains build      # Build toolchain (~30-60 min)
```

See [Toolchains](toolchains.md) for details.

## Next Steps

- [Get Started](getting-started.md) with your first kernel build
- Explore the [Interactive TUI](tui-guide.md)