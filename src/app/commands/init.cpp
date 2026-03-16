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
#include <fstream>
#include <stop_token>
#include <thread>

#include <embedded_resources.hpp>

namespace elmos::app::commands {
namespace fs = std::filesystem;

static auto parse_size_gb(const std::string& size) -> int {
    std::string numeric = size;
    if (numeric.ends_with("G") || numeric.ends_with("g"))
        numeric.pop_back();
    try {
        int gb = std::stoi(numeric);
        return gb < config::kMinimumImageSize ? config::kMinimumImageSize : gb;
    }
    catch (...) {
        return config::kMinimumImageSize;
    }
}

static void ensure_workspace_registered(const std::string& name, const std::string& image_path,
                                        const std::string& mount_point, const std::string& size) {
    auto ws_config_path = config::WorkspaceManager::workspace_config_path(name);
    if (fs::exists(ws_config_path))
        return;  // already registered

    // Create workspace directory at ~/.elmos/workspaces/<name>/
    std::error_code ec;
    fs::create_directories(fs::path(ws_config_path).parent_path(), ec);

    config::Config ws_cfg;
    ws_cfg.image.path = image_path;
    ws_cfg.image.volume_name = name;
    ws_cfg.image.size = size;
    ws_cfg.image.mount_point = mount_point;
    ws_cfg.paths.project_root = mount_point;
    ws_cfg.build.arch = std::string(config::kDefaultArch);
    ws_cfg.build.jobs = static_cast<int>(std::thread::hardware_concurrency());
    ws_cfg.build.llvm = true;
    ws_cfg.build.cross_compile = std::string(config::kDefaultCrossPrefix);
    ws_cfg.qemu.memory = std::string(config::kDefaultMemory);
    ws_cfg.qemu.gdb_port = config::kDefaultGDBPort;
    ws_cfg.qemu.ssh_port = config::kDefaultSSHPort;
    ws_cfg.qemu.smp = static_cast<int>(std::thread::hardware_concurrency());

    ws_cfg.save(ws_config_path);
}

static void init_one_workspace(App& app, const std::string& name, const std::string& size,
                               std::stop_token token) {
    auto& printer = app.printer();
    auto& platform = app.context().platform();
    auto& disk = platform.disk_image();
    int size_gb = parse_size_gb(size);

    auto ws_dir = config::WorkspaceManager::workspace_dir(name);
    {
        std::error_code ec;
        fs::create_directories(ws_dir, ec);
    }
    auto plat_name = platform.name();
    std::string ext = (plat_name == "darwin") ? ".sparseimage" : ".img";
    std::string image_path = ws_dir + "/" + name + ext;

    // Check if already mounted — register config + return early
    auto mounted = disk.is_mounted(token, name);
    if (mounted && mounted->mounted) {
        printer.info("'{}' already mounted at {}", name, mounted->mount_point);
        ensure_workspace_registered(name, image_path, mounted->mount_point, size);
        printer.success("Workspace '{}' ready at {}", name, mounted->mount_point);
        return;
    }

    // Create disk image (idempotent — skips if image already exists)
    printer.info("Creating '{}' ({}G)...", name, size_gb);
    if (auto r = disk.create(token, image_path, size_gb); !r) {
        printer.error("Failed to create image for '{}': {}", name, r.error().message());
        return;
    }

    // Mount the volume
    printer.info("Mounting '{}'...", name);
    auto mount_result = disk.mount(token, image_path);
    if (!mount_result) {
        printer.error("Failed to mount '{}': {}", name, mount_result.error().message());
        return;
    }
    auto mount_point = *mount_result;

    // Scaffold workspace structure at mount point
    config::WorkspaceManager ws(mount_point);
    if (auto r = ws.initialize(); !r) {
        printer.warn("Workspace structure init failed for '{}'", name);
    }

    // Extract embedded resources
    std::error_code ec;
    int file_count = 0;
    for (auto path : elmos::resources::list()) {
        std::string dest_sub;
        if (path.starts_with("examples/"))
            dest_sub = std::string(path);
        else if (path.starts_with("toolchains/configs/"))
            dest_sub = std::string(path);
        else
            continue;

        auto dest = fs::path(mount_point) / dest_sub;
        fs::create_directories(dest.parent_path(), ec);
        if (ec)
            continue;

        auto content = elmos::resources::get(path).value_or(std::string_view{});
        std::ofstream out(dest, std::ios::binary | std::ios::trunc);
        if (out) {
            out.write(content.data(), static_cast<std::streamsize>(content.size()));
            ++file_count;
        }
    }

    // Register workspace config
    ensure_workspace_registered(name, image_path, mount_point, size);

    printer.success("Workspace '{}' ready at {}", name, mount_point);
    if (file_count > 0)
        printer.info("  Extracted {} files", file_count);
}

void register_init(App& app, CLI::App& cli) {
    auto* sub = cli.add_subcommand("init", "Initialize workspace(s) — create disk image and mount");
    auto* names = sub->add_option("names", "Workspace name(s) to create")->expected(1, -1);
    auto* size_opt = sub->add_option("-s,--size", "Volume size (default: 40G)");

    sub->callback([&app, names, size_opt] {
        std::stop_source ss;
        auto token = ss.get_token();

        auto ws_names = names->as<std::vector<std::string>>();
        std::string size = size_opt->count() > 0 ? size_opt->as<std::string>()
                                                 : std::string(config::kDefaultImageSize);

        for (auto& name : ws_names) {
            init_one_workspace(app, name, size, token);
        }

        // Set the first workspace as active
        if (!ws_names.empty()) {
            if (auto r = config::WorkspaceManager::set_active_workspace(ws_names.front()); !r) {
                app.printer().warn("Could not set active workspace: {}", r.error().message());
            }
            else {
                app.printer().info("Active workspace: {}", ws_names.front());
            }
        }
    });
}

}  // namespace elmos::app::commands
