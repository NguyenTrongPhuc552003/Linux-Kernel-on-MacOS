// ============================================================================
// app/commands/qemu.cpp — QEMU emulator commands
// ============================================================================

#include <app/app.hpp>
#include <domain/emulator/qemu.hpp>

#include "commands.hpp"

#include <stop_token>

namespace elmos::app::commands {

void register_qemu(App& app, CLI::App& cli) {
    auto* qemu = cli.add_subcommand("qemu", "Run kernel in emulator");

    // qemu run
    auto* run_cmd = qemu->add_subcommand("run", "Boot kernel in QEMU");
    run_cmd->callback([&app] {
        if (!app.context().has_kernel_image()) {
            app.printer().warn("Kernel not built. Run 'elmos kernel build' first.");
            return;
        }
        app.printer().step("Launching QEMU...");
        std::stop_source ss;
        if (auto r = app.qemu_runner().run(ss.get_token(), domain::emulator::RunOptions{}); !r) {
            app.printer().error("QEMU failed: {}", r.error().message());
        }
    });

    // qemu debug
    auto* debug_cmd = qemu->add_subcommand("debug", "Boot with GDB server");
    debug_cmd->callback([&app] {
        if (!app.context().has_kernel_image()) {
            app.printer().warn("Kernel not built. Run 'elmos kernel build' first.");
            return;
        }
        app.printer().step("Launching QEMU with GDB server...");
        std::stop_source ss;
        if (auto r = app.qemu_runner().run(ss.get_token(),
                                           domain::emulator::RunOptions{.gdb = true});
            !r) {
            app.printer().error("QEMU failed: {}", r.error().message());
        }
    });
}

}  // namespace elmos::app::commands
