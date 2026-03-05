#pragma once
// ============================================================================
// domain/rootfs/builder.hpp — Root filesystem creation
// ============================================================================

#include <elmos/common.hpp>

#include <stop_token>
#include <string>
#include <vector>

namespace elmos::context {
class Context;
}

namespace elmos::domain::rootfs {

struct RootfsOptions {
    std::string distribution = "debian";
    std::string release = "bookworm";
    std::vector<std::string> packages;
    std::string post_build_script;
};

class Builder {
public:
    explicit Builder(context::Context* ctx);

    auto create(std::stop_token token, RootfsOptions opts) -> VoidResult;
    auto install_modules(std::stop_token token) -> VoidResult;
    auto customize(std::stop_token token, const std::string& script) -> VoidResult;
    auto clean() -> VoidResult;

private:
    context::Context* ctx_;

    auto run_debootstrap(std::stop_token token, const RootfsOptions& opts) -> VoidResult;
    auto install_packages(std::stop_token token, const std::vector<std::string>& packages)
        -> VoidResult;
};

}  // namespace elmos::domain::rootfs
