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

Or with Task:
```bash
task build
```

The binary is produced at `build/bin/elmos`.

## Initialize Workspace

Create a workspace directory structure:

```bash
./build/bin/elmos init my_project
```

This generates the workspace configuration and directory layout.

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
./build/bin/elmos arch set arm64        # Select architecture
./build/bin/elmos toolchains build      # Build toolchain (~30-60 min)
```

See [Toolchains](toolchains.md) for details.

## Next Steps

- [Get Started](getting-started.md) with your first kernel build
- Explore the [Interactive TUI](tui-guide.md)