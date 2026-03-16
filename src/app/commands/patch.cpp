// ============================================================================
// app/commands/patch.cpp — Patch management commands
// ============================================================================

#include <app/app.hpp>

#include "commands.hpp"

#include <filesystem>
#include <stop_token>

namespace elmos::app::commands {

void register_patch(App& app, CLI::App& cli) {
    auto* patch = cli.add_subcommand("patch", "Apply patches to kernel source");

    // Show help when 'elmos patch' is run with no subcommand
    patch->callback([patch] {
        if (patch->get_subcommands().empty()) {
            std::cout << patch->help();
        }
    });

    // patch list — list available patches organized by version/arch
    auto* list_cmd = patch->add_subcommand("list", "List available patches");
    list_cmd->callback([&app] {
        auto& cfg = app.config();
        if (cfg.paths.patches_dir.empty()) {
            app.printer().info("No patches directory configured");
            return;
        }
        if (!std::filesystem::exists(cfg.paths.patches_dir)) {
            app.printer().info("Patches directory not found: {}", cfg.paths.patches_dir);
            return;
        }

        app.printer().step("Available patches:");
        namespace fs = std::filesystem;
        // Walk patches_dir: version/arch/patch_file structure
        for (const auto& version_entry : fs::directory_iterator(cfg.paths.patches_dir)) {
            if (!version_entry.is_directory())
                continue;
            auto version = version_entry.path().filename().string();
            for (const auto& arch_entry : fs::directory_iterator(version_entry.path())) {
                if (!arch_entry.is_directory())
                    continue;
                auto arch = arch_entry.path().filename().string();
                for (const auto& patch_entry : fs::directory_iterator(arch_entry.path())) {
                    if (patch_entry.is_directory())
                        continue;
                    auto name = patch_entry.path().filename().string();
                    if (name.ends_with(".patch") || name.ends_with(".diff")) {
                        app.printer().print("  {}/{}/{}", version, arch, name);
                    }
                }
            }
        }
    });

    // patch apply <patch_path> [other_patches...]
    auto* apply_cmd = patch->add_subcommand("apply", "Apply patches to kernel source");
    auto* apply_patches =
        apply_cmd->add_option("patches", "Patch paths (relative to patches dir)")->expected(-1);
    apply_cmd->callback([&app, apply_patches] {
        auto& cfg = app.config();
        if (cfg.paths.kernel_dir.empty() || !std::filesystem::exists(cfg.paths.kernel_dir)) {
            app.printer().error("Kernel source not found. Run 'elmos kernel clone' first.");
            return;
        }

        std::stop_source ss;
        if (apply_patches->count() > 0) {
            // Apply specified patches
            auto patch_names = apply_patches->as<std::vector<std::string>>();
            for (const auto& name : patch_names) {
                // Resolve: try as absolute path, then relative to patches_dir
                std::string patch_file = name;
                if (!std::filesystem::exists(patch_file) && !cfg.paths.patches_dir.empty()) {
                    patch_file = (std::filesystem::path(cfg.paths.patches_dir) / name).string();
                }
                if (!std::filesystem::exists(patch_file)) {
                    app.printer().error("Patch not found: {}", name);
                    continue;
                }
                app.printer().step("Applying {}...", name);
                if (auto r = app.patcher().apply_single(ss.get_token(), cfg.paths.kernel_dir,
                                                        patch_file);
                    !r) {
                    app.printer().error("Failed to apply {}: {}", name, r.error().message());
                    continue;
                }
                app.printer().success("Applied: {}", name);
            }
        }
        else {
            // Apply all patches from patches_dir
            if (cfg.paths.patches_dir.empty()) {
                app.printer().info("No patches directory configured");
                return;
            }
            app.printer().step("Applying patches from {}...", cfg.paths.patches_dir);
            if (auto r = app.patcher().apply(ss.get_token(), cfg.paths.kernel_dir,
                                             cfg.paths.patches_dir);
                !r) {
                app.printer().error("Patch failed: {}", r.error().message());
                return;
            }
            app.printer().success("Patches applied!");
        }
    });

    // patch status — show applied/unapplied patches
    auto* status_cmd = patch->add_subcommand("status", "Show patch application status");
    status_cmd->callback([&app] {
        auto& cfg = app.config();
        if (cfg.paths.patches_dir.empty() || !std::filesystem::exists(cfg.paths.patches_dir)) {
            app.printer().info("No patches directory configured");
            return;
        }
        if (cfg.paths.kernel_dir.empty() || !std::filesystem::exists(cfg.paths.kernel_dir)) {
            app.printer().error("Kernel source not found");
            return;
        }

        auto patches = app.patcher().list_patches(cfg.paths.patches_dir);
        if (!patches || patches->empty()) {
            app.printer().info("No patches found");
            return;
        }

        app.printer().step("Patch status:");
        std::stop_source ss;
        for (auto& p : *patches) {
            bool applied = app.patcher().check_applied(ss.get_token(), cfg.paths.kernel_dir,
                                                       p.path);
            app.printer().print("  {} {}", applied ? "✓" : "○", p.name);
        }
    });

    // patch show <patch_path> — show patch content/info
    auto* show_cmd = patch->add_subcommand("show", "Show patch information");
    auto* show_name = show_cmd->add_option("patch", "Patch path")->required();
    show_cmd->callback([&app, show_name] {
        auto& cfg = app.config();
        auto name = show_name->as<std::string>();

        std::string patch_file = name;
        if (!std::filesystem::exists(patch_file) && !cfg.paths.patches_dir.empty()) {
            patch_file = (std::filesystem::path(cfg.paths.patches_dir) / name).string();
        }
        if (!std::filesystem::exists(patch_file)) {
            app.printer().error("Patch not found: {}", name);
            return;
        }

        app.printer().step("Patch: {}", name);
        app.printer().print("  File: {}", patch_file);
        auto size = std::filesystem::file_size(patch_file);
        app.printer().print("  Size: {} bytes", size);

        // Check if applied
        if (!cfg.paths.kernel_dir.empty() && std::filesystem::exists(cfg.paths.kernel_dir)) {
            std::stop_source ss;
            bool applied = app.patcher().check_applied(ss.get_token(), cfg.paths.kernel_dir,
                                                       patch_file);
            app.printer().print("  Status: {}", applied ? "applied" : "not applied");
        }

        // Show stat (summary) using git apply --stat
        std::stop_source ss;
        if (auto r = app.exec().run(ss.get_token(), "git", {"apply", "--stat", patch_file}); !r) {
            app.printer().warn("Could not read patch stat");
        }
    });

    // patch remove <patch_path>||all — reverse-apply patches
    auto* remove_cmd = patch->add_subcommand("remove", "Remove applied patches (reverse-apply)");
    auto* remove_name = remove_cmd->add_option("patch", "Patch path or 'all'")->required();
    remove_cmd->callback([&app, remove_name] {
        auto& cfg = app.config();
        if (cfg.paths.kernel_dir.empty() || !std::filesystem::exists(cfg.paths.kernel_dir)) {
            app.printer().error("Kernel source not found");
            return;
        }

        auto name = remove_name->as<std::string>();
        std::stop_source ss;

        if (name == "all") {
            // Remove all applied patches
            if (cfg.paths.patches_dir.empty()) {
                app.printer().info("No patches directory configured");
                return;
            }
            auto patches = app.patcher().list_patches(cfg.paths.patches_dir);
            if (!patches || patches->empty()) {
                app.printer().info("No patches to remove");
                return;
            }
            for (auto& p : *patches) {
                if (!app.patcher().check_applied(ss.get_token(), cfg.paths.kernel_dir, p.path))
                    continue;
                app.printer().step("Removing {}...", p.name);
                if (auto r = app.exec().run_in_dir(ss.get_token(), cfg.paths.kernel_dir, "git",
                                                   {"apply", "-R", p.path});
                    !r) {
                    app.printer().error("Failed to remove {}: {}", p.name, r.error().message());
                    continue;
                }
                app.printer().success("Removed: {}", p.name);
            }
        }
        else {
            // Remove specific patch
            std::string patch_file = name;
            if (!std::filesystem::exists(patch_file) && !cfg.paths.patches_dir.empty()) {
                patch_file = (std::filesystem::path(cfg.paths.patches_dir) / name).string();
            }
            if (!std::filesystem::exists(patch_file)) {
                app.printer().error("Patch not found: {}", name);
                return;
            }
            app.printer().step("Removing {}...", name);
            if (auto r = app.exec().run_in_dir(ss.get_token(), cfg.paths.kernel_dir, "git",
                                               {"apply", "-R", patch_file});
                !r) {
                app.printer().error("Failed to remove: {}", r.error().message());
                return;
            }
            app.printer().success("Patch removed!");
        }
    });
}

}  // namespace elmos::app::commands
