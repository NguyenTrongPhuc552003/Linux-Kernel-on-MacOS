// ============================================================================
// app/commands/module.cpp — Kernel module management commands
// ============================================================================

#include <app/app.hpp>

#include "commands.hpp"

#include <cstdlib>
#include <filesystem>
#include <stop_token>

namespace elmos::app::commands {

void register_module(App& app, CLI::App& cli) {
    auto* mod = cli.add_subcommand("module", "Manage kernel modules");

    // module list
    auto* list_cmd = mod->add_subcommand("list", "List kernel modules");
    list_cmd->callback([&app] {
        auto result = app.module_builder().get_modules();
        if (!result) {
            app.printer().error("Failed to list modules: {}", result.error().message());
            return;
        }
        auto& modules = *result;
        if (modules.empty()) {
            app.printer().info("No modules found");
            return;
        }
        app.printer().step("Kernel modules:");
        for (auto& m : modules) {
            app.printer().print("  {} {} — {}", m.built ? "✓" : "○", m.name,
                                m.description.empty() ? m.path : m.description);
        }
    });

    // module create <name> (renamed from "new")
    auto* create_cmd = mod->add_subcommand("create", "Create a new kernel module");
    auto* create_name = create_cmd->add_option("name", "Module name")->required();
    create_cmd->callback([&app, create_name] {
        auto name = create_name->as<std::string>();
        if (auto r = app.module_builder().create_module(name); !r) {
            app.printer().error("Create failed: {}", r.error().message());
            return;
        }
        app.printer().success("Module '{}' created!", name);
    });

    // module edit <name>
    auto* edit_cmd = mod->add_subcommand("edit", "Open module source in editor");
    auto* edit_name = edit_cmd->add_option("name", "Module name")->required();
    edit_cmd->callback([&app, edit_name] {
        auto name = edit_name->as<std::string>();
        auto module_dir = std::filesystem::path(app.config().paths.modules_dir) / name;
        if (!std::filesystem::exists(module_dir)) {
            app.printer().error("Module '{}' not found at {}", name, module_dir.string());
            return;
        }
        // Detect editor: $EDITOR, then fallback
        const char* editor = std::getenv("EDITOR");
        if (!editor)
            editor = "vi";
        app.printer().step("Opening module '{}' with {}...", name, editor);
        std::stop_source ss;
        if (auto r = app.exec().run(ss.get_token(), editor, {module_dir.string()}); !r) {
            app.printer().error("Editor failed: {}", r.error().message());
        }
    });

    // module build [name] [-j|--jobs]
    auto* build_cmd = mod->add_subcommand("build", "Build kernel module(s)");
    auto* build_name = build_cmd->add_option("name", "Module name (blank=all)");
    auto* jobs_opt = build_cmd->add_option("-j,--jobs", "Parallel build jobs")->default_val(0);
    build_cmd->callback([&app, build_name, jobs_opt] {
        std::stop_source ss;
        // jobs_opt available for future use when domain supports it
        (void)jobs_opt;
        if (build_name->count() > 0) {
            auto name = build_name->as<std::string>();
            app.printer().step("Building module {}...", name);
            if (auto r = app.module_builder().build(ss.get_token(), name); !r) {
                app.printer().error("Build failed: {}", r.error().message());
                return;
            }
        }
        else {
            app.printer().step("Building all modules...");
            if (auto r = app.module_builder().build(ss.get_token()); !r) {
                app.printer().error("Build failed: {}", r.error().message());
                return;
            }
        }
        app.printer().success("Module build complete!");
    });

    // module show [name]
    auto* show_cmd = mod->add_subcommand("show", "Show module build result");
    auto* show_name = show_cmd->add_option("name", "Module name (blank=all)");
    show_cmd->callback([&app, show_name] {
        auto result = app.module_builder().get_modules(
            show_name->count() > 0 ? show_name->as<std::string>() : "");
        if (!result) {
            app.printer().error("Failed: {}", result.error().message());
            return;
        }
        if (result->empty()) {
            app.printer().info("No modules found");
            return;
        }
        app.printer().step("Module build status:");
        for (auto& m : *result) {
            app.printer().print("  {} {}", m.built ? "✓" : "○", m.name);
            app.printer().print("    Path: {}", m.path);
            if (!m.description.empty())
                app.printer().print("    Desc: {}", m.description);
            // Check for .ko file
            auto ko_path = std::filesystem::path(m.path) / (m.name + ".ko");
            if (std::filesystem::exists(ko_path)) {
                auto size = std::filesystem::file_size(ko_path);
                app.printer().print("    .ko:  {} ({} KB)", ko_path.string(), size / 1024);
            }
        }
    });

    // module load [name]
    auto* load_cmd = mod->add_subcommand("load", "Mark module for loading at QEMU boot");
    auto* load_name = load_cmd->add_option("name", "Module name (blank=all)");
    load_cmd->callback([&app, load_name] {
        auto name = load_name->count() > 0 ? load_name->as<std::string>() : "";
        auto result = app.module_builder().get_modules(name);
        if (!result || result->empty()) {
            app.printer().error("No modules found{}", name.empty() ? "" : ": " + name);
            return;
        }
        for (auto& m : *result) {
            auto ko_path = std::filesystem::path(m.path) / (m.name + ".ko");
            if (!std::filesystem::exists(ko_path)) {
                app.printer().warn("Module '{}' not built — build first", m.name);
                continue;
            }
            // Copy .ko to rootfs modules directory
            auto rootfs_mod_dir = std::filesystem::path(app.config().paths.rootfs_dir) / "lib" /
                                  "modules";
            std::filesystem::create_directories(rootfs_mod_dir);
            auto dest = rootfs_mod_dir / (m.name + ".ko");
            std::error_code ec;
            std::filesystem::copy_file(ko_path, dest,
                                       std::filesystem::copy_options::overwrite_existing, ec);
            if (ec) {
                app.printer().error("Failed to copy {}: {}", m.name, ec.message());
                continue;
            }
            app.printer().success("Module '{}' ready for loading", m.name);
        }
    });

    // module clean [name]||all
    auto* clean_cmd = mod->add_subcommand("clean", "Clean module build artifacts");
    auto* clean_name = clean_cmd->add_option("name", "Module name or 'all' (default=all)");
    clean_cmd->callback([&app, clean_name] {
        std::stop_source ss;
        auto name = clean_name->count() > 0 ? clean_name->as<std::string>() : "";
        if (name == "all")
            name = "";
        if (auto r = app.module_builder().clean(ss.get_token(), name); !r) {
            app.printer().error("Clean failed: {}", r.error().message());
            return;
        }
        app.printer().success("Modules cleaned!");
    });

    // Show help when 'elmos module' is run with no subcommand
    mod->callback([mod] {
        if (mod->get_subcommands().empty()) {
            std::cout << mod->help();
        }
    });
}

}  // namespace elmos::app::commands
