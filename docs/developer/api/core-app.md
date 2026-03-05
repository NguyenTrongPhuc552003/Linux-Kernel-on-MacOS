# Core App API

The app layer provides CLI wiring and dependency ownership.

---

## App Class

`src/app/app.hpp`

```cpp
class App {
public:
    App(std::unique_ptr<infra::executor::Executor> exec,
        std::unique_ptr<infra::filesystem::FileSystem> fs,
        config::Config cfg);

    auto build_cli() -> CLI::App&;
    auto run(int argc, char** argv) -> int;

    // Accessors (return references to owned objects)
    auto exec() -> infra::executor::Executor&;
    auto fs() -> infra::filesystem::FileSystem&;
    auto config() -> config::Config&;
    auto context() -> context::Context&;
    auto printer() -> ui::Printer&;
    auto kernel_builder() -> domain::builder::KernelBuilder&;
    auto module_builder() -> domain::builder::ModuleBuilder&;
    auto app_builder() -> domain::builder::AppBuilder&;
    auto qemu_runner() -> domain::emulator::QEMURunner&;
    auto health_checker() -> domain::doctor::HealthChecker&;
    auto rootfs_builder() -> domain::rootfs::Builder&;
    auto patcher() -> domain::patch::Patcher&;
    auto toolchain_manager() -> domain::toolchain::Manager&;
    auto hook_executor() -> plugin::HookExecutor&;
    auto plugin_registry() -> plugin::Registry&;
};
```

`App` owns all infrastructure via `std::unique_ptr` and passes raw pointers to domain services.

---

## Ownership Model

```cpp
// App owns infrastructure
std::unique_ptr<infra::executor::Executor> exec_;
std::unique_ptr<infra::filesystem::FileSystem> fs_;

// Domain services receive raw pointers (no ownership)
kernel_builder_ = std::make_unique<KernelBuilder>(context_.get(), toolchain_manager_.get());
```

---

## Command Registration

Commands are registered via free functions in `src/app/commands/`:

```cpp
// src/app/commands/commands.hpp
void register_kernel(App& app, CLI::App& cli);
void register_qemu(App& app, CLI::App& cli);
void register_all(App& app, CLI::App& cli);  // wires everything
```

Each command function receives `App&` for dependency access and `CLI::App&` for subcommand registration.

---

## Entry Point

```cpp
// src/main.cpp
int main(int argc, char** argv) {
    auto exec = std::make_unique<ShellExecutor>();
    auto fs = std::make_unique<OSFileSystem>();
    auto cfg = config::load(config_path);
    App app(std::move(exec), std::move(fs), std::move(cfg));
    return app.run(argc, argv);
}
```