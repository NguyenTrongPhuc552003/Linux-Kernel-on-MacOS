// ============================================================================
// app/commands/pick.cpp — Select active workspace
// ============================================================================

#include <app/app.hpp>
#include <config/defaults.hpp>
#include <config/workspaces.hpp>
#include <infra/platform/interface.hpp>

#include "commands.hpp"

#include <stop_token>
#include <thread>

namespace elmos::app::commands {

void register_pick(App& app, CLI::App& cli) {
    auto* sub = cli.add_subcommand("pick", "Select or list workspaces");
    auto* name_opt = sub->add_option("workspace", "Workspace to activate");

    sub->callback([&app, name_opt] {
        auto& printer = app.printer();

        if (name_opt->count() == 0) {
            // List all workspaces and show which is active
            auto active = config::WorkspaceManager::get_active_workspace();
            auto list = config::WorkspaceManager::list_workspaces();
            if (!list || list->empty()) {
                printer.info("No workspaces — run 'elmos init <name>' to create one");
                return;
            }

            printer.step("Workspaces:");
            for (auto& ws : *list) {
                bool is_active = active && *active == ws;
                printer.print("  {} {}", is_active ? "*" : " ", ws);
            }
            return;
        }

        auto name = name_opt->as<std::string>();
        auto result = config::WorkspaceManager::set_active_workspace(name);
        if (!result) {
            // Check if it's a mounted volume without config — auto-register
            std::stop_source ss;
            auto& disk = app.platform().disk_image();
            auto mounted = disk.is_mounted(ss.get_token(), name);
            if (mounted && mounted->mounted) {
                auto& platform = app.platform();
                auto ws_dir = config::WorkspaceManager::workspace_dir(name);
                std::string ext = (platform.name() == "darwin") ? ".sparseimage" : ".img";
                std::string image_path = ws_dir + "/" + name + ext;

                config::Config ws_cfg;
                ws_cfg.image.path = image_path;
                ws_cfg.image.volume_name = name;
                ws_cfg.image.size = "40G";
                ws_cfg.image.mount_point = mounted->mount_point;
                ws_cfg.paths.project_root = mounted->mount_point;
                ws_cfg.build.arch = std::string(config::kDefaultArch);
                ws_cfg.build.jobs = static_cast<int>(std::thread::hardware_concurrency());
                ws_cfg.build.llvm = true;
                ws_cfg.build.cross_compile = std::string(config::kDefaultCrossPrefix);
                ws_cfg.qemu.memory = std::string(config::kDefaultMemory);
                ws_cfg.qemu.gdb_port = config::kDefaultGDBPort;
                ws_cfg.qemu.ssh_port = config::kDefaultSSHPort;
                ws_cfg.qemu.smp = static_cast<int>(std::thread::hardware_concurrency());

                auto ws_path = config::WorkspaceManager::workspace_config_path(name);
                // Ensure workspace directory exists
                std::error_code dir_ec;
                fs::create_directories(fs::path(ws_path).parent_path(), dir_ec);
                if (auto save_r = ws_cfg.save(ws_path); !save_r) {
                    printer.error("Cannot register workspace: {}", save_r.error().message());
                    return;
                }
                result = config::WorkspaceManager::set_active_workspace(name);
            }
            if (!result) {
                printer.error("{}", result.error().message());
                return;
            }
        }
        printer.success("Active workspace: {}", name);
    });
}

}  // namespace elmos::app::commands
