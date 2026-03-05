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
};

class QEMURunner {
public:
    explicit QEMURunner(context::Context* ctx);

    auto run(std::stop_token token, RunOptions opts) -> VoidResult;
    auto build_command(const RunOptions& opts)
        -> Result<std::pair<std::string, std::vector<std::string>>>;
    auto is_available(std::stop_token token) -> bool;

private:
    context::Context* ctx_;
};

}  // namespace elmos::domain::emulator
