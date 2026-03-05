// ============================================================================
// app/commands/tui.cpp — Launch interactive TUI
// ============================================================================

#include <app/app.hpp>
#include <ui/tui/app.hpp>

#include "commands.hpp"

namespace elmos::app::commands {

void register_tui(App& app, CLI::App& cli) {
    auto* tui = cli.add_subcommand("tui", "Launch interactive TUI");
    tui->callback([] { ui::tui::run_tui(); });
}

}  // namespace elmos::app::commands
