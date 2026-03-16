// ============================================================================
// app/commands/plugin.cpp — Plugin management commands
// ============================================================================

#include <app/app.hpp>

#include "commands.hpp"

#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <stop_token>

namespace elmos::app::commands {

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

struct PluginInfo {
    std::string name;
    std::string path;
    bool built{false};
};

static auto plugin_base_dir(const config::Config& cfg) -> std::filesystem::path {
    return std::filesystem::path(cfg.image.mount_point) / "plugins";
}

static auto plugin_build_dir(const config::Config& cfg) -> std::filesystem::path {
    return std::filesystem::path(cfg.image.mount_point) / "build" / "plugins";
}

static auto scan_plugins(const config::Config& cfg, const std::string& filter = "")
    -> std::vector<PluginInfo> {
    std::vector<PluginInfo> out;
    auto base = plugin_base_dir(cfg);
    if (!std::filesystem::is_directory(base))
        return out;
    for (auto& entry : std::filesystem::directory_iterator(base)) {
        if (!entry.is_directory())
            continue;
        auto name = entry.path().filename().string();
        if (!filter.empty() && name != filter)
            continue;
        PluginInfo info;
        info.name = name;
        info.path = entry.path().string();
        auto build = plugin_build_dir(cfg) / name;
        info.built = std::filesystem::is_directory(build) && !std::filesystem::is_empty(build);
        out.push_back(std::move(info));
    }
    return out;
}

// ---------------------------------------------------------------------------
// register_plugin
// ---------------------------------------------------------------------------

void register_plugin(App& app, CLI::App& cli) {
    auto* plug = cli.add_subcommand("plugin", "Manage plugins");

    // plugin list
    auto* list_cmd = plug->add_subcommand("list", "List plugins");
    list_cmd->callback([&app] {
        // Show workspace plugins on disk
        auto plugins = scan_plugins(app.config());
        if (plugins.empty()) {
            app.printer().info("No plugins found in workspace");
        }
        else {
            app.printer().step("Workspace plugins:");
            for (auto& p : plugins)
                app.printer().print("  {} {}", p.built ? "✓" : "○", p.name);
        }
        // Also show loaded runtime plugins
        auto loaded = app.plugin_registry().list_plugins();
        if (!loaded.empty()) {
            app.printer().step("Loaded runtime plugins ({}):", loaded.size());
            for (auto* p : loaded) {
                if (p)
                    app.printer().print("  • {} v{} — {}", p->name(), p->version(),
                                        p->description());
            }
        }
    });

    // plugin create <plugin_name>
    auto* create_cmd = plug->add_subcommand("create", "Create a new plugin scaffold");
    auto* create_name = create_cmd->add_option("plugin_name", "Plugin name")->required();
    create_cmd->callback([&app, create_name] {
        auto name = create_name->as<std::string>();
        auto dir = plugin_base_dir(app.config()) / name;
        if (std::filesystem::exists(dir)) {
            app.printer().error("Plugin '{}' already exists at {}", name, dir.string());
            return;
        }
        std::filesystem::create_directories(dir / "src");

        // CMakeLists.txt
        {
            std::ofstream f(dir / "CMakeLists.txt");
            f << "cmake_minimum_required(VERSION 3.20)\n"
              << "project(" << name << " LANGUAGES CXX)\n\n"
              << "set(CMAKE_CXX_STANDARD 23)\n\n"
              << "add_library(" << name << " SHARED src/" << name << ".cpp)\n"
              << "target_include_directories(" << name
              << " PRIVATE ${CMAKE_CURRENT_SOURCE_DIR}/src)\n";
        }
        // Main source file
        {
            std::ofstream f(dir / "src" / (name + ".cpp"));
            f << "// SPDX-License-Identifier: MIT\n"
              << "// Plugin: " << name << "\n\n"
              << "#include <iostream>\n\n"
              << "extern \"C\" {\n\n"
              << "const char* plugin_name()    { return \"" << name << "\"; }\n"
              << "const char* plugin_version() { return \"1.0.0\"; }\n\n"
              << "int plugin_init() {\n"
              << "    std::cout << \"" << name << " plugin initialised\" << std::endl;\n"
              << "    return 0;\n"
              << "}\n\n"
              << "void plugin_cleanup() {\n"
              << "    std::cout << \"" << name << " plugin cleaned up\" << std::endl;\n"
              << "}\n\n"
              << "}  // extern \"C\"\n";
        }
        // Makefile (alternative build path)
        {
            std::ofstream f(dir / "Makefile");
            f << "# Plugin Makefile for " << name << "\n"
              << "PLUGIN_NAME := " << name << "\n"
              << "BUILD_DIR ?= $(CURDIR)/../../build/plugins/$(PLUGIN_NAME)\n\n"
              << "SRC := $(wildcard src/*.cpp)\n"
              << "OBJ := $(patsubst src/%.cpp,$(BUILD_DIR)/%.o,$(SRC))\n"
              << "TARGET := $(BUILD_DIR)/lib$(PLUGIN_NAME).so\n\n"
              << "CXX ?= g++\n"
              << "CXXFLAGS ?= -std=c++23 -fPIC -shared\n\n"
              << ".PHONY: all clean\n\n"
              << "all: $(BUILD_DIR) $(TARGET)\n"
              << "\t@echo \"Plugin $(PLUGIN_NAME) built\"\n\n"
              << "$(BUILD_DIR):\n"
              << "\tmkdir -p $@\n\n"
              << "$(BUILD_DIR)/%.o: src/%.cpp | $(BUILD_DIR)\n"
              << "\t$(CXX) $(CXXFLAGS) -c -o $@ $<\n\n"
              << "$(TARGET): $(OBJ)\n"
              << "\t$(CXX) $(CXXFLAGS) -o $@ $^\n\n"
              << "clean:\n"
              << "\trm -rf $(BUILD_DIR)\n";
        }

        app.printer().success("Plugin '{}' created at {}", name, dir.string());
    });

    // plugin edit <plugin_name>
    auto* edit_cmd = plug->add_subcommand("edit", "Open plugin source in editor");
    auto* edit_name = edit_cmd->add_option("plugin_name", "Plugin name")->required();
    edit_cmd->callback([&app, edit_name] {
        auto name = edit_name->as<std::string>();
        auto dir = plugin_base_dir(app.config()) / name;
        if (!std::filesystem::exists(dir)) {
            app.printer().error("Plugin '{}' not found at {}", name, dir.string());
            return;
        }
        const char* editor = std::getenv("EDITOR");
        if (!editor)
            editor = "vi";
        app.printer().step("Opening plugin '{}' with {}...", name, editor);
        std::stop_source ss;
        if (auto r = app.exec().run(ss.get_token(), editor, {dir.string()}); !r)
            app.printer().error("Editor failed: {}", r.error().message());
    });

    // plugin build [plugin_name] [-j|--jobs]
    auto* build_cmd = plug->add_subcommand("build", "Build plugin(s)");
    auto* build_name = build_cmd->add_option("plugin_name", "Plugin name (blank=all)");
    auto* jobs_opt = build_cmd->add_option("-j,--jobs", "Parallel build jobs")->default_val(0);
    build_cmd->callback([&app, build_name, jobs_opt] {
        auto filter = build_name->count() > 0 ? build_name->as<std::string>() : "";
        auto plugins = scan_plugins(app.config(), filter);
        if (plugins.empty()) {
            app.printer().error("No plugins found{}", filter.empty() ? "" : ": " + filter);
            return;
        }
        auto jobs = jobs_opt->as<int>();
        for (auto& p : plugins) {
            app.printer().step("Building plugin {}...", p.name);
            auto out_dir = plugin_build_dir(app.config()) / p.name;
            std::filesystem::create_directories(out_dir);

            std::vector<std::string> args = {"-C", p.path, "BUILD_DIR=" + out_dir.string()};
            if (jobs > 0)
                args.push_back("-j" + std::to_string(jobs));
            std::stop_source ss;
            if (auto r = app.exec().run(ss.get_token(), "make", args); !r) {
                app.printer().error("Build failed for {}: {}", p.name, r.error().message());
                return;
            }
        }
        app.printer().success("Plugin build complete!");
    });

    // plugin show [plugin_name]
    auto* show_cmd = plug->add_subcommand("show", "Show plugin build result");
    auto* show_name = show_cmd->add_option("plugin_name", "Plugin name (blank=all)");
    show_cmd->callback([&app, show_name] {
        auto filter = show_name->count() > 0 ? show_name->as<std::string>() : "";
        auto plugins = scan_plugins(app.config(), filter);
        if (plugins.empty()) {
            app.printer().info("No plugins found");
            return;
        }
        app.printer().step("Plugin build status:");
        for (auto& p : plugins) {
            app.printer().print("  {} {}", p.built ? "✓" : "○", p.name);
            app.printer().print("    Source: {}", p.path);
            auto out = plugin_build_dir(app.config()) / p.name;
            if (std::filesystem::is_directory(out)) {
                app.printer().print("    Build:  {}", out.string());
                for (auto& f : std::filesystem::directory_iterator(out)) {
                    if (f.is_regular_file()) {
                        auto sz = std::filesystem::file_size(f.path());
                        app.printer().print("      {} ({} KB)", f.path().filename().string(),
                                            sz / 1024);
                    }
                }
            }
        }
    });

    // plugin clean [plugin_name]||all
    auto* clean_cmd = plug->add_subcommand("clean", "Clean plugin build artifacts");
    auto* clean_name = clean_cmd->add_option("plugin_name", "Plugin name or 'all' (default=all)");
    clean_cmd->callback([&app, clean_name] {
        auto name = clean_name->count() > 0 ? clean_name->as<std::string>() : "";
        if (name == "all")
            name = "";
        auto build = plugin_build_dir(app.config());
        if (name.empty()) {
            std::error_code ec;
            std::filesystem::remove_all(build, ec);
            if (ec) {
                app.printer().error("Clean failed: {}", ec.message());
                return;
            }
        }
        else {
            auto target = build / name;
            std::error_code ec;
            std::filesystem::remove_all(target, ec);
            if (ec) {
                app.printer().error("Clean failed for {}: {}", name, ec.message());
                return;
            }
        }
        app.printer().success("Plugin artifacts cleaned!");
    });

    // Show help when 'elmos plugin' is run with no subcommand
    plug->callback([plug] {
        if (plug->get_subcommands().empty()) {
            std::cout << plug->help();
        }
    });
}

}  // namespace elmos::app::commands
