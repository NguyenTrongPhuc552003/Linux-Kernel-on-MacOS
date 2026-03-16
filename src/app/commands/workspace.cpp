// ============================================================================
// app/commands/workspace.cpp — Workspace management compound command
//   elmos workspace list
//   elmos workspace init <name> [names...]
//   elmos workspace <name>
//   elmos workspace exit [name...]||all
//   elmos workspace show [name...]
//   elmos workspace clean <name>||all
// ============================================================================

#include <app/app.hpp>
#include <config/defaults.hpp>
#include <config/loader.hpp>
#include <config/workspaces.hpp>
#include <plugin/interface.hpp>
#include <ui/printer.hpp>

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

// ── helpers (migrated from init.cpp) ────────────────────────────────────────

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
        return;

    // Create workspace directory at ~/.elmos/workspaces/<name>/
    auto ws_dir = config::WorkspaceManager::workspace_dir(name);
    std::error_code ec;
    fs::create_directories(ws_dir, ec);

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

    // Already mounted — register + return early
    auto mounted = disk.is_mounted(token, name);
    if (mounted && mounted->mounted) {
        printer.info("'{}' already mounted at {}", name, mounted->mount_point);
        ensure_workspace_registered(name, image_path, mounted->mount_point, size);
        printer.success("Workspace '{}' ready at {}", name, mounted->mount_point);
        return;
    }

    // Create disk image (idempotent)
    printer.info("Creating '{}' ({}G)...", name, size_gb);
    if (auto r = disk.create(token, image_path, size_gb); !r) {
        printer.error("Failed to create image for '{}': {}", name, r.error().message());
        return;
    }

    // Mount
    printer.info("Mounting '{}'...", name);
    auto mount_result = disk.mount(token, image_path);
    if (!mount_result) {
        printer.error("Failed to mount '{}': {}", name, mount_result.error().message());
        return;
    }
    auto mount_point = *mount_result;

    // Scaffold workspace structure
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

// ── pick helper ─────────────────────────────────────────────────────────────

static auto resolve_pick_image_path(App& app, const std::string& pick_input) -> std::string {
    // Existing path (absolute or relative) wins as-is.
    if (fs::exists(pick_input))
        return pick_input;

    const auto input_path = fs::path(pick_input);

    // If caller provided an explicit path-like token, do not guess by name.
    if (input_path.has_parent_path() || input_path.has_extension())
        return pick_input;

    // Treat plain token as workspace name and probe known image locations.
    const auto workspace_name = pick_input;
    const auto ws_dir = config::WorkspaceManager::workspace_dir(workspace_name);
    const auto cache_dir = app.platform().paths().cache_dir();

    for (const auto& base_dir : {ws_dir, cache_dir}) {
        for (const auto* ext : {".sparseimage", ".img"}) {
            const auto candidate = (fs::path(base_dir) / (workspace_name + ext)).string();
            if (fs::exists(candidate))
                return candidate;
        }
    }

    return pick_input;
}

static auto fit_cell(std::string text, std::size_t width) -> std::string {
    if (text.size() > width) {
        if (width > 3)
            return text.substr(0, width - 3) + "...";
        return text.substr(0, width);
    }
    text.append(width - text.size(), ' ');
    return text;
}

