// ============================================================================
// app/commands/exit.cpp — Workspace exit (unmount & cleanup) command
// ============================================================================

#include <app/app.hpp>
#include <config/workspaces.hpp>
#include <plugin/interface.hpp>

#include "commands.hpp"

#include <filesystem>
#include <stop_token>

namespace elmos::app::commands {
namespace fs = std::filesystem;

void register_exit(App& app, CLI::App& cli) {
    auto* sub = cli.add_subcommand("exit", "Unmount workspace volume and cleanup");
    auto* clean_flag = sub->add_flag("--clean", "Also remove build cache");

    sub->callback([&app, clean_flag] {
        std::stop_source ss;
        auto token = ss.get_token();

        auto& ctx = app.context();
        auto& printer = app.printer();

        // Check if mounted
        if (!ctx.is_mounted(token)) {
            printer.info("No volume mounted — nothing to do");
            return;
        }

        // Fire cleanup hooks if registered
        auto& hooks = app.hook_executor();
        if (hooks.has_hooks(plugin::events::kOnCleanup)) {
            plugin::Event cleanup_event{plugin::events::kOnCleanup};
            hooks.execute(token, cleanup_event);
        }

        // Unmount the disk image
        auto mount_result = ctx.get_actual_mount_point(token);
        if (!mount_result) {
            printer.warn("Could not determine mount point: {}", mount_result.error().message());
            return;
        }

        printer.info("Unmounting {}...", *mount_result);

        auto& disk = ctx.platform().disk_image();
        auto unmount_result = disk.unmount(token, *mount_result);
        if (!unmount_result) {
            printer.error("Unmount failed: {}", unmount_result.error().message());
            return;
        }

        // Optionally clean build cache
        if (clean_flag->count() > 0) {
            auto ws_root = config::WorkspaceManager::find_workspace_root();
            if (ws_root) {
                config::WorkspaceManager ws(*ws_root);
                if (auto r = ws.clean_cache(); !r) {
                    printer.warn("Cache cleanup failed: {}", r.error().message());
                }
                else {
                    printer.info("Build cache cleaned");
                }
            }
        }

        printer.success("Workspace exited cleanly");
    });
}

}  // namespace elmos::app::commands
