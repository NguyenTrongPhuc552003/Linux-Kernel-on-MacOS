// ============================================================================
// app/commands/bsp.cpp — Board Support Package commands
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

struct BspInfo {
    std::string name;
    std::string path;
    bool built{false};
};

static auto bsp_base_dir(const config::Config& cfg) -> std::filesystem::path {
    return std::filesystem::path(cfg.image.mount_point) / "bsp";
}

static auto bsp_build_dir(const config::Config& cfg) -> std::filesystem::path {
    return std::filesystem::path(cfg.image.mount_point) / "build" / "bsp";
}

static auto scan_bsps(const config::Config& cfg, const std::string& filter = "")
    -> std::vector<BspInfo> {
    std::vector<BspInfo> out;
    auto base = bsp_base_dir(cfg);
    if (!std::filesystem::is_directory(base))
        return out;
    for (auto& entry : std::filesystem::directory_iterator(base)) {
        if (!entry.is_directory())
            continue;
        auto name = entry.path().filename().string();
        if (!filter.empty() && name != filter)
            continue;
        BspInfo info;
        info.name = name;
        info.path = entry.path().string();
        auto build = bsp_build_dir(cfg) / name;
        info.built = std::filesystem::is_directory(build) && !std::filesystem::is_empty(build);
        out.push_back(std::move(info));
    }
    return out;
}

// ---------------------------------------------------------------------------
// register_bsp
// ---------------------------------------------------------------------------

