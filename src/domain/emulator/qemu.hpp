#pragma once
// ============================================================================
// domain/emulator/qemu.hpp — QEMU integration
// ============================================================================

#include <elmos/common.hpp>

#include <infra/executor/interface.hpp>

#include <stop_token>
#include <string>
#include <vector>

namespace elmos::config {
struct Config;
}
namespace elmos::context {
class Context;
}

namespace elmos::domain::emulator {

struct RunOptions {
    bool gdb = false;
    bool graphic = false;
    std::string initrd;
    std::string append;
    std::vector<std::string> extra_args;
    /// When true, always pass -kernel <linux_image> directly even if a U-Boot
    /// binary is present in the bootloader directory.
    bool force_direct_kernel = false;
};

class QEMURunner {
public:
    explicit QEMURunner(context::Context* ctx);

    auto run(std::stop_token token, RunOptions opts) -> VoidResult;
    auto build_command(const RunOptions& opts)
        -> Result<std::pair<std::string, std::vector<std::string>>>;
    auto is_available(std::stop_token token) -> bool;

    /// Returns true when a U-Boot binary exists in the bootloader directory.
    auto has_bootloader() const -> bool;

    /// Returns the path to the U-Boot binary, or empty string if not found.
    /// Search order: u-boot.bin → u-boot.itb → u-boot (ELF).
    auto find_bootloader() const -> std::string;

private:
    context::Context* ctx_;
};

}  // namespace elmos::domain::emulator
