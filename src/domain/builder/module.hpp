#pragma once
// ============================================================================
// domain/builder/module.hpp — Kernel module build orchestration
// ============================================================================

#include <elmos/common.hpp>

#include <stop_token>
#include <string>
#include <vector>

namespace elmos::context {
class Context;
}
namespace elmos::domain::toolchain {
class Manager;
}

namespace elmos::domain::builder {

struct ModuleInfo {
    std::string name;
    std::string path;
    std::string description;
    bool built = false;
};

class ModuleBuilder {
public:
    ModuleBuilder(context::Context* ctx, toolchain::Manager* tm);

    auto build(std::stop_token token, const std::string& name = "") -> VoidResult;
    auto clean(std::stop_token token, const std::string& name = "") -> VoidResult;
    auto get_modules(const std::string& name = "") -> Result<std::vector<ModuleInfo>>;
    auto prepare_headers(std::stop_token token) -> VoidResult;
    auto create_module(const std::string& name) -> VoidResult;

private:
    context::Context* ctx_;
    toolchain::Manager* tm_;

    auto build_module(std::stop_token token, const ModuleInfo& mod) -> VoidResult;
    auto get_module_info(const std::string& name, const std::string& path) -> ModuleInfo;
    static auto extract_description(const std::string& content) -> std::string;
};

}  // namespace elmos::domain::builder
