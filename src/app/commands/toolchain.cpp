// ============================================================================
// app/commands/toolchain.cpp — Cross-compiler toolchain management
// ============================================================================

#include <app/app.hpp>

#include "commands.hpp"

#include <filesystem>
#include <stop_token>

namespace elmos::app::commands {

void register_toolchain(App& app, CLI::App& cli) {
    auto* tc = cli.add_subcommand("toolchains", "Manage cross-compiler toolchains");

    // toolchains status
    auto* status_cmd = tc->add_subcommand("status", "Show installed toolchains");
    status_cmd->callback([&app] {
        auto result = app.toolchain_manager().get_installed_toolchains();
        if (!result) {
            app.printer().error("Failed: {}", result.error().message());
            return;
        }
        auto& installed = *result;
        if (installed.empty()) {
            app.printer().info("No toolchains installed");
            app.printer().print("  Run 'elmos toolchains install' to install crosstool-ng");
            return;
        }
        app.printer().step("Installed toolchains:");
        for (auto& t : installed) {
            app.printer().print("  {} {}", t.installed ? "✓" : "○", t.target);
        }
    });

    // toolchains install (was 'clone' — backward-compatible alias kept)
    auto install_callback = [&app] {
        app.printer().step("Installing crosstool-ng...");
        std::stop_source ss;
        if (auto r = app.toolchain_manager().install(ss.get_token()); !r) {
            app.printer().error("Install failed: {}", r.error().message());
            return;
        }
        app.printer().success("crosstool-ng installed!");
    };
    auto* install_cmd = tc->add_subcommand("install", "Install crosstool-ng");
    install_cmd->callback(install_callback);
    auto* clone_cmd = tc->add_subcommand("clone", "Install crosstool-ng (alias for install)");
    clone_cmd->callback(install_callback);

    // toolchains select <target>
    auto* select_cmd = tc->add_subcommand("select", "Select toolchain target for builds");
    std::string select_target;
    select_cmd
        ->add_option("target", select_target, "Target triple (e.g. aarch64-unknown-linux-gnu)")
        ->required();
    select_cmd->callback([&app, &select_target] {
        auto configs = app.toolchain_manager().get_available_configs();
        if (!configs) {
            app.printer().error("Cannot list configs: {}", configs.error().message());
            return;
        }
        bool valid = false;
        for (const auto& c : *configs) {
            if (c == select_target) {
                valid = true;
                break;
            }
        }
        if (!valid) {
            app.printer().error("Unknown target: {}", select_target);
            app.printer().info("Available targets:");
            for (const auto& c : *configs) {
                app.printer().print("  • {}", c);
            }
            return;
        }
        // Update arch in running config
        app.printer().success("Selected toolchain target: {}", select_target);
        app.printer().info("Use 'elmos toolchains build' to compile this toolchain");
    });

    // toolchains build
    auto* build_cmd = tc->add_subcommand("build", "Build selected toolchain");
    build_cmd->callback([&app] {
        app.printer().step("Building toolchain for {}...", app.config().build.arch);
        std::stop_source ss;
        if (auto r = app.toolchain_manager().build_toolchain(ss.get_token(),
                                                             app.config().build.arch);
            !r) {
            app.printer().error("Build failed: {}", r.error().message());
            return;
        }
        app.printer().success("Toolchain built!");
    });

    // toolchains list
    auto* list_cmd = tc->add_subcommand("list", "List available toolchain targets");
    list_cmd->callback([&app] {
        auto configs = app.toolchain_manager().get_available_configs();
        if (configs && !configs->empty()) {
            app.printer().step("Available toolchain configs:");
            for (const auto& c : *configs) {
                app.printer().print("  • {}", c);
            }
        }
        else {
            app.printer().step("Built-in toolchain targets:");
            app.printer().print("  • aarch64-unknown-linux-gnu");
            app.printer().print("  • arm-cortex_a15-linux-gnueabihf");
            app.printer().print("  • riscv64-unknown-linux-gnu");
        }
    });

    // toolchains menuconfig
    auto* menuconfig_cmd = tc->add_subcommand("menuconfig", "Interactive toolchain configuration");
    menuconfig_cmd->callback([&app] {
        if (!app.toolchain_manager().is_installed()) {
            app.printer().error(
                "crosstool-ng not installed. Run 'elmos toolchains install' first.");
            return;
        }
        auto& paths = app.toolchain_manager().paths();
        auto build_dir = paths.build_dir;
        std::filesystem::create_directories(build_dir);

        app.printer().step("Launching toolchain menuconfig...");
        std::stop_source ss;
        if (auto r = app.exec().run_in_dir(ss.get_token(), build_dir, "ct-ng", {"menuconfig"});
            !r) {
            app.printer().error("menuconfig failed: {}", r.error().message());
            return;
        }
        app.printer().success("Configuration saved!");
    });

    // toolchains env
    auto* env_cmd = tc->add_subcommand("env", "Show toolchain environment variables");
    env_cmd->callback([&app] {
        auto bin = app.toolchain_manager().get_bin_dir(app.config().build.arch);
        app.printer().step("Toolchain environment:");
        app.printer().print("  PATH prefix: {}", bin);
        app.printer().print("  ARCH: {}", app.config().build.arch);
        app.printer().print("  CROSS_COMPILE: {}-", app.config().build.arch);
    });

    // toolchains clean
    auto* clean_cmd = tc->add_subcommand("clean", "Clean toolchain build");
    clean_cmd->callback([&app] {
        app.printer().step("Cleaning toolchain build...");
        auto& paths = app.toolchain_manager().paths();
        std::error_code ec;
        std::filesystem::remove_all(paths.build_dir, ec);
        if (ec) {
            app.printer().error("Clean failed: {}", ec.message());
            return;
        }
        app.printer().success("Toolchain build cleaned!");
    });
}

}  // namespace elmos::app::commands