static auto pick_one_workspace(App& app, const std::string& pick_input, std::stop_token token)
    -> std::optional<std::string> {
    auto& printer = app.printer();
    auto& platform = app.context().platform();
    auto& disk = platform.disk_image();

    auto image_path = resolve_pick_image_path(app, pick_input);

    if (!fs::exists(image_path)) {
        printer.error("Image file not found: {}", pick_input);
        return std::nullopt;
    }

    auto name = fs::path(image_path).stem().string();

    // Already mounted — register + return early
    auto mounted = disk.is_mounted(token, name);
    if (mounted && mounted->mounted) {
        printer.info("'{}' already mounted at {}", name, mounted->mount_point);
        std::string size_str = std::string(config::kDefaultImageSize);
        auto fsize = fs::file_size(image_path);
        if (fsize > 0)
            size_str = std::to_string(fsize / (1024ULL * 1024 * 1024)) + "G";
        ensure_workspace_registered(name, image_path, mounted->mount_point, size_str);
        printer.success("Workspace '{}' ready at {}", name, mounted->mount_point);
        return name;
    }

    // Mount the existing image
    printer.info("Mounting '{}' from {}...", name, image_path);
    auto mount_result = disk.mount(token, image_path);
    if (!mount_result) {
        printer.error("Failed to mount '{}': {}", name, mount_result.error().message());
        return std::nullopt;
    }
    auto mount_point = *mount_result;

    // Scaffold workspace structure at mount point
    config::WorkspaceManager ws(mount_point);
    if (auto r = ws.initialize(); !r) {
        printer.warn("Workspace structure init failed for '{}'", name);
    }

    // Determine size from actual file
    std::string size_str = std::string(config::kDefaultImageSize);
    auto fsize = fs::file_size(image_path);
    if (fsize > 0)
        size_str = std::to_string(fsize / (1024ULL * 1024 * 1024)) + "G";

    // Register workspace config
    ensure_workspace_registered(name, image_path, mount_point, size_str);

    printer.success("Workspace '{}' ready at {}", name, mount_point);
    return name;
}

// ── clean helper ────────────────────────────────────────────────────────────

static void clean_one_workspace(App& app, const std::string& name, std::stop_token token) {
    auto& printer = app.printer();
    auto& platform = app.platform();
    auto& disk = platform.disk_image();

    // Unmount if mounted
    auto mounted = disk.is_mounted(token, name);
    if (mounted && mounted->mounted) {
        printer.info("Unmounting '{}'...", name);
        if (auto r = disk.unmount(token, mounted->mount_point); !r) {
            printer.error("Unmount failed for '{}': {}", name, r.error().message());
            return;
        }
    }

    // Remove workspace directory (~/.elmos/workspaces/<name>/) — contains image + config
    auto ws_dir = config::WorkspaceManager::workspace_dir(name);
    std::error_code ec;
    if (fs::is_directory(ws_dir, ec)) {
        fs::remove_all(ws_dir, ec);
        if (ec) {
            printer.error("Failed to remove workspace dir for '{}': {}", name, ec.message());
            return;
        }
    }

    // Also clean up legacy image at cache_dir root (backward compatibility)
    auto cache_dir = platform.paths().cache_dir();
    for (auto* ext : {".sparseimage", ".img"}) {
        auto legacy_path = fs::path(cache_dir) / (name + ext);
        if (fs::exists(legacy_path, ec))
            fs::remove(legacy_path, ec);
    }

    // If this was the active workspace, clear active
    auto active = config::WorkspaceManager::get_active_workspace();
    if (active && *active == name) {
        auto active_path = config::WorkspaceManager::global_elmos_dir() + "/active";
        fs::remove(active_path, ec);
    }

    printer.success("Workspace '{}' cleaned", name);
}

// ── workspace command registration ──────────────────────────────────────────

