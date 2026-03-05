# Architecture

ELMOS follows a layered, domain-driven architecture in C++23 with strict separation of concerns and pointer-based dependency injection.

## Visual Overview

The following diagrams provide comprehensive views of ELMOS architecture:

### Layered Architecture Diagram

```plantuml
--8<-- "docs/diagrams/architecture.puml"
```

This diagram shows all layers and their relationships:
- **UI Layer:** CLI, TUI, and output formatting
- **Application Layer:** Command routing and dependency container
- **Configuration Layer:** Loading and validation
- **Context Layer:** Build state management
- **Domain Layer:** Business logic (5 subdomains)
- **Infrastructure Layer:** System abstractions
- **Cross-Cutting Concerns:** Results, logging, versioning

### System Context Diagram

```plantuml
--8<-- "docs/diagrams/system-context.puml"
```

Shows ELMOS within the broader ecosystem:
- Developer and CI/CD actors
- External systems (toolchain, registries, repositories)
- Output artifacts and deployment targets

### Build Pipeline State Machine

```plantuml
--8<-- "docs/diagrams/build-pipeline.puml"
```

Visualizes the complete build workflow:
- Initialization phase (validation → config → context)
- DAG planning (dependency analysis → optimization)
- Parallel execution (kernel, modules, rootfs)
- Assembly and validation
- Cache optimization paths

### Data Flow Diagram

```plantuml
--8<-- "docs/diagrams/dataflow.puml"
```

Shows how data transforms through the pipeline:
- Configuration loading
- Fingerprinting for incremental builds
- Task execution stages
- Artifact production

### Plugin Architecture

```plantuml
--8<-- "docs/diagrams/plugin-architecture.puml"
```

Demonstrates the hook-based extension system:
- Plugin interface contract
- 13 hook events at critical build stages
- Priority-based hook execution
- Integration with domain services

### Domain Class Model

```plantuml
--8<-- "docs/diagrams/domain-classes-simple.puml"
```

Shows core domain classes and relationships:
- Configuration hierarchy
- Build context and services
- Design patterns used
- Dependency injection

### User Command Execution Sequence

```plantuml
--8<-- "docs/diagrams/sequence-build.puml"
```

Detailed interaction sequence when user runs a command:
- CLI parsing and app initialization
- Configuration loading
- Context setup
- Build orchestration
- Result reporting

### Deployment & Infrastructure

```plantuml
--8<-- "docs/diagrams/deployment-infrastructure-simple.puml"
```

Infrastructure considerations for:
- Cross-platform support (macOS, Linux, Windows/WSL2)
- Toolchain caching
- Build execution environments
- Deployment targets

---

## Layer Overview

| Layer       | Directory      | Purpose                            |
| ----------- | -------------- | ---------------------------------- |
| **App**     | `src/app/`     | CLI commands, dependency injection |
| **Domain**  | `src/domain/`  | Business logic, platform-agnostic  |
| **Infra**   | `src/infra/`   | System interactions, external deps |
| **Config**  | `src/config/`  | Configuration types and loading    |
| **Context** | `src/context/` | Build state and environment        |
| **Plugin**  | `src/plugin/`  | Plugin registry and hook execution |
| **UI**      | `src/ui/`      | TUI and output formatting          |

---

## App Layer (`src/app/`)

The application entry point and command registration.

### App Class

