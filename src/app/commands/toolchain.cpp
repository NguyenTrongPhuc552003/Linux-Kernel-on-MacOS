// ============================================================================
// app/commands/toolchain.cpp — Cross-compiler toolchain management
// ============================================================================

#include <app/app.hpp>
#include <config/arch.hpp>
#include <ui/printer.hpp>

#include "commands.hpp"

#include <algorithm>
#include <filesystem>
#include <format>
#include <stop_token>

namespace elmos::app::commands {

void register_toolchain(App& app, CLI::App& cli) {
    auto* tc = cli.add_subcommand("toolchain", "Manage cross-compiler toolchain");

    // toolchain status
    auto* status_cmd = tc->add_subcommand("status", "Show installed toolchains");
    status_cmd->callback([&app] {
        // ct-ng installation status
        auto ct_ng_installed = app.toolchain_manager().is_installed();
        app.printer().step("crosstool-ng: {}", ct_ng_installed ? "installed" : "not installed");
        if (!ct_ng_installed) {
            app.printer().print("  Run 'elmos toolchain clone' to install crosstool-ng");
            return;
        }
        app.printer().print("  Binary: {}", app.toolchain_manager().ct_ng_bin());

        // Available configs
        auto configs = app.toolchain_manager().get_available_configs();
        if (configs && !configs->empty()) {
            app.printer().step("Available configs:");
            for (const auto& c : *configs) {
                app.printer().print("  • {}", c);
            }
        }

        // Built toolchains
        auto result = app.toolchain_manager().get_installed_toolchains();
        if (!result) {
            app.printer().error("Failed: {}", result.error().message());
            return;
        }
        auto& installed = *result;
        if (installed.empty()) {
            app.printer().info("No toolchains built yet");
            app.printer().print("  Run 'elmos toolchain build' to compile a toolchain");
            return;
        }
        app.printer().step("Built toolchains:");
        for (auto& t : installed) {
            app.printer().print("  {} {}", t.installed ? "✓" : "○", t.target);
        }
    });

    // toolchain clone (install crosstool-ng)
    auto* clone_cmd = tc->add_subcommand("clone", "Install crosstool-ng");
    clone_cmd->callback([&app] {
        if (app.toolchain_manager().is_installed()) {
            app.printer().success("crosstool-ng already installed at {}",
                                  app.toolchain_manager().ct_ng_bin());
            return;
        }
        app.printer().step("Installing crosstool-ng...");
        std::stop_source ss;
        if (auto r = app.toolchain_manager().install(ss.get_token()); !r) {
            app.printer().error("Install failed: {}", r.error().message());
            return;
        }
        app.printer().success("crosstool-ng installed!");
    });

    // toolchain build
    auto* build_cmd = tc->add_subcommand("build", "Build selected toolchain");
    build_cmd->callback([&app] {
        if (!app.toolchain_manager().is_installed()) {
            app.printer().error("crosstool-ng not installed. Run 'elmos toolchain clone' first.");
            return;
        }

        auto& arch = app.config().build.arch;
        auto resolved = app.toolchain_manager().resolve_target(arch);
        if (!resolved) {
            app.printer().error("{}", resolved.error().message());
            return;
        }

        app.printer().step("Building toolchain '{}' for arch '{}'...", *resolved, arch);
        std::stop_source ss;
        if (auto r = app.toolchain_manager().build_toolchain(ss.get_token(), arch); !r) {
            app.printer().error("Build failed: {}", r.error().message());
            return;
        }
        app.printer().success("Toolchain '{}' built!", *resolved);
    });

    // toolchain list
    auto* list_cmd = tc->add_subcommand("list", "List available toolchain targets");
    auto* list_all_flag =
        list_cmd->add_flag("-a,--all", "List all available toolchains from crosstool-ng configs");
    list_cmd->callback([&app, list_all_flag] {
        auto normalize_config_arch = [](const std::string& ct_arch,
                                        const std::string& ct_arch_bitness) -> std::string {
            if (ct_arch == "aarch64") {
                return "arm64";
            }
            if (ct_arch == "arm" && ct_arch_bitness == "64") {
                return "arm64";
            }
            return ct_arch.empty() ? "unknown" : ct_arch;
        };

        if (list_all_flag->count() > 0) {
            auto configs = app.toolchain_manager().get_available_configs();
            if (!configs) {
                app.printer().error("Cannot list toolchain targets: {}", configs.error().message());
                return;
            }

            std::ranges::sort(*configs);
            app.printer().step("All available toolchain targets from crosstool-ng configs:");

            if (configs->empty()) {
                app.printer().warn("No toolchain configs found in '{}'",
                                   app.toolchain_manager().paths().config_dir);
                return;
            }

            for (const auto& target : *configs) {
                auto details = app.toolchain_manager().get_config_details(target);
                std::string arch_label = "unknown";
                if (details) {
                    arch_label = normalize_config_arch(details->ct_arch, details->ct_arch_bitness);
                }

                app.printer().print("  {}○{} {} {}[arch: {}]{}", ui::color::kCyan,
                                    ui::color::kReset, target, ui::color::kWhite, arch_label,
                                    ui::color::kReset);
            }

            app.printer().print("");
            app.printer().step("Default target by architecture:");
            for (const auto& [arch_name, _] : config::architectures()) {
                auto default_target = app.toolchain_manager().resolve_target(arch_name);
                if (!default_target) {
                    app.printer().print("  {}{}{} (default: unresolved)", ui::color::kYellow,
                                        arch_name, ui::color::kReset);
                    continue;
                }

                app.printer().print("  {}{}{} (default: {}{}{}{})", ui::color::kGreen, arch_name,
                                    ui::color::kReset, ui::color::kBold, ui::color::kCyan,
                                    *default_target, ui::color::kReset);
            }
            return;
        }

        const auto& arch = app.config().build.arch;
        auto targets = app.toolchain_manager().get_targets_for_arch(arch);
        if (!targets) {
            app.printer().error("Cannot list toolchain targets: {}", targets.error().message());
            return;
        }

        auto resolved = app.toolchain_manager().resolve_target(arch);
        if (resolved) {
            app.printer().step("Available toolchain targets for arch '{}' (default: {}):", arch,
                               *resolved);
        }
        else {
            app.printer().step("Available toolchain targets for arch '{}':", arch);
        }

        if (targets->empty()) {
            app.printer().warn("No matching toolchain target found in '{}'", arch);
            app.printer().print("  Check {}/ for cloned crosstool-ng configs",
                                app.toolchain_manager().paths().config_dir);
            return;
        }

        const auto active_target = resolved ? *resolved : std::string{};
        for (const auto& target : *targets) {
            const bool is_active = (!active_target.empty() && target == active_target);
            const auto marker = is_active ? "●" : "○";
            const auto color = is_active ? ui::color::kGreen : ui::color::kCyan;

            app.printer().print("  {}{}{}{} {}", color, marker, ui::color::kReset,
                                is_active ? " (active)" : "", target);
        }
    });

    // toolchain menuconfig
    auto* menuconfig_cmd = tc->add_subcommand("menuconfig", "Interactive toolchain configuration");
    menuconfig_cmd->callback([&app] {
        if (!app.toolchain_manager().is_installed()) {
            app.printer().error("crosstool-ng not installed. Run 'elmos toolchain clone' first.");
            return;
        }
        auto& paths = app.toolchain_manager().paths();
        auto build_dir = paths.build_dir;
        std::filesystem::create_directories(build_dir);

        app.printer().step("Launching toolchain menuconfig...");
        std::stop_source ss;
        auto ct_ng = app.toolchain_manager().ct_ng_bin();
        if (auto r = app.exec().run_in_dir(ss.get_token(), build_dir, ct_ng, {"menuconfig"}); !r) {
            app.printer().error("menuconfig failed: {}", r.error().message());
            return;
        }
        app.printer().success("Configuration saved!");
    });

    // toolchain show
    auto* show_cmd = tc->add_subcommand("show", "Show current toolchain information");
    show_cmd->callback([&app] {
        const auto& arch = app.config().build.arch;
        auto resolved = app.toolchain_manager().resolve_target(arch);
        if (!resolved) {
            app.printer().error("{}", resolved.error().message());
            return;
        }

        auto details = app.toolchain_manager().get_config_details(*resolved);
        if (!details) {
            app.printer().error("{}", details.error().message());
            return;
        }

        auto installed = app.toolchain_manager().get_installed_toolchains();
        bool toolchain_installed = false;
        if (installed) {
            toolchain_installed = std::any_of(
                installed->begin(), installed->end(),
                [&](const auto& t) { return t.target == details->target && t.installed; });
        }

        const auto yes = std::format("{}yes{}", ui::color::kGreen, ui::color::kReset);
        const auto no = std::format("{}no{}", ui::color::kRed, ui::color::kReset);

        app.printer().print("{}{}Toolchain Overview{}", ui::color::kBold, ui::color::kMagenta,
                            ui::color::kReset);
        app.printer().print("  {}Selected architecture:{}  {}", ui::color::kCyan, ui::color::kReset,
                            arch);
        app.printer().print("  {}Resolved target:{}        {}", ui::color::kCyan, ui::color::kReset,
                            details->target);
        app.printer().print("  {}ct-ng installed:{}         {}", ui::color::kCyan,
                            ui::color::kReset, app.toolchain_manager().is_installed() ? yes : no);
        app.printer().print("  {}Built target present:{}    {}", ui::color::kCyan,
                            ui::color::kReset, toolchain_installed ? yes : no);

        app.printer().print("");
        app.printer().print("{}{}Toolchain Details{}", ui::color::kBold, ui::color::kBlue,
                            ui::color::kReset);
        app.printer().print("  {}CT_ARCH:{}               {}", ui::color::kCyan, ui::color::kReset,
                            details->ct_arch);
        app.printer().print("  {}CT_ARCH_BITNESS:{}       {}", ui::color::kCyan, ui::color::kReset,
                            details->ct_arch_bitness.empty() ? "n/a" : details->ct_arch_bitness);
        app.printer().print("  {}GCC version:{}           {}", ui::color::kCyan, ui::color::kReset,
                            details->gcc_version.empty() ? "unknown" : details->gcc_version);
        app.printer().print("  {}C library:{}             {}", ui::color::kCyan, ui::color::kReset,
                            details->libc.empty() ? "unknown" : details->libc);
        app.printer().print("  {}C library version:{}     {}", ui::color::kCyan, ui::color::kReset,
                            details->libc_version.empty() ? "unknown" : details->libc_version);
        app.printer().print("  {}Install prefix:{}        {}", ui::color::kCyan, ui::color::kReset,
                            details->prefix_dir.empty() ? "unknown" : details->prefix_dir);
        app.printer().print("  {}Bin directory:{}         {}", ui::color::kCyan, ui::color::kReset,
                            app.toolchain_manager().get_bin_dir(details->target));
        if (app.toolchain_manager().is_installed()) {
            app.printer().print("  {}ct-ng binary:{}          {}", ui::color::kCyan,
                                ui::color::kReset, app.toolchain_manager().ct_ng_bin());
        }
        app.printer().print("  {}Config file:{}           {}", ui::color::kCyan, ui::color::kReset,
                            details->config_file);
    });

    // toolchain clean
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

    // toolchain <target> — positional to pick a toolchain target
    auto* target_opt = tc->add_option("target", "Toolchain target to select");

    tc->callback([&app, tc, status_cmd, clone_cmd, build_cmd, list_cmd, menuconfig_cmd, show_cmd,
                  clean_cmd, target_opt] {
        // Skip if a subcommand already handled it
        if (status_cmd->parsed() || clone_cmd->parsed() || build_cmd->parsed() ||
            list_cmd->parsed() || menuconfig_cmd->parsed() || show_cmd->parsed() ||
            clean_cmd->parsed())
            return;

        if (target_opt->count() == 0) {
            std::cout << tc->help();
            return;
        }

        auto select_target = target_opt->as<std::string>();
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
        app.printer().success("Selected toolchain target: {}", select_target);
        app.printer().info("Use 'elmos toolchain build' to compile this toolchain");
    });
}

}  // namespace elmos::app::commands
