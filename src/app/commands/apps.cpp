// ============================================================================
// app/commands/apps.cpp — Userspace app management commands
// ============================================================================

#include <app/app.hpp>

#include "commands.hpp"

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
            app.printer().print("  • {}", a.name);
        }
    });

    // app build [name]
    auto* build_cmd = apps->add_subcommand("build", "Build userspace app(s)");
    auto* app_name = build_cmd->add_option("name", "App name (blank=all)");
    build_cmd->callback([&app, app_name] {
        std::stop_source ss;
        if (app_name->count() > 0) {
            auto name = app_name->as<std::string>();
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

    // app new <name>
    auto* new_cmd = apps->add_subcommand("new", "Create a new userspace app");
    std::string new_name;
    new_cmd->add_option("name", new_name, "App name")->required();
    new_cmd->callback([&app, &new_name] {
        if (auto r = app.app_builder().create_app(new_name); !r) {
            app.printer().error("Create failed: {}", r.error().message());
            return;
        }
        app.printer().success("App '{}' created!", new_name);
    });

    // app clean
    auto* clean_cmd = apps->add_subcommand("clean", "Clean app build artifacts");
    clean_cmd->callback([&app] {
        std::stop_source ss;
        if (auto r = app.app_builder().clean(ss.get_token()); !r) {
            app.printer().error("Clean failed: {}", r.error().message());
            return;
        }
        app.printer().success("Apps cleaned!");
    });
}

}  // namespace elmos::app::commands
