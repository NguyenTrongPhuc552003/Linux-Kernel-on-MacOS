# Troubleshooting

Common issues and solutions for ELMOS.

## Build Issues

### CMake configure fails
- Ensure CMake 3.25+: `cmake --version`
- Install dependencies: `sudo apt install libcli11-dev libyaml-cpp-dev nlohmann-json3-dev libspdlog-dev libssl-dev catch2 libcpp-httplib-dev`
- Clean and retry: `rm -rf build/ && cmake --preset default`

### Toolchain Build Fails
- Run `./build/bin/elmos doctor` for missing deps
- Clean and retry: `./build/bin/elmos toolchains clean`
- Check crosstool-ng: `ct-ng version`

### Kernel Build Errors
- Regenerate config: `./build/bin/elmos kernel config defconfig`
- Apply patches: `./build/bin/elmos patch apply`
- Verify toolchain: `./build/bin/elmos toolchains status`

## Runtime Issues

### QEMU Won't Start
- Ensure kernel and rootfs exist
- Check architecture match: `./build/bin/elmos arch`
- Install QEMU: `sudo apt install qemu-system-arm qemu-system-misc`

### Module Load Fails
- Verify toolchain compatibility
- Check kernel symbols: `modinfo module.ko`

### TUI Shows Help Text
- Rebuild ELMOS: `cmake --build build --parallel`

## Environment

### Slow Builds
- Increase jobs in config YAML
- Use SSD for workspace

## General

### Doctor Fails
- Install all required packages (see Installation guide)
- Update your compiler: `sudo apt install g++-13`

### FetchContent Fails (inja/ftxui)
- Check network connectivity
- If HTTPS blocked, configure SSH: FetchContent uses `GIT_REPOSITORY`
- Try vcpkg preset: `cmake --preset vcpkg`

## Getting Help

- [GitHub Issues](https://github.com/NguyenTrongPhuc552003/elmos/issues)
- Check logs in `build/`
- Check logs in `build/`