// ============================================================================
// app/app.cpp — Application container implementation
// ============================================================================

#include "app.hpp"

#include <domain/plugin/builtin.hpp>
#include <infra/platform/interface.hpp>
#include <ui/help.hpp>

#include "commands/commands.hpp"

namespace elmos::app {

App::App(std::unique_ptr<infra::executor::Executor> exec,
         std::unique_ptr<infra::filesystem::FileSystem> fs, config::Config cfg)
    : exec_(std::move(exec)), fs_(std::move(fs)), config_(std::move(cfg)),
      cli_("elmos", "Embedded Linux SDK - Native kernel build tools") {

    // Build context
    context_ = std::make_unique<context::Context>(&config_, exec_.get(), fs_.get());

    // Create platform
    platform_ = infra::platform::create_platform(exec_.get());

    // Domain services
    toolchain_manager_ = std::make_unique<domain::toolchain::Manager>(exec_.get(), fs_.get(),
                                                                      &config_);
    kernel_builder_ = std::make_unique<domain::builder::KernelBuilder>(context_.get(),
                                                                       toolchain_manager_.get());
    module_builder_ = std::make_unique<domain::builder::ModuleBuilder>(context_.get(),
                                                                       toolchain_manager_.get());
    app_builder_ = std::make_unique<domain::builder::AppBuilder>(context_.get(),
                                                                 toolchain_manager_.get());
    qemu_runner_ = std::make_unique<domain::emulator::QEMURunner>(context_.get());
    health_checker_ = std::make_unique<domain::doctor::HealthChecker>(
        exec_.get(), fs_.get(), &config_, platform_.get(), toolchain_manager_.get());
    rootfs_builder_ = std::make_unique<domain::rootfs::Builder>(context_.get());
    patcher_ = std::make_unique<domain::patch::Patcher>(exec_.get(), fs_.get());

    // Plugin system
    hook_executor_ = std::make_unique<plugin::HookExecutor>();
    plugin_registry_ = std::make_unique<plugin::Registry>(hook_executor_.get());

    // Register builtin plugin factories
    domain::plugin::builtin::register_all();

    // Load builtin plugins
    auto err = plugin_registry_->load_builtins(context_.get(), {});
    if (!err) {
        printer_.warn("Failed to load some plugins");
    }
}

auto App::build_cli() -> CLI::App& {
    cli_.set_help_all_flag("--help-all", "Show all help");
    cli_.formatter(std::make_shared<ui::HelpFormatter>());

    // Global flags
    cli_.add_flag("-e,--verbose", verbose_, "Enable verbose output");
    cli_.add_option("-c,--config", config_file_, "Config file path");

    // Register all subcommands
    register_commands();

    return cli_;
}

auto App::run(int argc, char** argv) -> int {
    build_cli();
    CLI11_PARSE(cli_, argc, argv);
    return 0;
}

void App::register_commands() {
    commands::register_all(*this, cli_);
}

}  // namespace elmos::app
