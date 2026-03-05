#pragma once
// ============================================================================
// domain/builder/app.hpp — Userspace application build orchestration
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

struct AppInfo {
    std::string name;
    std::string path;
    bool built = false;
};

class AppBuilder {
public:
    AppBuilder(context::Context* ctx, toolchain::Manager* tm);

    auto build(std::stop_token token, const std::string& name = "") -> VoidResult;
    auto clean(std::stop_token token, const std::string& name = "") -> VoidResult;
    auto get_apps(const std::string& name = "") -> Result<std::vector<AppInfo>>;
    auto create_app(const std::string& name) -> VoidResult;

private:
    context::Context* ctx_;
    toolchain::Manager* tm_;

    auto build_app(std::stop_token token, const AppInfo& app, const std::string& compiler,
                   const EnvList& env) -> VoidResult;
    auto get_cross_compiler(const std::string& prefix) -> std::string;
    auto get_app_info(const std::string& name, const std::string& path) -> AppInfo;
};

}  // namespace elmos::domain::builder
