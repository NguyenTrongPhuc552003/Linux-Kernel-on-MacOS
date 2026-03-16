#pragma once
// ============================================================================
// domain/doctor/checker.hpp — Environment health checks
// ============================================================================

#include <elmos/common.hpp>

#include <infra/executor/interface.hpp>
#include <infra/filesystem/interface.hpp>
#include <infra/platform/interface.hpp>

#include <stop_token>
#include <string>
#include <vector>

namespace elmos::config {
struct Config;
}
namespace elmos::domain::toolchain {
class Manager;
}

namespace elmos::domain::doctor {

struct CheckResult {
    std::string name;
    bool passed = false;
    bool required = true;
    std::string message;
    std::string category;
    std::string install_hint;
};

class HealthChecker {
public:
    HealthChecker(infra::executor::Executor* exec, infra::filesystem::FileSystem* fs,
                  config::Config* cfg, infra::platform::Platform* platform, toolchain::Manager* tm);

    auto check_all(std::stop_token token) -> std::pair<std::vector<CheckResult>, int>;
    auto check_package_manager(std::stop_token token) -> CheckResult;
    auto check_packages(std::stop_token token) -> std::vector<CheckResult>;
    auto check_headers() -> std::vector<CheckResult>;
    auto check_cross_gdb(std::stop_token token) -> std::vector<CheckResult>;
    auto check_cross_gcc(std::stop_token token) -> std::vector<CheckResult>;
    auto check_toolchains() -> std::vector<CheckResult>;

    auto fix_packages(std::stop_token token, const std::vector<CheckResult>& failed)
        -> std::vector<std::string>;
    auto fix_headers(std::stop_token token) -> std::vector<std::string>;

private:
    infra::executor::Executor* exec_;
    infra::filesystem::FileSystem* fs_;
    config::Config* cfg_;
    infra::platform::Platform* platform_;
    toolchain::Manager* tm_;
};

}  // namespace elmos::domain::doctor
