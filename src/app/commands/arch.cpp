// ============================================================================
// app/commands/arch.cpp — Architecture selection command
// ============================================================================

#include <app/app.hpp>
#include <config/arch.hpp>

#include "commands.hpp"

namespace elmos::app::commands {

void register_arch(App& app, CLI::App& cli) {
    auto* arch = cli.add_subcommand("arch", "Set target architecture");

    // arch show
    auto* show_cmd = arch->add_subcommand("show", "Show current architecture config");
    show_cmd->callback([&app] {
        auto& cfg = app.config();
        app.printer().step("Current architecture: {}", cfg.build.arch);
        auto ac = config::get_arch_config(cfg.build.arch);
        if (ac) {
            app.printer().print("  Kernel arch:  {}", ac->kernel_arch);
            app.printer().print("  Kernel image: {}", ac->kernel_image);
            app.printer().print("  QEMU system:  {}", ac->qemu_binary);
            app.printer().print("  GCC prefix:   {}", ac->gcc_binary);
        }
    });

    // arch <target>
    auto* set_cmd = arch->add_subcommand("set", "Set target architecture");
    std::string target;
    set_cmd->add_option("target", target, "Architecture (arm64/arm/riscv)")->required();
    set_cmd->callback([&app, &target] {
        if (!config::is_valid_arch(target)) {
            app.printer().error("Invalid architecture: {}", target);
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
        app.printer().success("Architecture set to {}", target);
    });
}

}  // namespace elmos::app::commands
