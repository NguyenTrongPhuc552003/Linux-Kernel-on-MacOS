#pragma once
// ============================================================================
// domain/builder/kernel.hpp — Kernel build orchestration
// ============================================================================

#include <elmos/common.hpp>

#include <stop_token>
#include <string>
#include <vector>

namespace elmos::config {
struct Config;
}
namespace elmos::context {
class Context;
}
namespace elmos::infra::executor {
class Executor;
}
namespace elmos::infra::filesystem {
class FileSystem;
}
namespace elmos::domain::toolchain {
class Manager;
}

namespace elmos::domain::builder {

struct BuildOptions {
    int jobs = 0;
    std::vector<std::string> targets;
};

/// Orchestrates kernel build operations.
class KernelBuilder {
public:
    KernelBuilder(context::Context* ctx, toolchain::Manager* tm);

    auto build(std::stop_token token, BuildOptions opts) -> VoidResult;
    auto configure(std::stop_token token, const std::string& config_type) -> VoidResult;
    auto clean(std::stop_token token) -> VoidResult;
    auto enable_kvm_config(std::stop_token token) -> VoidResult;
    auto get_default_targets() -> std::vector<std::string>;
    auto has_config() -> bool;
    auto has_kernel_image() -> bool;

private:
    context::Context* ctx_;
    toolchain::Manager* tm_;

    auto get_toolchain_env() -> Result<std::pair<EnvList, std::string>>;
    auto force_graphics_config(std::stop_token token) -> VoidResult;
};

}  // namespace elmos::domain::builder
