// ============================================================================
// domain/doctor/checker.cpp — Environment health checks
// ============================================================================

#include "checker.hpp"

#include <config/arch.hpp>
#include <config/defaults.hpp>
#include <config/types.hpp>
#include <domain/toolchain/manager.hpp>

namespace elmos::domain::doctor {

HealthChecker::HealthChecker(infra::executor::Executor* exec, infra::filesystem::FileSystem* fs,
                             config::Config* cfg, infra::platform::Platform* platform,
                             toolchain::Manager* tm)
    : exec_(exec), fs_(fs), cfg_(cfg), platform_(platform), tm_(tm) {}

auto HealthChecker::check_all(std::stop_token token) -> std::pair<std::vector<CheckResult>, int> {
    std::vector<CheckResult> results;
    int issues = 0;

    auto add = [&](CheckResult r) {
        if (!r.passed && r.required)
            issues++;
        results.push_back(std::move(r));
    };

    add(check_package_manager(token));

    for (auto& r : check_packages(token))
        add(std::move(r));
    for (auto& r : check_headers())
        add(std::move(r));
    for (auto& r : check_cross_gdb(token))
        add(std::move(r));
    for (auto& r : check_cross_gcc(token))
        add(std::move(r));
    for (auto& r : check_toolchains())
        add(std::move(r));

    return {results, issues};
}

auto HealthChecker::check_package_manager(std::stop_token token) -> CheckResult {
#ifdef ELMOS_PLATFORM_DARWIN
    auto r = exec_->look_path("brew");
    return CheckResult{.name = "Homebrew",
                       .passed = r.has_value(),
                       .required = true,
                       .message = "Install from: https://brew.sh"};
#else
    for (const auto& mgr : {"apt-get", "dnf", "apk", "pacman"}) {
        if (exec_->look_path(mgr).has_value()) {
            return CheckResult{.name = "Package Manager", .passed = true, .required = true};
        }
    }
    return CheckResult{.name = "Package Manager",
                       .passed = false,
                       .required = true,
                       .message = "apt-get, dnf, apk, or pacman required"};
#endif
}

auto HealthChecker::check_packages(std::stop_token /*token*/) -> std::vector<CheckResult> {
    std::vector<CheckResult> results;
    for (const auto& pkg : config::required_packages()) {
        bool installed = platform_->packages().is_installed(std::string(pkg.name));
        results.push_back(CheckResult{
            .name = std::string(pkg.name),
            .passed = installed,
            .required = pkg.required,
            .message = std::string(pkg.description),
        });
    }
    return results;
}

auto HealthChecker::check_headers() -> std::vector<CheckResult> {
    std::vector<CheckResult> results;
    for (auto hdr : config::required_headers()) {
        auto path = cfg_->paths.libraries_dir + "/" + std::string(hdr);
        results.push_back(CheckResult{
            .name = "Header: " + std::string(hdr),
            .passed = fs_->exists(path),
            .required = true,
        });
    }
    return results;
}

auto HealthChecker::check_cross_gdb(std::stop_token /*token*/) -> std::vector<CheckResult> {
    std::vector<CheckResult> results;
    for (const auto& [name, arch] : config::architectures()) {
        if (arch.gdb_binary.empty())
            continue;
        auto r = exec_->look_path(arch.gdb_binary);
        results.push_back(CheckResult{
            .name = "GDB: " + arch.gdb_binary,
            .passed = r.has_value(),
            .required = false,
        });
    }
    return results;
}

auto HealthChecker::check_cross_gcc(std::stop_token /*token*/) -> std::vector<CheckResult> {
    std::vector<CheckResult> results;
    for (const auto& [name, arch] : config::architectures()) {
        if (arch.gcc_binary.empty())
            continue;
        auto r = exec_->look_path(arch.gcc_binary);
        results.push_back(CheckResult{
            .name = "GCC: " + arch.gcc_binary,
            .passed = r.has_value(),
            .required = false,
        });
    }
    return results;
}

auto HealthChecker::check_toolchains() -> std::vector<CheckResult> {
    std::vector<CheckResult> results;
    results.push_back({
        .name = "crosstool-ng",
        .passed = tm_->is_installed(),
        .required = false,
        .message = "Run: elmos toolchains install",
    });

    if (tm_->is_installed()) {
        auto tcs = tm_->get_installed_toolchains();
        if (tcs) {
            for (const auto& tc : *tcs) {
                results.push_back({
                    .name = "Toolchain: " + tc.target,
                    .passed = tc.installed,
                    .required = false,
                });
            }
        }
    }
    return results;
}

}  // namespace elmos::domain::doctor
