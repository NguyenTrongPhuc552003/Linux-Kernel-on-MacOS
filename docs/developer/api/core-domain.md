# Core Domain API

Business logic classes in the domain layer. All accept raw pointers via constructor injection.

---

## Builder Package

### KernelBuilder

`src/domain/builder/kernel.hpp`

```cpp
struct BuildOptions {
    int jobs = 0;
    std::vector<std::string> targets;
};

class KernelBuilder {
public:
    KernelBuilder(context::Context* ctx, toolchain::Manager* tm);

    auto build(std::stop_token token, BuildOptions opts) -> VoidResult;
    auto configure(std::stop_token token, const std::string& config_type) -> VoidResult;
    auto clean(std::stop_token token) -> VoidResult;
    auto enable_kvm_config(std::stop_token token) -> VoidResult;
    auto get_default_targets() -> std::vector<std::string>;
    auto has_config() -> bool;
    auto has_kernel_image() -> bool;
};
```

### ModuleBuilder

`src/domain/builder/module.hpp`

```cpp
struct ModuleInfo {
    std::string name, path, description;
    bool built = false;
};

class ModuleBuilder {
public:
    ModuleBuilder(context::Context* ctx, toolchain::Manager* tm);

    auto build(std::stop_token token, const std::string& name = "") -> VoidResult;
    auto clean(std::stop_token token, const std::string& name = "") -> VoidResult;
    auto get_modules(const std::string& name = "") -> Result<std::vector<ModuleInfo>>;
    auto prepare_headers(std::stop_token token) -> VoidResult;
    auto create_module(const std::string& name) -> VoidResult;
};
```

### AppBuilder

`src/domain/builder/app.hpp`

```cpp
struct AppInfo {
    std::string name, path;
    bool built = false;
};

class AppBuilder {
public:
    AppBuilder(context::Context* ctx, toolchain::Manager* tm);

    auto build(std::stop_token token, const std::string& name = "") -> VoidResult;
    auto clean(std::stop_token token, const std::string& name = "") -> VoidResult;
    auto get_apps(const std::string& name = "") -> Result<std::vector<AppInfo>>;
    auto create_app(const std::string& name) -> VoidResult;
};
```

---

## Doctor Package

### HealthChecker

`src/domain/doctor/checker.hpp`

```cpp
struct CheckResult {
    std::string name;
    bool passed = false;
    bool required = true;
    std::string message;
};

class HealthChecker {
public:
    HealthChecker(infra::executor::Executor* exec,
                  infra::filesystem::FileSystem* fs,
                  config::Config* cfg,
                  infra::platform::Platform* platform,
                  toolchain::Manager* tm);

    auto check_all(std::stop_token token) -> std::pair<std::vector<CheckResult>, int>;
    auto check_packages(std::stop_token token) -> std::vector<CheckResult>;
    auto check_headers() -> std::vector<CheckResult>;
    auto check_cross_gdb(std::stop_token token) -> std::vector<CheckResult>;
    auto check_toolchains() -> std::vector<CheckResult>;
};
```

---

## Emulator Package

### QEMURunner

`src/domain/emulator/qemu.hpp`

```cpp
struct RunOptions {
    bool gdb = false;
    bool graphic = false;
    std::string initrd, append;
    std::vector<std::string> extra_args;
};

class QEMURunner {
public:
    explicit QEMURunner(context::Context* ctx);

    auto run(std::stop_token token, RunOptions opts) -> VoidResult;
    auto build_command(const RunOptions& opts) -> Result<std::pair<std::string, std::vector<std::string>>>;
    auto is_available(std::stop_token token) -> bool;
};
```

---

## Patch Package

### Patcher

`src/domain/patch/patcher.hpp`

```cpp
struct PatchInfo {
    std::string name, path;
    bool applied = false;
};

class Patcher {
public:
    Patcher(infra::executor::Executor* exec, infra::filesystem::FileSystem* fs);

    auto apply(std::stop_token token, const std::string& target_dir,
               const std::string& patch_dir) -> VoidResult;
    auto apply_single(std::stop_token token, const std::string& target_dir,
                      const std::string& patch_file) -> VoidResult;
    auto list_patches(const std::string& patch_dir) -> Result<std::vector<PatchInfo>>;
    auto check_applied(std::stop_token token, const std::string& target_dir,
                       const std::string& patch_file) -> bool;
};
```

---

## Rootfs Package

### Builder

`src/domain/rootfs/builder.hpp`

```cpp
struct RootfsOptions {
    std::string distribution = "debian";
    std::string release = "bookworm";
    std::vector<std::string> packages;
    std::string post_build_script;
};

class Builder {
public:
    explicit Builder(context::Context* ctx);

    auto create(std::stop_token token, RootfsOptions opts) -> VoidResult;
    auto install_modules(std::stop_token token) -> VoidResult;
    auto customize(std::stop_token token, const std::string& script) -> VoidResult;
    auto clean() -> VoidResult;
};
```

---

## Toolchain Package

### Manager

`src/domain/toolchain/manager.hpp`

```cpp
struct ToolchainInfo {
    std::string target, config_file;
    bool installed = false;
};

struct ToolchainPaths {
    std::string base_dir, x_tools, config_dir, build_dir, ct_ng_dir;
};

class Manager {
public:
    Manager(infra::executor::Executor* exec,
            infra::filesystem::FileSystem* fs,
            config::Config* cfg);

    auto paths() const -> const ToolchainPaths&;
    auto is_installed() const -> bool;
    auto install(std::stop_token token) -> VoidResult;
    auto build_toolchain(std::stop_token token, const std::string& target) -> VoidResult;
    auto get_installed_toolchains() -> Result<std::vector<ToolchainInfo>>;
    auto get_available_configs() -> Result<std::vector<std::string>>;
    auto get_bin_dir(const std::string& target) -> std::string;
};
```

---

All domain classes use pointer-based constructor injection. No class owns its dependencies — the `App` class holds all `std::unique_ptr`s.