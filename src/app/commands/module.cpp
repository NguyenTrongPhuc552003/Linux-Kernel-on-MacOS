// ============================================================================
// app/commands/module.cpp — Kernel module management commands
// ============================================================================

#include <app/app.hpp>

#include "commands.hpp"

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
            app.printer().print("  • {}", m.name);
        }
    });

    // module build [name]
    auto* build_cmd = mod->add_subcommand("build", "Build kernel module(s)");
    auto* mod_name = build_cmd->add_option("name", "Module name (blank=all)");
    build_cmd->callback([&app, mod_name] {
        std::stop_source ss;
        if (mod_name->count() > 0) {
            auto name = mod_name->as<std::string>();
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

    // module new <name>
    auto* new_cmd = mod->add_subcommand("new", "Create a new kernel module");
    std::string new_name;
    new_cmd->add_option("name", new_name, "Module name")->required();
    new_cmd->callback([&app, &new_name] {
        if (auto r = app.module_builder().create_module(new_name); !r) {
            app.printer().error("Create failed: {}", r.error().message());
            return;
        }
        app.printer().success("Module '{}' created!", new_name);
    });

    // module clean
    auto* clean_cmd = mod->add_subcommand("clean", "Clean module build artifacts");
    clean_cmd->callback([&app] {
        std::stop_source ss;
        if (auto r = app.module_builder().clean(ss.get_token()); !r) {
            app.printer().error("Clean failed: {}", r.error().message());
            return;
        }
        app.printer().success("Modules cleaned!");
    });
}

}  // namespace elmos::app::commands