```cpp
// src/app/app.hpp
class App {
public:
    App();
    auto run(int argc, char** argv) -> int;

    // Service accessors (raw pointers, App retains ownership)
    auto config() -> config::Config&;
    auto context() -> context::Context&;
    auto printer() -> ui::Printer&;
    auto exec() -> infra::executor::Executor&;
    auto kernel_builder() -> domain::builder::KernelBuilder&;
    auto module_builder() -> domain::builder::ModuleBuilder&;
    auto app_builder() -> domain::builder::AppBuilder&;
    auto qemu_runner() -> domain::emulator::QEMURunner&;
    auto health_checker() -> domain::doctor::HealthChecker&;
    auto rootfs_builder() -> domain::rootfs::RootfsBuilder&;
    auto toolchain_manager() -> domain::toolchain::Manager&;
    auto plugin_registry() -> plugin::Registry&;

private:
    CLI::App cli_;
    std::unique_ptr<config::Config> config_;
    std::unique_ptr<infra::executor::Executor> exec_;
    std::unique_ptr<infra::filesystem::FileSystem> fs_;
    std::unique_ptr<infra::platform::Platform> platform_;
    std::unique_ptr<context::Context> context_;
    // ... all domain services owned via unique_ptr
};
```

**Key Design:** `App` owns all infrastructure and domain services via `std::unique_ptr` and passes raw pointers to domain services for dependency injection.

### Command Registration

Commands are registered as free functions in `src/app/commands/`:

```cpp
// src/app/commands/commands.hpp
void register_kernel(App& app, CLI::App& cli);
void register_qemu(App& app, CLI::App& cli);
void register_toolchain(App& app, CLI::App& cli);
// ... one per command group

void register_all(App& app, CLI::App& cli);
```

---

## Domain Layer (`src/domain/`)

Platform-independent business logic.

### Modules

| Module          | Purpose                    | Key Types                                      |
| --------------- | -------------------------- | ---------------------------------------------- |
| `builder/`      | Kernel, module, app builds | `KernelBuilder`, `ModuleBuilder`, `AppBuilder` |
| `doctor/`       | Environment health checks  | `HealthChecker`                                |
| `emulator/`     | QEMU execution             | `QEMURunner`, `RunOptions`                     |
| `patch/`        | Kernel patch management    | `Patcher`                                      |
| `rootfs/`       | Root filesystem creation   | `RootfsBuilder`                                |
| `toolchain/`    | Cross-compiler management  | `Manager`, `ToolchainInfo`                     |
| `bsp/`          | Board support packages     | `FirmwareManager`, `BlobSpec`                  |
| `orchestrator/` | DAG pipeline               | `Pipeline`, `Fingerprinter`                    |
| `plugin/`       | Builtin plugins            | `BuiltinPlugin` implementations                |

### Example: KernelBuilder

```cpp
// src/domain/builder/kernel.hpp
class KernelBuilder {
public:
    KernelBuilder(context::Context* ctx, toolchain::Manager* tm);

    auto build(std::stop_token token, BuildOptions opts) -> VoidResult;
    auto configure(std::stop_token token, const std::string& type) -> VoidResult;
    auto clean(std::stop_token token) -> VoidResult;

private:
    context::Context* ctx_;
    toolchain::Manager* tm_;
};
```

---

## Infra Layer (`src/infra/`)

External system interactions with abstract base classes for testability.

### Interfaces

| Package       | Abstract Class | Purpose                          |
| ------------- | -------------- | -------------------------------- |
| `executor/`   | `Executor`     | Run shell commands               |
| `filesystem/` | `FileSystem`   | File I/O operations              |
| `platform/`   | `Platform`     | OS-specific path abstraction     |
| `homebrew/`   | `Resolver`     | Homebrew path resolution (macOS) |

### Executor Interface

```cpp
// src/infra/executor/interface.hpp
class Executor {
public:
    virtual ~Executor() = default;

    virtual auto run(std::stop_token token, const std::string& cmd,
                     const std::vector<std::string>& args) -> VoidResult = 0;
    virtual auto run_with_env(std::stop_token token, const EnvList& env,
                              const std::string& cmd,
                              const std::vector<std::string>& args) -> VoidResult = 0;
    virtual auto output(std::stop_token token, const std::string& cmd,
                        const std::vector<std::string>& args) -> Result<std::string> = 0;
    virtual auto look_path(const std::string& cmd) -> Result<std::string> = 0;
    // ...
};
```

---

## Result Types and Error Handling

All errors are returned via `std::expected`, never exceptions:

