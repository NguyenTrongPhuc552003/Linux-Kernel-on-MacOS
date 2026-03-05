// ============================================================================
// app/commands/bsp.cpp — Board Support Package commands
// ============================================================================

#include <app/app.hpp>

#include "commands.hpp"

namespace elmos::app::commands {

void register_bsp(App& app, CLI::App& cli) {
    auto* bsp = cli.add_subcommand("bsp", "Board support package management");

    // bsp list
    auto* list_cmd = bsp->add_subcommand("list", "List available machines");
    list_cmd->callback([&app] {
        app.printer().step("Available machines:");
        app.printer().info("BSP registry not yet fully implemented (Phase 3)");
        app.printer().print("  • rock5b_plus (built-in)");
    });

    // bsp fetch <machine>
    auto* fetch_cmd = bsp->add_subcommand("fetch", "Fetch machine definition");
    std::string machine;
    fetch_cmd->add_option("machine", machine, "Machine name")->required();
    fetch_cmd->callback([&app, &machine] {
        app.printer().step("Fetching machine definition: {}", machine);
        app.printer().info("Registry fetch not yet implemented (Phase 3)");
    });
}

}  // namespace elmos::app::commands
