#pragma once
// ============================================================================
// app/commands/commands.hpp — Command registration
// ============================================================================

#include <CLI/CLI.hpp>

namespace elmos::app {
class App;
}

namespace elmos::app::commands {

/// Register all CLI subcommands on the given CLI::App.
void register_all(App& app, CLI::App& cli);

// Individual command builders
void register_workspace(App& app, CLI::App& cli);
void register_kernel(App& app, CLI::App& cli);
void register_module(App& app, CLI::App& cli);
void register_apps(App& app, CLI::App& cli);
void register_doctor(App& app, CLI::App& cli);
void register_arch(App& app, CLI::App& cli);
void register_qemu(App& app, CLI::App& cli);
void register_toolchain(App& app, CLI::App& cli);
void register_rootfs(App& app, CLI::App& cli);
void register_bootloader(App& app, CLI::App& cli);
void register_patch(App& app, CLI::App& cli);
void register_plugin(App& app, CLI::App& cli);
void register_version(App& app, CLI::App& cli);
void register_tui(App& app, CLI::App& cli);
void register_bsp(App& app, CLI::App& cli);

}  // namespace elmos::app::commands
