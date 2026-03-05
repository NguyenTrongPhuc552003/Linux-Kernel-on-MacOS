// ============================================================================
// app/commands/plugins.cpp — Plugin management commands
// ============================================================================

#include <app/app.hpp>

#include "commands.hpp"

namespace elmos::app::commands {

void register_plugins(App& app, CLI::App& cli) {
    auto* plugins = cli.add_subcommand("plugins", "Manage plugins");

    // plugins list
    auto* list_cmd = plugins->add_subcommand("list", "List loaded plugins");
    list_cmd->callback([&app] {
        auto loaded = app.plugin_registry().list_plugins();
        if (loaded.empty()) {
            app.printer().info("No plugins loaded");
            return;
        }
        app.printer().step("Loaded plugins ({}):", loaded.size());
        for (auto* p : loaded) {
            if (p) {
                app.printer().print("  • {} v{} — {}", p->name(), p->version(), p->description());
            }
        }
    });
}

}  // namespace elmos::app::commands
