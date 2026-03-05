// ============================================================================
// app/commands/bootloader.cpp — Bootloader build commands
// ============================================================================

#include <app/app.hpp>

#include "commands.hpp"

namespace elmos::app::commands {

void register_bootloader(App& app, CLI::App& cli) {
    auto* bl = cli.add_subcommand("bootloader", "Build and install U-Boot");

    // bootloader build
    auto* build_cmd = bl->add_subcommand("build", "Build bootloader");
    build_cmd->callback([&app] {
        app.printer().step("Building bootloader...");
        app.printer().info("Bootloader build not yet implemented (Phase 3)");
    });

    // bootloader install
    auto* install_cmd = bl->add_subcommand("install", "Install bootloader artifacts");
    install_cmd->callback([&app] {
        app.printer().step("Installing bootloader...");
        app.printer().info("Bootloader install not yet implemented (Phase 3)");
    });

    // bootloader blobs
    auto* blobs_cmd = bl->add_subcommand("blobs", "List/download firmware blobs");
    blobs_cmd->callback([&app] {
        app.printer().step("Firmware blobs:");
        app.printer().info("Blob management not yet implemented (Phase 3)");
    });
}

}  // namespace elmos::app::commands
