// ============================================================================
// app/commands/init.cpp — Workspace initialization command
// ============================================================================

#include <app/app.hpp>
#include <config/defaults.hpp>
#include <config/workspaces.hpp>

#include "commands.hpp"

#include <cstdlib>
#include <filesystem>
#include <format>

namespace elmos::app::commands {
namespace fs = std::filesystem;

void register_init(App& app, CLI::App& cli) {
    auto* sub = cli.add_subcommand("init", "Initialize workspace (mount volume)");
    auto* name_opt = sub->add_option("workspace_name", "Workspace name");
    auto* size_opt = sub->add_option("size", "Volume size (e.g. 40G)");

    sub->callback([&app, name_opt, size_opt] {
        auto& cfg = app.config();

        std::string name = name_opt->count() > 0 ? name_opt->as<std::string>()
                                                 : (cfg.image.volume_name.empty()
                                                        ? std::string(config::kDefaultVolumeName)
                                                        : cfg.image.volume_name);

        std::string size = size_opt->count() > 0
                               ? size_opt->as<std::string>()
                               : (cfg.image.size.empty() ? std::string(config::kDefaultImageSize)
                                                         : cfg.image.size);

        cfg.image.volume_name = name;
        cfg.image.size = size;

        // Set up workspace directory
        auto cwd = fs::current_path();
        auto local_dir = cwd / name;
        fs::create_directories(local_dir);

        cfg.paths.project_root = local_dir.string();

        // Initialize workspace structure
        config::WorkspaceManager ws(local_dir.string());
        if (auto r = ws.initialize(); !r) {
            app.printer().warn("Workspace structure init failed");
        }

        app.printer().success("Workspace initialized at {}", local_dir.string());
        app.printer().info("Next step: cd {}", name);
    });
}

}  // namespace elmos::app::commands
