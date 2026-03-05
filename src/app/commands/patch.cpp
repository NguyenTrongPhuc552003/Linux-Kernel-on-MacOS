// ============================================================================
// app/commands/patch.cpp — Patch management commands
// ============================================================================

#include <app/app.hpp>

#include "commands.hpp"

#include <stop_token>

namespace elmos::app::commands {

void register_patch(App& app, CLI::App& cli) {
    auto* patch = cli.add_subcommand("patch", "Apply patches to kernel source");

    // patch apply
    auto* apply_cmd = patch->add_subcommand("apply", "Apply patches from patch directory");
    apply_cmd->callback([&app] {
        auto& cfg = app.config();
        if (cfg.paths.patches_dir.empty()) {
            app.printer().info("No patches directory configured");
            return;
        }
        app.printer().step("Applying patches from {}...", cfg.paths.patches_dir);
        std::stop_source ss;
        if (auto r = app.patcher().apply(ss.get_token(), cfg.paths.patches_dir,
                                         cfg.paths.kernel_dir);
            !r) {
            app.printer().error("Patch failed: {}", r.error().message());
            return;
        }
        app.printer().success("Patches applied!");
    });
}

}  // namespace elmos::app::commands
