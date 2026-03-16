#pragma once
// ============================================================================
// app/app.hpp — Application container with dependency wiring
// ============================================================================

#include <config/loader.hpp>
#include <config/types.hpp>
#include <context/context.hpp>
#include <domain/bsp/registry.hpp>
#include <domain/builder/app.hpp>
#include <domain/builder/kernel.hpp>
#include <domain/builder/module.hpp>
#include <domain/doctor/checker.hpp>
#include <domain/emulator/qemu.hpp>
#include <domain/patch/patcher.hpp>
#include <domain/rootfs/builder.hpp>
#include <domain/toolchain/manager.hpp>
#include <infra/executor/interface.hpp>
#include <infra/filesystem/interface.hpp>
#include <infra/platform/interface.hpp>
#include <plugin/executor.hpp>
#include <plugin/loader.hpp>
#include <ui/printer.hpp>

#include <CLI/CLI.hpp>

#include <memory>
#include <string>

namespace elmos::app {

/// App holds all application dependencies and builds the CLI.
class App {
public:
    App(std::unique_ptr<infra::executor::Executor> exec,
        std::unique_ptr<infra::filesystem::FileSystem> fs, config::Config cfg);

    /// Build the root CLI11 application with all subcommands.
    auto build_cli() -> CLI::App&;

    /// Run the CLI (parse args and execute).
    auto run(int argc, char** argv) -> int;

    // Accessors for command handlers
    auto exec() -> infra::executor::Executor& { return *exec_; }
    auto fs() -> infra::filesystem::FileSystem& { return *fs_; }
    auto config() -> config::Config& { return config_; }
    auto context() -> context::Context& { return *context_; }
    auto printer() -> ui::Printer& { return printer_; }
    auto kernel_builder() -> domain::builder::KernelBuilder& { return *kernel_builder_; }
    auto module_builder() -> domain::builder::ModuleBuilder& { return *module_builder_; }
    auto app_builder() -> domain::builder::AppBuilder& { return *app_builder_; }
    auto qemu_runner() -> domain::emulator::QEMURunner& { return *qemu_runner_; }
    auto health_checker() -> domain::doctor::HealthChecker& { return *health_checker_; }
    auto rootfs_builder() -> domain::rootfs::Builder& { return *rootfs_builder_; }
    auto patcher() -> domain::patch::Patcher& { return *patcher_; }
    auto toolchain_manager() -> domain::toolchain::Manager& { return *toolchain_manager_; }
    auto hook_executor() -> plugin::HookExecutor& { return *hook_executor_; }
    auto plugin_registry() -> plugin::Registry& { return *plugin_registry_; }
    auto platform() -> infra::platform::Platform& { return *platform_; }

    auto save_workspace_config() -> VoidResult;

private:
    // Owned infrastructure
    std::unique_ptr<infra::executor::Executor> exec_;
    std::unique_ptr<infra::filesystem::FileSystem> fs_;
    config::Config config_;
    std::unique_ptr<context::Context> context_;
    std::unique_ptr<infra::platform::Platform> platform_;
    ui::Printer printer_;

    // Domain services
    std::unique_ptr<domain::toolchain::Manager> toolchain_manager_;
    std::unique_ptr<domain::builder::KernelBuilder> kernel_builder_;
    std::unique_ptr<domain::builder::ModuleBuilder> module_builder_;
    std::unique_ptr<domain::builder::AppBuilder> app_builder_;
    std::unique_ptr<domain::emulator::QEMURunner> qemu_runner_;
    std::unique_ptr<domain::doctor::HealthChecker> health_checker_;
    std::unique_ptr<domain::rootfs::Builder> rootfs_builder_;
    std::unique_ptr<domain::patch::Patcher> patcher_;

    // Plugin system
    std::unique_ptr<plugin::HookExecutor> hook_executor_;
    std::unique_ptr<plugin::Registry> plugin_registry_;

    // CLI
    CLI::App cli_;
    bool verbose_ = false;
    std::string config_file_;

    void register_commands();
    void ensure_first_run_setup();
};

}  // namespace elmos::app