void register_bsp(App& app, CLI::App& cli) {
    auto* bsp = cli.add_subcommand("bsp", "Board support package management");

    // bsp list
    auto* list_cmd = bsp->add_subcommand("list", "List available BSPs");
    list_cmd->callback([&app] {
        auto bsps = scan_bsps(app.config());
        if (bsps.empty()) {
            app.printer().info("No BSPs found");
            return;
        }
        app.printer().step("Board support packages:");
        for (auto& b : bsps)
            app.printer().print("  {} {}", b.built ? "✓" : "○", b.name);
    });

    // bsp create <bsp_name>
    auto* create_cmd = bsp->add_subcommand("create", "Create a new BSP scaffold");
    auto* create_name = create_cmd->add_option("bsp_name", "BSP name")->required();
    create_cmd->callback([&app, create_name] {
        auto name = create_name->as<std::string>();
        auto dir = bsp_base_dir(app.config()) / name;
        if (std::filesystem::exists(dir)) {
            app.printer().error("BSP '{}' already exists at {}", name, dir.string());
            return;
        }
        std::filesystem::create_directories(dir / "dts");
        std::filesystem::create_directories(dir / "scripts");

        // config.yaml
        {
            std::ofstream f(dir / "config.yaml");
            f << "# BSP configuration for " << name << "\n"
              << "name: " << name << "\n"
              << "arch: arm64\n"
              << "description: Board support package for " << name << "\n";
        }
        // Makefile
        {
            std::ofstream f(dir / "Makefile");
            f << "# BSP Makefile for " << name << "\n"
              << "BSP_NAME := " << name << "\n"
              << "BUILD_DIR ?= $(CURDIR)/../../build/bsp/$(BSP_NAME)\n\n"
              << "DTS_SRC := $(wildcard dts/*.dts)\n"
              << "DTB_OUT := $(patsubst dts/%.dts,$(BUILD_DIR)/%.dtb,$(DTS_SRC))\n\n"
              << ".PHONY: all clean\n\n"
              << "all: $(BUILD_DIR) $(DTB_OUT)\n"
              << "\t@echo \"BSP $(BSP_NAME) built\"\n\n"
              << "$(BUILD_DIR):\n"
              << "\tmkdir -p $@\n\n"
              << "$(BUILD_DIR)/%.dtb: dts/%.dts | $(BUILD_DIR)\n"
              << "\tdtc -I dts -O dtb -o $@ $<\n\n"
              << "clean:\n"
              << "\trm -rf $(BUILD_DIR)\n";
        }
        // starter device tree overlay
        {
            std::ofstream f(dir / "dts" / (name + "-overlay.dts"));
            f << "// SPDX-License-Identifier: GPL-2.0\n"
              << "/dts-v1/;\n"
              << "/plugin/;\n\n"
              << "/ {\n"
              << "    compatible = \"vendor," << name << "\";\n\n"
              << "    fragment@0 {\n"
              << "        target-path = \"/\";\n"
              << "        __overlay__ {\n"
              << "            /* Add board-specific nodes here */\n"
              << "        };\n"
              << "    };\n"
              << "};\n";
        }
        // board init script
        {
            std::ofstream f(dir / "scripts" / "init.sh");
            f << "#!/bin/sh\n"
              << "# Board-specific initialisation script for " << name << "\n"
              << "echo \"" << name << " BSP initialised\"\n";
        }

        app.printer().success("BSP '{}' created at {}", name, dir.string());
    });

    // bsp edit <bsp_name>
    auto* edit_cmd = bsp->add_subcommand("edit", "Open BSP source in editor");
    auto* edit_name = edit_cmd->add_option("bsp_name", "BSP name")->required();
    edit_cmd->callback([&app, edit_name] {
        auto name = edit_name->as<std::string>();
        auto dir = bsp_base_dir(app.config()) / name;
        if (!std::filesystem::exists(dir)) {
            app.printer().error("BSP '{}' not found at {}", name, dir.string());
            return;
        }
        const char* editor = std::getenv("EDITOR");
        if (!editor)
            editor = "vi";
        app.printer().step("Opening BSP '{}' with {}...", name, editor);
        std::stop_source ss;
        if (auto r = app.exec().run(ss.get_token(), editor, {dir.string()}); !r)
            app.printer().error("Editor failed: {}", r.error().message());
    });

    // bsp build [bsp_name] [-j|--jobs]
    auto* build_cmd = bsp->add_subcommand("build", "Build BSP(s)");
    auto* build_name = build_cmd->add_option("bsp_name", "BSP name (blank=all)");
    auto* jobs_opt = build_cmd->add_option("-j,--jobs", "Parallel build jobs")->default_val(0);
    build_cmd->callback([&app, build_name, jobs_opt] {
        auto filter = build_name->count() > 0 ? build_name->as<std::string>() : "";
        auto bsps = scan_bsps(app.config(), filter);
        if (bsps.empty()) {
            app.printer().error("No BSPs found{}", filter.empty() ? "" : ": " + filter);
            return;
        }
        auto jobs = jobs_opt->as<int>();
        for (auto& b : bsps) {
            app.printer().step("Building BSP {}...", b.name);
            auto out_dir = bsp_build_dir(app.config()) / b.name;
            std::filesystem::create_directories(out_dir);

            std::vector<std::string> args = {"-C", b.path, "BUILD_DIR=" + out_dir.string()};
            if (jobs > 0)
                args.push_back("-j" + std::to_string(jobs));
            std::stop_source ss;
            if (auto r = app.exec().run(ss.get_token(), "make", args); !r) {
                app.printer().error("Build failed for {}: {}", b.name, r.error().message());
                return;
            }
        }
        app.printer().success("BSP build complete!");
    });

    // bsp show [bsp_name]
    auto* show_cmd = bsp->add_subcommand("show", "Show BSP build result");
    auto* show_name = show_cmd->add_option("bsp_name", "BSP name (blank=all)");
    show_cmd->callback([&app, show_name] {
        auto filter = show_name->count() > 0 ? show_name->as<std::string>() : "";
        auto bsps = scan_bsps(app.config(), filter);
        if (bsps.empty()) {
            app.printer().info("No BSPs found");
            return;
        }
        app.printer().step("BSP build status:");
        for (auto& b : bsps) {
            app.printer().print("  {} {}", b.built ? "✓" : "○", b.name);
            app.printer().print("    Source: {}", b.path);
            auto out = bsp_build_dir(app.config()) / b.name;
            if (std::filesystem::is_directory(out)) {
                app.printer().print("    Build:  {}", out.string());
                // list built artefacts
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

    // bsp clean [bsp_name]||all
    auto* clean_cmd = bsp->add_subcommand("clean", "Clean BSP build artifacts");
    auto* clean_name = clean_cmd->add_option("bsp_name", "BSP name or 'all' (default=all)");
    clean_cmd->callback([&app, clean_name] {
        auto name = clean_name->count() > 0 ? clean_name->as<std::string>() : "";
        if (name == "all")
            name = "";
        auto build = bsp_build_dir(app.config());
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
        app.printer().success("BSP artifacts cleaned!");
    });

    // Show help when 'elmos bsp' is run with no subcommand
    bsp->callback([bsp] {
        if (bsp->get_subcommands().empty()) {
            std::cout << bsp->help();
        }
    });
}

}  // namespace elmos::app::commands
