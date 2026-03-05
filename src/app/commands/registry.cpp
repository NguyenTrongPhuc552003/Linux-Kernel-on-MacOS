// ============================================================================
// app/commands/registry.cpp — Wires all subcommands to CLI::App
// ============================================================================

#include "commands.hpp"

namespace elmos::app::commands {

void register_all(App& app, CLI::App& cli) {
    register_init(app, cli);
    register_kernel(app, cli);
    register_module(app, cli);
    register_apps(app, cli);
    register_doctor(app, cli);
    register_arch(app, cli);
    register_qemu(app, cli);
    register_toolchain(app, cli);
    register_rootfs(app, cli);
    register_bootloader(app, cli);
    register_patch(app, cli);
    register_plugins(app, cli);
    register_version(app, cli);
    register_tui(app, cli);
    register_status(app, cli);
    register_bsp(app, cli);
}

}  // namespace elmos::app::commands
