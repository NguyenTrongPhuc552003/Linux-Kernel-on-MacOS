// ============================================================================
// app/commands/arch.cpp — Architecture selection command
// ============================================================================

#include <app/app.hpp>
#include <config/arch.hpp>

#include "commands.hpp"

namespace elmos::app::commands {

namespace {
void show_arch(App& app) {
    auto& cfg = app.config();
    app.printer().step("Current architecture: {}", cfg.build.arch);
    auto ac = config::get_arch_config(cfg.build.arch);
    if (ac) {
        app.printer().print("  Kernel arch:  {}", ac->kernel_arch);
        app.printer().print("  Kernel image: {}", ac->kernel_image);
        app.printer().print("  QEMU system:  {}", ac->qemu_binary);
        app.printer().print("  GCC prefix:   {}", ac->gcc_binary);
    }
}
}  // namespace

void register_arch(App& app, CLI::App& cli) {
    auto* arch = cli.add_subcommand("arch", "Set or show target architecture");

    // arch [target] — positional arg; no args → show current
    // Using Option* (not a local value ref) so the pointer stays valid after
    // register_arch() returns and the CLI::App owns the option's lifetime.
    auto* target_opt = arch->add_option("target", "Architecture to set (arm64 / arm / riscv)");

    // arch show — explicit subcommand for scripting/completions
    auto* show_cmd = arch->add_subcommand("show", "Show current architecture details");
    show_cmd->callback([&app] { show_arch(app); });

    // arch list — list all supported architectures
    auto* list_cmd = arch->add_subcommand("list", "List available architectures");
    list_cmd->callback([&app] {
        app.printer().step("Available architectures:");
        for (auto& a : config::supported_architectures()) {
            auto ac = config::get_arch_config(a);
            bool is_current = (a == app.config().build.arch);
            app.printer().print("  {} {} — kernel_arch={}, image={}", is_current ? "*" : " ", a,
                                ac ? ac->kernel_arch : "?", ac ? ac->kernel_image : "?");
        }
    });

    arch->callback([&app, show_cmd, list_cmd, target_opt] {
        // In CLI11, parent callbacks run after subcommand callbacks.
        // Skip the default arch logic if a subcommand already ran.
        if (show_cmd->parsed() || list_cmd->parsed())
            return;
        if (target_opt->count() == 0) {
            // No target given → show current architecture
            show_arch(app);
            return;
        }
        auto target = target_opt->as<std::string>();
        if (!config::is_valid_arch(target)) {
            app.printer().error("Invalid architecture: '{}'", target);
            app.printer().info("Supported: {}", [] {
                std::string s;
                for (auto& a : config::supported_architectures()) {
                    if (!s.empty())
                        s += ", ";
                    s += a;
                }
                return s;
            }());
            return;
        }
        app.config().build.arch = target;
        app.save_workspace_config();
        app.printer().success("Architecture set to '{}'", target);
    });
}

}  // namespace elmos::app::commands
