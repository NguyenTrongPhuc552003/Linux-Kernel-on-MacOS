// ============================================================================
// app/commands/app.cpp — Userspace app management commands
// ============================================================================

#include <app/app.hpp>

#include "commands.hpp"

#include <cstdlib>
#include <filesystem>
#include <stop_token>

namespace elmos::app::commands {

void register_apps(App& app, CLI::App& cli) {
    auto* apps = cli.add_subcommand("app", "Manage userspace apps");

    // app list
    auto* list_cmd = apps->add_subcommand("list", "List userspace apps");
    list_cmd->callback([&app] {
        auto result = app.app_builder().get_apps();
        if (!result) {
            app.printer().error("Failed to list apps: {}", result.error().message());
            return;
        }
        auto& list = *result;
        if (list.empty()) {
            app.printer().info("No apps found");
            return;
        }
        app.printer().step("Userspace apps:");
        for (auto& a : list) {
            app.printer().print("  {} {}", a.built ? "✓" : "○", a.name);
        }
    });

    // app create <name> (renamed from "new")
    auto* create_cmd = apps->add_subcommand("create", "Create a new userspace app");
    auto* create_name = create_cmd->add_option("name", "App name")->required();
    create_cmd->callback([&app, create_name] {
        auto name = create_name->as<std::string>();
        if (auto r = app.app_builder().create_app(name); !r) {
            app.printer().error("Create failed: {}", r.error().message());
            return;
        }
        app.printer().success("App '{}' created!", name);
    });

    // app edit <name>
    auto* edit_cmd = apps->add_subcommand("edit", "Open app source in editor");
    auto* edit_name = edit_cmd->add_option("name", "App name")->required();
    edit_cmd->callback([&app, edit_name] {
        auto name = edit_name->as<std::string>();
        auto app_dir = std::filesystem::path(app.config().paths.apps_dir) / name;
        if (!std::filesystem::exists(app_dir)) {
            app.printer().error("App '{}' not found at {}", name, app_dir.string());
            return;
        }
        const char* editor = std::getenv("EDITOR");
        if (!editor)
            editor = "vi";
        app.printer().step("Opening app '{}' with {}...", name, editor);
        std::stop_source ss;
        if (auto r = app.exec().run(ss.get_token(), editor, {app_dir.string()}); !r) {
            app.printer().error("Editor failed: {}", r.error().message());
        }
    });

    // app build [name] [-j|--jobs]
    auto* build_cmd = apps->add_subcommand("build", "Build userspace app(s)");
    auto* build_name = build_cmd->add_option("name", "App name (blank=all)");
    auto* jobs_opt = build_cmd->add_option("-j,--jobs", "Parallel build jobs")->default_val(0);
    build_cmd->callback([&app, build_name, jobs_opt] {
        std::stop_source ss;
        (void)jobs_opt;
        if (build_name->count() > 0) {
            auto name = build_name->as<std::string>();
            app.printer().step("Building app {}...", name);
            if (auto r = app.app_builder().build(ss.get_token(), name); !r) {
                app.printer().error("Build failed: {}", r.error().message());
                return;
            }
        }
        else {
            app.printer().step("Building all apps...");
            if (auto r = app.app_builder().build(ss.get_token()); !r) {
                app.printer().error("Build failed: {}", r.error().message());
                return;
            }
        }
        app.printer().success("App build complete!");
    });

    // app show [name]
    auto* show_cmd = apps->add_subcommand("show", "Show app build result");
    auto* show_name = show_cmd->add_option("name", "App name (blank=all)");
    show_cmd->callback([&app, show_name] {
        auto result =
            app.app_builder().get_apps(show_name->count() > 0 ? show_name->as<std::string>() : "");
        if (!result) {
            app.printer().error("Failed: {}", result.error().message());
            return;
        }
        if (result->empty()) {
            app.printer().info("No apps found");
            return;
        }
        app.printer().step("App build status:");
        for (auto& a : *result) {
            app.printer().print("  {} {}", a.built ? "✓" : "○", a.name);
            app.printer().print("    Path: {}", a.path);
            // Check for built executable
            auto exe_path = std::filesystem::path(app.config().image.mount_point) / "build" /
                            "apps" / a.name;
            if (std::filesystem::exists(exe_path)) {
                app.printer().print("    Built: {}", exe_path.string());
            }
        }
    });

    // app load [name]
    auto* load_cmd = apps->add_subcommand("load", "Load app into rootfs");
    auto* load_name = load_cmd->add_option("name", "App name (blank=all)");
    load_cmd->callback([&app, load_name] {
        auto name = load_name->count() > 0 ? load_name->as<std::string>() : "";
        auto result = app.app_builder().get_apps(name);
        if (!result || result->empty()) {
            app.printer().error("No apps found{}", name.empty() ? "" : ": " + name);
            return;
        }
        for (auto& a : *result) {
            auto exe_path = std::filesystem::path(app.config().image.mount_point) / "build" /
                            "apps" / a.name;
            if (!std::filesystem::exists(exe_path)) {
                app.printer().warn("App '{}' not built — build first", a.name);
                continue;
            }
            // Copy to rootfs /usr/local/bin/
            auto rootfs_bin_dir = std::filesystem::path(app.config().paths.rootfs_dir) / "usr" /
                                  "local" / "bin";
            std::filesystem::create_directories(rootfs_bin_dir);
            auto dest = rootfs_bin_dir / a.name;
            std::error_code ec;
            std::filesystem::copy_file(exe_path, dest,
                                       std::filesystem::copy_options::overwrite_existing, ec);
            if (ec) {
                app.printer().error("Failed to copy {}: {}", a.name, ec.message());
                continue;
            }
            app.printer().success("App '{}' loaded into rootfs at /usr/local/bin/{}", a.name,
                                  a.name);
        }
    });

    // app clean [name]||all
    auto* clean_cmd = apps->add_subcommand("clean", "Clean app build artifacts");
    auto* clean_name = clean_cmd->add_option("name", "App name or 'all' (default=all)");
    clean_cmd->callback([&app, clean_name] {
        std::stop_source ss;
        auto name = clean_name->count() > 0 ? clean_name->as<std::string>() : "";
        if (name == "all")
            name = "";
        if (auto r = app.app_builder().clean(ss.get_token(), name); !r) {
            app.printer().error("Clean failed: {}", r.error().message());
            return;
        }
        app.printer().success("Apps cleaned!");
    });

    // Show help when 'elmos app' is run with no subcommand
    apps->callback([apps] {
        if (apps->get_subcommands().empty()) {
            std::cout << apps->help();
        }
    });
}

}  // namespace elmos::app::commands