```cpp
// include/elmos/common.hpp
using VoidResult = std::expected<void, Error>;
template<typename T> using Result = std::expected<T, Error>;

// Usage
auto result = kernel_builder.build(token, opts);
if (!result) {
    printer.error("Build failed: {}", result.error().message());
    return;
}
```

---

## Data Flow

```
User → CLI (CLI11) → Command Handler
                          ↓
              Domain Service (e.g., KernelBuilder)
                          ↓
              Infra Interface (e.g., Executor::run())
                          ↓
                    External System (make, qemu, etc.)
```

1. User runs `elmos kernel build`
2. CLI11 parses args, calls command handler lambda
3. Handler uses `app.kernel_builder().build(token, opts)`
4. KernelBuilder calls `exec_->run_with_env(token, env, "make", args)`
5. Output streamed back to user

---

## Key Patterns

### Pointer-Based Dependency Injection

All domain services receive dependencies via constructor pointers:

```cpp
// src/app/app.cpp — wiring
exec_ = std::make_unique<infra::executor::ShellExecutor>();
fs_ = std::make_unique<infra::filesystem::OSFileSystem>();
context_ = std::make_unique<context::Context>(config_.get(), exec_.get(), fs_.get());
kernel_builder_ = std::make_unique<KernelBuilder>(context_.get(), toolchain_manager_.get());
```

### Abstract Base Classes

Domain defines abstract interfaces, infra implements:

```cpp
// Domain uses:
class Executor { virtual auto run(...) -> VoidResult = 0; };

// Infra provides:
class ShellExecutor : public Executor { /* real implementation */ };
class MockExecutor : public Executor  { /* for testing */ };
```

### Cooperative Cancellation

Long-running operations accept `std::stop_token`:

```cpp
auto build(std::stop_token token, BuildOptions opts) -> VoidResult;
// Command handler creates std::stop_source, passes token
```

---

## Directory Structure

```
src/
├── main.cpp                    # Entry point only; no logic
├── app/
│   ├── app.hpp / app.cpp       # App class, owns all services
│   └── commands/               # CLI command handlers
│       ├── commands.hpp        # Registration declarations
│       ├── kernel.cpp          # elmos kernel *
│       ├── qemu.cpp            # elmos qemu *
│       ├── toolchain.cpp       # elmos toolchains *
│       └── ...
├── config/
│   ├── arch.hpp                # Architecture configs (arm64, riscv)
│   ├── defaults.hpp            # Default values and required packages
│   ├── loader.hpp              # YAML config loading
│   └── types.hpp               # Config struct definitions
├── context/
│   └── context.hpp             # Build context, path resolution
├── domain/
│   ├── builder/                # Kernel/module/app builders
│   ├── bsp/                    # Board support package registry
│   ├── doctor/                 # Health checks
│   ├── emulator/               # QEMU runner
│   ├── orchestrator/           # DAG pipeline + fingerprinting
│   ├── patch/                  # Patch management
│   ├── plugin/builtin/         # Compiled-in plugins
│   ├── rootfs/                 # RootFS creation
│   └── toolchain/              # Toolchain management
├── infra/
│   ├── executor/               # Command execution (shell + mock)
│   ├── filesystem/             # File operations
│   ├── homebrew/               # Homebrew paths (macOS)
│   └── platform/               # OS abstraction layer
├── plugin/
│   ├── interface.hpp           # Plugin base class + hook events
│   ├── registry.hpp            # Plugin registry
│   └── executor.hpp            # Hook executor
└── ui/
    ├── printer.hpp             # Styled output
    ├── help.hpp                # Custom CLI11 help formatter
    └── tui/                    # Interactive TUI (FTXUI)
include/elmos/
    ├── common.hpp              # Result types, Error class
    └── forward.hpp             # Forward declarations
cmake/
    ├── Platform.cmake          # OS detection + source selection
    ├── Version.cmake           # Git-based version injection
    └── EmbedResources.cmake    # Compile-time resource embedding
```