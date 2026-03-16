// ============================================================================
// app/commands/status.cpp — Workspace status command
// ============================================================================

#include <app/app.hpp>
#include <config/workspaces.hpp>

#include "commands.hpp"

namespace elmos::app::commands {

void register_status(App& app, CLI::App& cli) {
    auto* status = cli.add_subcommand("status", "Show workspace status");
    status->callback([&app] {
        auto& cfg = app.config();

        // Show active workspace
        auto active = config::WorkspaceManager::get_active_workspace();
        if (active) {
            app.printer().step("Active workspace: {}", *active);
        }
        else {
            app.printer().warn("No active workspace — run 'elmos init <name>'");
        }

        app.printer().print("  Project root:  {}", cfg.paths.project_root);
        app.printer().print("  Architecture:  {}", cfg.build.arch);
        app.printer().print("  Mount point:   {}", cfg.image.mount_point);
        app.printer().print("  Image:         {}", cfg.image.path);

        if (app.context().is_mounted()) {
            app.printer().success("Volume is mounted");
        }
        else {
            app.printer().warn("Volume is NOT mounted");
        }

        if (app.context().kernel_exists()) {
            app.printer().success("Kernel source found");
        }
        else {
            app.printer().info("No kernel source");
        }

        if (app.context().has_kernel_image()) {
            app.printer().success("Kernel is built");
        }
        else {
            app.printer().info("Kernel not built");
        }
    });
}

}  // namespace elmos::app::commands
