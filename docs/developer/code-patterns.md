# Code Patterns

C++23 idioms and patterns used throughout ELMOS.

---

## Dependency Injection

ELMOS uses pointer-based constructor injection. The `App` class owns all services and passes raw pointers:

```cpp
// src/app/app.cpp
App::App() {
    exec_ = std::make_unique<infra::executor::ShellExecutor>();
    fs_ = std::make_unique<infra::filesystem::OSFileSystem>();
    platform_ = infra::platform::create_platform();
    config_ = std::make_unique<config::Config>();
    context_ = std::make_unique<context::Context>(config_.get(), exec_.get(), fs_.get());
    toolchain_manager_ = std::make_unique<toolchain::Manager>(exec_.get(), fs_.get(), config_.get());
    kernel_builder_ = std::make_unique<KernelBuilder>(context_.get(), toolchain_manager_.get());
    // ...
}
```

**Benefits:**

- Testable — inject mock implementations
- Explicit dependencies — no hidden coupling
- No global state — everything flows through constructors
- Clear ownership — `App` owns via `unique_ptr`, services borrow via raw pointer

---

## Abstract Base Classes

Domain defines abstract interfaces, infra implements:

```cpp
// Domain expects:
class Executor {
public:
    virtual ~Executor() = default;
    virtual auto run(std::stop_token token, const std::string& cmd,
                     const std::vector<std::string>& args) -> VoidResult = 0;
    virtual auto output(std::stop_token token, const std::string& cmd,
                        const std::vector<std::string>& args) -> Result<std::string> = 0;
};

// Infra provides:
class ShellExecutor : public Executor { /* real implementation */ };
class MockExecutor : public Executor  { /* test mock */ };
```

This allows unit testing domain logic without real shell commands.

---

## Error Handling

### Result Types (std::expected)

```cpp
// include/elmos/common.hpp
using VoidResult = std::expected<void, Error>;
template<typename T> using Result = std::expected<T, Error>;
```

### Error Construction

```cpp
// Returning errors
return std::unexpected(Error::generic("failed to build kernel"));
return std::unexpected(Error(ErrorCode::Dependency, "cmake not found"));

// Checking results
if (auto r = build(token, opts); !r) {
    printer.error("Build failed: {}", r.error().message());
    return;
}

// Accessing values
auto result = get_modules();
if (!result) { /* handle */ }
auto& modules = *result;
```

**Rule:** Never throw exceptions in domain/infra code. Use `Result<T>` / `VoidResult` consistently.

---

## Configuration Pattern

### YAML-backed Structs

```cpp
// src/config/types.hpp
struct Config {
    ImageConfig image;
    BuildConfig build;
    QEMUConfig qemu;
    PathsConfig paths;
};
```

### Computed Defaults

```cpp
// src/config/loader.cpp
void apply_defaults(Config& cfg) {
    if (cfg.paths.project_root.empty()) {
        cfg.paths.project_root = std::filesystem::current_path().string();
    }
    // Derive other paths from project_root...
}
```

---

## Embedded Assets

Templates embedded at compile-time via CMake's `EmbedResources.cmake`:

```cmake
# cmake/EmbedResources.cmake
elmos_embed_resources(elmos_resources
    SOURCES assets/templates/module/module.c.tmpl
            assets/templates/app/main.c.tmpl
)
```

Used for scaffolding new modules and apps.

---

## Command Registration

### Grouped Commands

```cpp
// src/app/commands/commands.hpp
void register_kernel(App& app, CLI::App& cli);
void register_qemu(App& app, CLI::App& cli);
void register_toolchain(App& app, CLI::App& cli);
// ...

void register_all(App& app, CLI::App& cli);
```

### Command Implementation

```cpp
// src/app/commands/kernel.cpp
void register_kernel(App& app, CLI::App& cli) {
    auto* kernel = cli.add_subcommand("kernel", "Kernel commands");

    auto* build_cmd = kernel->add_subcommand("build", "Build kernel");
    build_cmd->callback([&app] {
        std::stop_source ss;
        if (auto r = app.kernel_builder().build(ss.get_token(), {}); !r) {
            app.printer().error("Build failed: {}", r.error().message());
            return;
        }
        app.printer().success("Build complete!");
    });
}
```

---

## Printer Pattern

Styled output with std::format:

```cpp
// src/ui/printer.hpp
class Printer {
public:
    void step(std::format_string<Args...> fmt, Args&&... args);    // → prefix
    void success(std::format_string<Args...> fmt, Args&&... args); // ✓ prefix
    void error(std::format_string<Args...> fmt, Args&&... args);   // ✗ prefix
    void info(std::format_string<Args...> fmt, Args&&... args);    // ℹ prefix
    void warn(std::format_string<Args...> fmt, Args&&... args);    // ⚠ prefix
};
```

---

## Cooperative Cancellation

All long-running operations take `std::stop_token`:

```cpp
auto build(std::stop_token token, BuildOptions opts) -> VoidResult;

// At the command handler level:
std::stop_source ss;
auto result = builder.build(ss.get_token(), opts);
```

---

## Plugin Hooks

Lifecycle events with priority ordering:

```cpp
// src/plugin/interface.hpp
namespace events {
    constexpr auto kPreKernelBuild = "pre_kernel_build";
    constexpr auto kPostKernelBuild = "post_kernel_build";
    // ... 13 events total
}

// Hook priority: higher number runs first (range 0-10)
struct HookRegistration {
    std::string event;
    int priority;
    HookFn handler;
};
```

---

## Naming Conventions

| Type         | Convention     | Example                             |
| ------------ | -------------- | ----------------------------------- |
| Class        | PascalCase     | `KernelBuilder`, `QEMURunner`       |
| Method       | snake_case     | `build()`, `run()`, `configure()`   |
| Member var   | trailing `_`   | `ctx_`, `exec_`, `config_`          |
| Namespace    | snake_case     | `elmos::domain::builder`            |
| Header guard | `#pragma once` | (no traditional guards)             |
| Error        | `Error::*`     | `Error::generic("msg")`             |
| Result       | `Result<T>`    | `Result<std::string>`, `VoidResult` |