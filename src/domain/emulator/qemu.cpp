// ============================================================================
// domain/emulator/qemu.cpp — QEMU integration
// ============================================================================

#include "qemu.hpp"

#include <config/arch.hpp>
#include <config/types.hpp>
#include <context/context.hpp>

namespace elmos::domain::emulator {

QEMURunner::QEMURunner(context::Context* ctx) : ctx_(ctx) {}

auto QEMURunner::is_available(std::stop_token token) -> bool {
    auto arch_cfg = ctx_->config().get_arch_config();
    if (!arch_cfg)
        return false;
    return ctx_->exec().look_path(arch_cfg->qemu_binary).has_value();
}

auto QEMURunner::build_command(const RunOptions& opts)
    -> Result<std::pair<std::string, std::vector<std::string>>> {
    auto& cfg = ctx_->config();
    auto arch_cfg = cfg.get_arch_config();
    if (!arch_cfg) {
        return make_error(Error::config("no architecture config for " + cfg.build.arch));
    }

    auto kernel_image = ctx_->get_kernel_image();
    if (kernel_image.empty()) {
        return make_error(Error::build("kernel image not found"));
    }

    std::vector<std::string> args;

    args.push_back("-machine");
    args.push_back(arch_cfg->qemu_machine);
    args.push_back("-cpu");
    args.push_back(arch_cfg->qemu_cpu);
    args.push_back("-m");
    args.push_back(cfg.qemu.memory);
    args.push_back("-smp");
    args.push_back(std::to_string(cfg.qemu.smp));
    args.push_back("-kernel");
    args.push_back(kernel_image);

    if (!opts.graphic) {
        args.push_back("-nographic");
    }

    // Serial console
    std::string console = arch_cfg->console;
    std::string append = "console=" + console + " " + opts.append;
    args.push_back("-append");
    args.push_back(append);

    // Network
    args.push_back("-netdev");
    args.push_back("user,id=net0,hostfwd=tcp::" + std::to_string(cfg.qemu.ssh_port) + "-:22");
    args.push_back("-device");
    args.push_back("virtio-net-pci,netdev=net0");

    // Initrd
    if (!opts.initrd.empty()) {
        args.push_back("-initrd");
        args.push_back(opts.initrd);
    }

    // GDB
    if (opts.gdb) {
        args.push_back("-s");
        args.push_back("-S");
    }

    // Extra args
    for (const auto& a : opts.extra_args) {
        args.push_back(a);
    }

    return std::pair{arch_cfg->qemu_binary, args};
}

auto QEMURunner::run(std::stop_token token, RunOptions opts) -> VoidResult {
    auto cmd = build_command(opts);
    if (!cmd)
        return make_error(cmd.error());
    auto& [binary, args] = *cmd;
    return ctx_->exec().run(token, binary, args);
}

}  // namespace elmos::domain::emulator
