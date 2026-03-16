# CLAUDE.md — ELMOS Codebase Codex

This file governs how AI assistants (and human contributors) should approach this codebase.
Read it before writing a single line of code.

---

## Project Identity

| Field      | Value                                                         |
| ---------- | ------------------------------------------------------------- |
| Language   | C++23 (GCC 13+)                                               |
| Build      | CMake 3.25+ with presets                                      |
| Binary     | `build/bin/elmos`                                             |
| Purpose    | Embedded Linux SDK — smarter, faster alternative to Buildroot |
| Repository | `github.com/NguyenTrongPhuc552003/elmos`                      |

---

## Quick-Start Commands

```bash
# Configure (first time or after CMakeLists changes)
cmake --preset default         # system packages + FetchContent
cmake --preset vcpkg           # OR use vcpkg (requires VCPKG_ROOT)

# Build
cmake --build build --parallel

# Test
cmake --build build --target test

# Clean
rm -rf build/

# Release build
cmake --preset release
cmake --build build-release --parallel
```

After **every** code change, rebuild and verify before committing:
```bash
cmake --build build --parallel && ./build/bin/elmos doctor
```

---

## Dependencies

### System packages (apt)
```bash
sudo apt install libcli11-dev libyaml-cpp-dev nlohmann-json3-dev \
    libspdlog-dev libssl-dev catch2 libcpp-httplib-dev \
    pkg-config ninja-build
```

### FetchContent (automatic via CMake)
- **inja** v3.4.0 — template engine
- **ftxui** v5.0.0 — terminal UI framework

### vcpkg alternative
Set `VCPKG_ROOT` and use `cmake --preset vcpkg` to use vcpkg for all dependencies.

---

## Architecture Layers

```
src/main.cpp                  ← binary entry point only; no logic here
src/app/                      ← CLI wiring, CLI11 commands, App container
src/config/                   ← config structs + loader; no business logic
src/context/                  ← shared runtime state passed through layers
src/domain/                   ← all business logic lives here
  bsp/                        ← BSP registry client + firmware blob manager
  builder/                    ← kernel/module/app build orchestration
  doctor/                     ← environment health checks
  emulator/                   ← QEMU integration
  orchestrator/               ← DAG pipeline + fingerprinter
  patch/                      ← patch application
  plugin/builtin/             ← compiled-in plugins
  rootfs/                     ← rootfs creation
  toolchain/                  ← cross-compilation toolchain management
src/infra/                    ← I/O adapters (no domain logic)
  executor/                   ← shell command execution interface
  filesystem/                 ← file I/O abstractions
  homebrew/                   ← Homebrew resolver (macOS only)
  platform/                   ← OS abstraction (Linux/macOS/Windows impls)
src/plugin/                   ← plugin registry + hook executor + factory map
src/ui/                       ← printer, help formatter, TUI components
include/elmos/                ← public headers (common.hpp, forward.hpp)
cmake/                        ← CMake modules (Platform.cmake, Version.cmake)
```

### Include Rules (strictly enforced)

- `domain/` headers **must not** include `app/` or `app/commands/`
- `infra/` headers **must not** include `domain/`
- `plugin/builtin/` **must not** include `plugin/` directly — only `plugin` interfaces
- `app/commands/` may include any layer below it
- Circular includes are always wrong; use forward declarations and pointer injection
- The `plugin` ↔ `domain/plugin/builtin` cycle is broken via:
  - `plugin/interface.hpp` exposes `register_builtin_factory(name, factory)`
  - `domain/plugin/builtin/` calls `register_builtin_factory` at static init time
  - `app/app.cpp` includes `<domain/plugin/builtin.hpp>` to trigger registration

---

## Dependency Injection Pattern

All services use **pointer-based constructor injection**. Domain services receive raw pointers
to infrastructure they don't own:

```cpp
// Correct — pointer injection, no ownership transfer
class KernelBuilder {
public:
    KernelBuilder(context::Context* ctx, toolchain::Manager* tm);
private:
    context::Context* ctx_;
    toolchain::Manager* tm_;
};

// Wrong — taking ownership or references
KernelBuilder(std::unique_ptr<Context> ctx);  // takes ownership
KernelBuilder(Context& ctx);                   // can't reassign
```

The `App` class owns all infrastructure via `std::unique_ptr` and passes raw pointers
to domain services:

```cpp
kernel_builder_ = std::make_unique<KernelBuilder>(context_.get(), toolchain_manager_.get());
```

---

## Adding a New Builtin Plugin (3-Step Recipe)

### Step 1 — Create the plugin file

`src/domain/plugin/builtin/my_plugin.hpp`:

```cpp
#pragma once
#include <plugin/interface.hpp>

namespace elmos::domain::plugin::builtin {

class MyPlugin final : public ::elmos::plugin::Plugin {
public:
    auto name() const -> std::string override { return "my-plugin"; }
    auto version() const -> std::string override { return "1.0.0"; }
    auto description() const -> std::string override { return "one-line description"; }

    auto init(context::Context* ctx, const AnyMap& config) -> VoidResult override;
    auto validate() -> VoidResult override;
    auto cleanup() -> VoidResult override;
    auto hooks() -> std::vector<::elmos::plugin::HookRegistration> override;
};

} // namespace elmos::domain::plugin::builtin
```

### Step 2 — Register the factory

In `src/domain/plugin/builtin/builtin.cpp`, add:

```cpp
void register_all() {
    // existing registrations…
    plugin::register_builtin_factory("my-plugin", [] {
        return std::make_unique<MyPlugin>();
    });
}
```