void register_workspace(App& app, CLI::App& cli) {
    auto* ws = cli.add_subcommand("workspace", "Manage workspaces");

    // workspace list
    auto* list_cmd = ws->add_subcommand("list", "List all available workspaces");
    list_cmd->callback([&app] {
        auto active = config::WorkspaceManager::get_active_workspace();
        auto list = config::WorkspaceManager::list_workspaces();
        if (!list || list->empty()) {
            app.printer().info("No workspaces — run 'elmos workspace init <name>' to create one");
            return;
        }
        app.printer().step("Workspaces:");
        for (auto& ws_name : *list) {
            bool is_active = active && *active == ws_name;
            app.printer().print("  {} {}", is_active ? "*" : " ", ws_name);
        }
    });

    // workspace init <name> [names...] [-p|--pick <path>]
    auto* init_cmd = ws->add_subcommand("init",
                                        "Initialize workspace(s) — create disk image and mount");
    auto* names = init_cmd->add_option("names", "Workspace name(s) to create")->expected(1, -1);
    auto* size_opt = init_cmd->add_option("-s,--size", "Volume size (default: 40G)");
    auto* pick_opt =
        init_cmd
            ->add_option("-p,--pick",
                         "Pick existing workspace image path(s) or workspace name(s) to mount")
            ->expected(1, -1);
    names->excludes(pick_opt);
    pick_opt->excludes(names);
    init_cmd->callback([&app, names, size_opt, pick_opt] {
        std::stop_source ss;
        auto token = ss.get_token();

        // Handle --pick: mount existing image file(s)
        if (pick_opt->count() > 0) {
            auto paths = pick_opt->as<std::vector<std::string>>();
            std::string first_name;
            for (auto& path : paths) {
                auto picked_name = pick_one_workspace(app, path, token);
                if (first_name.empty() && picked_name)
                    first_name = *picked_name;
            }
            if (!first_name.empty()) {
                if (auto r = config::WorkspaceManager::set_active_workspace(first_name); !r)
                    app.printer().warn("Could not set active workspace: {}", r.error().message());
                else
                    app.printer().info("Active workspace: {}", first_name);
            }
            return;
        }

        auto ws_names = names->as<std::vector<std::string>>();

        std::string size = size_opt->count() > 0 ? size_opt->as<std::string>()
                                                 : std::string(config::kDefaultImageSize);

        for (auto& name : ws_names) {
            init_one_workspace(app, name, size, token);
        }

        // Auto-select first workspace as active
        if (!ws_names.empty()) {
            if (auto r = config::WorkspaceManager::set_active_workspace(ws_names.front()); !r) {
                app.printer().warn("Could not set active workspace: {}", r.error().message());
            }
            else {
                app.printer().info("Active workspace: {}", ws_names.front());
            }
        }
    });

    // workspace clean <name>||all
    auto* clean_cmd = ws->add_subcommand("clean", "Unmount and remove workspace image(s)");
    auto* clean_name = clean_cmd->add_option("name", "Workspace name or 'all'")->required();
    clean_cmd->callback([&app, clean_name] {
        std::stop_source ss;
        auto token = ss.get_token();
        auto name = clean_name->as<std::string>();

        if (name == "all") {
            auto list = config::WorkspaceManager::list_workspaces();
            if (!list || list->empty()) {
                app.printer().info("No workspaces to clean");
                return;
            }
            for (auto& ws_name : *list) {
                clean_one_workspace(app, ws_name, token);
            }
            app.printer().success("All workspaces cleaned");
        }
        else {
            clean_one_workspace(app, name, token);
        }
    });

    // workspace exit [name...]||all — unmount workspace(s) but keep image files
    auto* exit_cmd = ws->add_subcommand("exit", "Unmount workspace(s) — keeps image files");
    auto* exit_names = exit_cmd->add_option("names", "Workspace name(s) or 'all' (default: active)")
                           ->expected(0, -1);
    exit_cmd->callback([&app, exit_names] {
        std::stop_source ss;
        auto token = ss.get_token();
        auto& printer = app.printer();
        auto& disk = app.platform().disk_image();

        std::vector<std::string> targets;
        if (exit_names->count() > 0) {
            targets = exit_names->as<std::vector<std::string>>();
        }

        // "all" — gather every registered workspace
        if (targets.size() == 1 && targets.front() == "all") {
            auto list = config::WorkspaceManager::list_workspaces();
            if (!list || list->empty()) {
                printer.info("No workspaces to exit");
                return;
            }
            targets = *list;
        }

        // No argument — use the current active workspace
        if (targets.empty()) {
            auto active = config::WorkspaceManager::get_active_workspace();
            if (!active) {
                printer.error("No active workspace. Specify a name or use 'all'.");
                return;
            }
            targets.push_back(*active);
        }

        for (auto& name : targets) {
            auto mounted = disk.is_mounted(token, name);
            if (!mounted || !mounted->mounted) {
                printer.info("'{}' is not mounted — nothing to do", name);
                continue;
            }

            printer.info("Unmounting '{}'...", name);
            if (auto r = disk.unmount(token, mounted->mount_point); !r) {
                printer.error("Unmount failed for '{}': {}", name, r.error().message());
                continue;
            }

            // Clear active pointer if this was the active workspace
            auto active = config::WorkspaceManager::get_active_workspace();
            if (active && *active == name) {
                std::error_code ec;
                fs::remove(config::WorkspaceManager::global_elmos_dir() + "/active", ec);
            }

            printer.success("Workspace '{}' exited (image preserved)", name);
        }
    });

    // workspace show [name...] — display detailed workspace information
    auto* show_cmd = ws->add_subcommand("show", "Show workspace details");
    auto* show_names =
        show_cmd->add_option("names", "Workspace name(s) (default: active)")->expected(0, -1);
    show_cmd->callback([&app, show_names] {
        std::stop_source ss;
        auto token = ss.get_token();
        auto& printer = app.printer();
        auto& disk = app.platform().disk_image();

        auto active = config::WorkspaceManager::get_active_workspace();

        std::vector<std::string> targets;
        if (show_names->count() > 0) {
            targets = show_names->as<std::vector<std::string>>();
        }
        else {
            // Default: show active workspace
            if (!active) {
                printer.error("No active workspace. Specify a name or run 'elmos workspace init'.");
                return;
            }
            targets.push_back(*active);
        }

        for (auto& name : targets) {
            bool is_active = active && *active == name;

            // Load workspace config
            auto cfg_path = config::WorkspaceManager::workspace_config_path(name);
            if (!fs::exists(cfg_path)) {
                printer.error("Workspace '{}' not found (no config at {})", name, cfg_path);
                continue;
            }

            auto cfg_result = config::load(cfg_path);
            if (!cfg_result) {
                printer.error("Cannot load config for '{}': {}", name,
                              cfg_result.error().message());
                continue;
            }
            auto& cfg = *cfg_result;

            // Check mount status
            auto mounted = disk.is_mounted(token, name);
            bool is_mounted = mounted && mounted->mounted;
            std::string mount_point = is_mounted ? mounted->mount_point : cfg.image.mount_point;

            // Use image path from stored config — avoids platform mismatch on
            // OrbStack/WSL2 where the workspace was created on a different OS.
            auto image_path = fs::path(cfg.image.path);

            constexpr std::size_t kInnerWidth = 74;
            const auto border = std::string(ui::color::kMagenta);
            const auto reset = std::string(ui::color::kReset);
            const auto title_color = std::string(ui::color::kBold) + std::string(ui::color::kCyan);
            const auto section_color = std::string(ui::color::kBold) +
                                       std::string(ui::color::kYellow);

            auto print_top = [&](const std::string& title) {
                auto fitted_title = fit_cell(title, kInnerWidth);
                printer.print("{}╭─{}{}{}─╮{}", border, title_color, fitted_title, border, reset);
            };

            auto print_bottom = [&] {
                printer.print("{}╰{}╯{}", border, std::string(kInnerWidth + 2, '-'), reset);
            };

            auto print_empty = [&] {
                printer.print("{}│{} {} {}│{}", border, reset, std::string(kInnerWidth, ' '),
                              border, reset);
            };

            auto print_row = [&](const std::string& key, const std::string& value) {
                auto text = fit_cell(std::format("{:<14} {}", key, value), kInnerWidth);
                printer.print("{}│{} {} {}│{}", border, reset, text, border, reset);
            };

            auto print_section = [&](const std::string& label) {
                auto text = fit_cell(std::format("[ {} ]", label), kInnerWidth);
                printer.print("{}│{} {}{}{} {}│{}", border, section_color, text, reset, border, "",
                              reset);
            };

            const auto status_value =
                is_mounted ? (std::string(ui::color::kGreen) + "● mounted" + reset)
                           : (std::string(ui::color::kYellow) + "○ unmounted" + reset);

            const auto active_suffix =
                is_active ? (std::string(ui::color::kGreen) + "  [active]" + reset) : "";

            print_top(std::format("Workspace: {}{}", name, active_suffix));
            print_empty();
            print_row("Status", status_value);
            print_row("Mount point", mount_point);
            print_row("Config", config::WorkspaceManager::workspace_dir(name));
            print_empty();

            print_section("Image");
            print_row("Path", image_path.string());
            if (fs::exists(image_path)) {
                auto fsize = fs::file_size(image_path);
                if (fsize >= 1024ULL * 1024 * 1024)
                    print_row("File size",
                              std::format("{:.1f} GB",
                                          static_cast<double>(fsize) / (1024.0 * 1024.0 * 1024.0)));
                else
                    print_row("File size", std::format("{} MB", fsize / (1024 * 1024)));
            }
            else {
                print_row("File size", std::string(ui::color::kRed) + "(image not found)" + reset);
            }
            print_row("Volume size", cfg.image.size);
            print_empty();

            print_section("Build");
            print_row("Architecture", cfg.build.arch);
            print_row("Jobs", std::to_string(cfg.build.jobs));
            print_row("LLVM", cfg.build.llvm ? (std::string(ui::color::kGreen) + "yes" + reset)
                                             : (std::string(ui::color::kRed) + "no" + reset));
            print_row("Cross prefix", cfg.build.cross_compile);
            print_empty();

            print_section("QEMU");
            print_row("Memory", cfg.qemu.memory);
            print_row("SMP", std::to_string(cfg.qemu.smp));
            print_row("GDB port", std::to_string(cfg.qemu.gdb_port));
            print_row("SSH port", std::to_string(cfg.qemu.ssh_port));

            if (is_mounted) {
                print_empty();
                print_section("Directories");
                auto mp = fs::path(mount_point);
                auto check_dir = [&](const std::string& sub) {
                    auto p = mp / sub;
                    std::error_code ec;
                    return fs::is_directory(p, ec);
                };
                for (auto& sub : {"linux", "rootfs", "toolchains", "bootloader", "module", "apps",
                                  "bsp", "plugins", "build", ".elmos", "machine"}) {
                    const auto exists = check_dir(sub);
                    const auto marker = exists ? (std::string(ui::color::kGreen) + "✓" + reset)
                                               : (std::string(ui::color::kYellow) + "○" + reset);
                    print_row("", std::format("{} {}/", marker, sub));
                }
            }

            print_bottom();
        }
    });

    // workspace <name> — positional to select active workspace
    // Must be added after subcommands so CLI11 tries subcommands first
    auto* select_opt = ws->add_option("workspace", "Workspace to activate");

    ws->callback([&app, ws, list_cmd, init_cmd, clean_cmd, exit_cmd, show_cmd, select_opt] {
        // Skip if a subcommand already handled it
        if (list_cmd->parsed() || init_cmd->parsed() || clean_cmd->parsed() || exit_cmd->parsed() ||
            show_cmd->parsed())
            return;

        if (select_opt->count() == 0) {
            // No subcommand, no positional → show help
            std::cout << ws->help();
            return;
        }

        auto name = select_opt->as<std::string>();
        auto result = config::WorkspaceManager::set_active_workspace(name);
        if (!result) {
            // Check if it's a mounted volume without config — auto-register
            std::stop_source ss;
            auto& disk = app.platform().disk_image();
            auto mounted = disk.is_mounted(ss.get_token(), name);
            if (mounted && mounted->mounted) {
                auto& platform = app.platform();
                auto ws_dir = config::WorkspaceManager::workspace_dir(name);
                auto cache_dir = platform.paths().cache_dir();

                // Try both extensions — check workspace dir first, then legacy cache_dir.
                // The workspace may have been created on a different platform (OrbStack/WSL2).
                std::string image_path;
                for (auto* dir : {ws_dir.c_str(), cache_dir.c_str()}) {
                    for (auto* ext : {".sparseimage", ".img"}) {
                        auto candidate = std::string(dir) + "/" + name + ext;
                        if (fs::exists(candidate)) {
                            image_path = candidate;
                            break;
                        }
                    }
                    if (!image_path.empty())
                        break;
                }
                if (image_path.empty()) {
                    // Fallback: use workspace dir with current platform's extension
                    std::string ext = (platform.name() == "darwin") ? ".sparseimage" : ".img";
                    image_path = ws_dir + "/" + name + ext;
                }

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
                    app.printer().error("Cannot register workspace: {}", save_r.error().message());
                    return;
                }
                result = config::WorkspaceManager::set_active_workspace(name);
            }
            if (!result) {
                app.printer().error("{}", result.error().message());
                return;
            }
        }
        app.printer().success("Active workspace: {}", name);
    });
}

}  // namespace elmos::app::commands
