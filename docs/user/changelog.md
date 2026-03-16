# Changelog

User-facing changes in ELMOS releases.

## v4.1.0 (Workspace Management)

- **Multi-workspace support**: `elmos init ws1 ws2` creates multiple workspaces at once
- **`elmos pick`**: Select active workspace — all commands target the picked workspace
- **Location-independent**: No need to `cd` into workspaces; control from anywhere
- **Auto-mount**: Volumes are transparently re-mounted after reboot
- **Persistent config**: Architecture and settings saved per-workspace in `~/.elmos/workspaces/`
- **`elmos arch`** with no args now shows current architecture (was: help)
- Config changes (`elmos arch riscv`) persist automatically to workspace config

## v4.0.0 (C++23 Rewrite)

- **Complete rewrite** from Go to C++23 (GCC 13+)
- CMake 3.25+ build system with presets (default, vcpkg, release)
- Cross-platform support: Linux, macOS, Windows
- CLI11 replaces Cobra for command parsing
- FTXUI replaces Bubble Tea for terminal UI
- Plugin system with builtin and external plugin support
- DAG-based build orchestration with fingerprinting
- BSP registry client for firmware blob management
- OpenSSL EVP API for SHA256 checksums
- Cooperative cancellation via `std::stop_token`
- `std::expected<T, Error>` for error handling (no exceptions)

## v6.18

- Support for Linux v6.18+ kernels
- Patches for `copy_file_range()` incompatibility
- ARM64, ARM, RISC-V toolchain configs
- Interactive TUI with real-time output
- Automatic toolchain detection
- Improved doctor checks

## Previous Versions

- v6.0 support with ARM/RISC-V patches
- Initial crosstool-ng integration
- Basic QEMU emulation
- Module and app templates

For full changelog, see [GitHub Releases](https://github.com/NguyenTrongPhuc552003/elmos/releases).