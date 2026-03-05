# Build System

ELMOS uses CMake 3.25+ for building and optional Task for developer workflow automation.

---

## CMake Build

### Presets

| Preset    | Purpose                                  | Command                  |
| --------- | ---------------------------------------- | ------------------------ |
| `default` | System packages + FetchContent           | `cmake --preset default` |
| `vcpkg`   | All deps via vcpkg (requires VCPKG_ROOT) | `cmake --preset vcpkg`   |
| `release` | Optimized release build                  | `cmake --preset release` |

### Build Commands

```bash
cmake --preset default              # Configure (first time)
cmake --build build --parallel      # Build
cmake --build build --target test   # Run tests
cmake --install build --prefix /usr/local  # Install
```

### Dependencies

Two strategies supported:
1. **System packages (apt)** + FetchContent for inja/ftxui
2. **vcpkg** for all packages (set `VCPKG_ROOT` env)

| Library       | Purpose          | apt Package          | FetchContent |
| ------------- | ---------------- | -------------------- | ------------ |
| CLI11         | CLI parsing      | `libcli11-dev`       | —            |
| yaml-cpp      | YAML config      | `libyaml-cpp-dev`    | —            |
| nlohmann/json | JSON handling    | `nlohmann-json3-dev` | —            |
| spdlog        | Logging          | `libspdlog-dev`      | —            |
| OpenSSL       | SHA256 checksums | `libssl-dev`         | —            |
| cpp-httplib   | HTTP client      | `libcpp-httplib-dev` | —            |
| Catch2        | Testing          | `catch2`             | —            |
| inja          | Template engine  | —                    | v3.4.0       |
| FTXUI         | Terminal UI      | —                    | v5.0.0       |

---

## Taskfile Overview

The `Taskfile.yml` provides developer workflow tasks:

| Task                | Purpose                   |
| ------------------- | ------------------------- |
| `task build`        | Configure + build binary  |
| `task clean`        | Remove build directory    |
| `task test`         | Run all tests             |
| `task release`      | Optimized release build   |
| `task docs`         | Build documentation site  |
| `task install`      | Install to /usr/local/bin |
| `task dev:check`    | Pre-commit style check    |
| `task dev:setup`    | Full development setup    |
| `task elmos:doctor` | Run `elmos doctor`        |

---

## Kernel Build System

### KernelBuilder

Located in `src/domain/builder/kernel.hpp`:

```cpp
class KernelBuilder {
public:
    KernelBuilder(context::Context* ctx, toolchain::Manager* tm);

    auto build(std::stop_token token, BuildOptions opts) -> VoidResult;
    auto configure(std::stop_token token, const std::string& type) -> VoidResult;
    auto clean(std::stop_token token) -> VoidResult;
};
```

### BuildOptions

```cpp
struct BuildOptions {
    int jobs = 0;                      // Parallel jobs (-j)
    std::vector<std::string> targets;  // e.g., {"Image", "dtbs", "modules"}
};
```

### Build Flow

```
KernelBuilder::build()
    ├── Get toolchain environment
    ├── Construct make arguments:
    │   - ARCH=arm64
    │   - CROSS_COMPILE=<prefix>
    │   - -j<jobs>
    └── exec_->run_with_env(token, env, "make", args)
```

---

## Module Build System

### ModuleBuilder

Located in `src/domain/builder/module.hpp`:

```cpp
class ModuleBuilder {
public:
    ModuleBuilder(context::Context* ctx, toolchain::Manager* tm);

    auto build(std::stop_token token, const std::string& name) -> VoidResult;
    auto clean(std::stop_token token, const std::string& name) -> VoidResult;
    auto create_module(const std::string& name) -> VoidResult;
    auto get_modules() -> Result<std::vector<ModuleInfo>>;
};
```

---

## App Build System

### AppBuilder

Located in `src/domain/builder/app.hpp`. Cross-compiles userspace applications for the target architecture.

---

## CMake Modules

### Platform.cmake

OS detection and platform-specific source selection:

```cmake
include(Platform)
elmos_platform_sources(target DARWIN darwin.cpp LINUX linux.cpp WINDOWS windows.cpp)
```

### Version.cmake

Git-based version injection via `add_compile_definitions`:

```cmake
add_compile_definitions(
    ELMOS_VERSION="${ELMOS_GIT_VERSION}"
    ELMOS_COMMIT="${ELMOS_GIT_COMMIT}"
    ELMOS_BUILD_DATE="${ELMOS_BUILD_DATE}"
)
```

### EmbedResources.cmake

Compile-time resource embedding (replaces Go's `//go:embed`).

---

## CLI Integration

### Build Command

```bash
elmos kernel build              # Default targets for arch
elmos kernel build -j 8         # Custom job count
```

### Configure Command

```bash
elmos kernel config defconfig   # Default config
elmos kernel config menuconfig  # Interactive
```

---

## Compiler Flags

Warning flags enabled globally:

```cmake
add_compile_options(
    -Wall -Wextra -Wpedantic
    -Wno-unused-parameter
    -Wno-missing-field-initializers
    -Wshadow -Wnon-virtual-dtor
)
```

Standard: C++23 with GCC 13+.