### Step 3 — Verify

```bash
cmake --build build --parallel && ./build/bin/elmos plugins list
```

---

## Adding a New CLI Command (2-Step Recipe)

### Step 1 — Create the command file

`src/app/commands/mycommand.cpp`:

```cpp
#include "commands.hpp"
#include <app/app.hpp>

namespace elmos::app::commands {

void register_mycommand(App& app, CLI::App& cli) {
    auto* cmd = cli.add_subcommand("mycommand", "One-line description");
    cmd->callback([&app] {
        // use app.config(), app.printer(), app.exec(), etc.
    });
}

} // namespace elmos::app::commands
```

### Step 2 — Wire into registry

In `src/app/commands/commands.hpp`, add the declaration:
```cpp
void register_mycommand(App& app, CLI::App& cli);
```

In the `register_all` function, add the call:
```cpp
register_mycommand(app, cli);
```

---

## Result Types and Error Handling

Use the type aliases defined in `include/elmos/common.hpp`:

```cpp
using VoidResult = std::expected<void, Error>;
using Result<T>  = std::expected<T, Error>;

// Returning errors
return std::unexpected(Error::generic("failed to build kernel"));

// Checking results
if (auto r = build(); !r) {
    printer.error("Build failed: {}", r.error().message());
    return;
}

// Accessing values
auto result = get_modules();
if (!result) { /* handle error */ }
auto& modules = *result;
```

Never throw exceptions in domain/infra code. Use `Result<T>` / `VoidResult` consistently.

---

## Shell Command Execution

All shell commands **must** go through `infra::executor::Executor`. Never call `system()`,
`popen()`, or `exec*()` directly in domain or plugin code.

```cpp
// Correct — testable, mockable
auto result = exec_->run(token, "make", {"-C", dir, "-j8"});

// Wrong — untestable, bypasses abstraction
system("make -C dir -j8");
```

The `Executor` interface is in `src/infra/executor/interface.hpp`. Inject via pointer constructor.

---

## Platform Path Rules

**Never** hardcode OS-specific paths. Every path that differs between operating systems
must go through the platform layer (`src/infra/platform/`):

| Don't do this                      | Do this instead                          |
| ---------------------------------- | ---------------------------------------- |
| `/Volumes/<name>`                  | `platform->paths().workspace_root(name)` |
| `/opt/homebrew/bin/gcc`            | `platform->packages().get_bin_path(pkg)` |
| `exec->run(token, "brew", ...)`    | `platform->packages().install(pkg)`      |
| `exec->run(token, "hdiutil", ...)` | `platform->disk_image().create(path, n)` |
| `/usr/local/` hardcoded            | `platform->paths().toolchain_dir()`      |

---

## Cache Directory Convention

All cache/state stored under `~/.elmos/`. Never invent a new top-level path.

```cpp
// Correct — derive from HOME
auto home = std::string(std::getenv("HOME"));
auto cache_dir = home + "/.elmos/bsp-cache";

// Wrong
auto cache_dir = "/tmp/elmos-cache";
```

---

## Plugin Hook Events

Defined in `src/plugin/interface.hpp`. Current supported events:

| Constant                       | When Fired                            |
| ------------------------------ | ------------------------------------- |
| `events::kPreKernelBuild`      | Before kernel compilation starts      |
| `events::kPostKernelBuild`     | After kernel compilation completes    |
| `events::kPreBootloaderBuild`  | Before bootloader build starts        |
| `events::kPostBootloaderBuild` | After bootloader build completes      |
| `events::kAfterConfigLoad`     | After workspace config YAML is parsed |
| `events::kPreRootfsCreate`     | Before rootfs population              |
| `events::kPostRootfsCreate`    | After rootfs population               |
| `events::kPreImageAssemble`    | Before final image assembly           |
| `events::kPostImageAssemble`   | After final image assembly            |
| `events::kPreQEMUBoot`         | Before QEMU emulator starts           |
| `events::kPostQEMUBoot`        | After QEMU emulator starts            |
| `events::kOnBuildError`        | On any build error                    |
| `events::kOnCleanup`           | During resource cleanup               |

Hook priority: **higher number runs first** (range 0–10). Use 8–10 for validation hooks.

---

## Async and Cancellation

All long-running operations accept `std::stop_token` as their first parameter for
cooperative cancellation:

```cpp
auto build(std::stop_token token, BuildOptions opts) -> VoidResult;
```

Create a `std::stop_source` at the command handler level. Pass `ss.get_token()` to domain services.

---

## Code Review Checklist

Before submitting any PR, verify:

- [ ] `cmake --build build --parallel` produces zero errors
- [ ] `./build/bin/elmos doctor` runs without crashes
- [ ] No new `system()` or `popen()` calls in `src/domain/` or `src/plugin/`
- [ ] No hardcoded OS-specific paths; use platform layer
- [ ] All errors returned via `Result<T>` / `VoidResult`, not exceptions
- [ ] No new global mutable state except builtin factory registration
- [ ] All config reaches components via constructor injection
- [ ] New builtin plugin registered in `builtin/` and factory map
- [ ] New CLI command declared in `commands.hpp` and wired in `register_all`

---

## Codebase Statistics

| Metric           | Value                            |
| ---------------- | -------------------------------- |
| Source files     | 93                               |
| Total LOC        | ~7,900                           |
| CMakeLists files | 23                               |
| CMake modules    | 3                                |
| Binary size      | 34 MB (debug)                    |
| Build time       | ~30s (parallel, incremental ~2s) |
