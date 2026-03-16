// ============================================================================
// domain/doctor/checker.cpp — Environment health checks
// ============================================================================

#include "checker.hpp"

#include <config/arch.hpp>
#include <config/defaults.hpp>
#include <config/types.hpp>
#include <config/workspaces.hpp>
#include <domain/toolchain/manager.hpp>

#include <filesystem>

namespace elmos::domain::doctor {

namespace {

const std::vector<std::string>& required_asm_headers() {
    static const std::vector<std::string> headers = {
        "bitsperlong.h",
        "int-ll64.h",
        "posix_types.h",
        "types.h",
    };
    return headers;
}

auto get_install_hint([[maybe_unused]] infra::platform::Platform* platform,
                      const std::string& pkg_name) -> std::string {
#ifdef ELMOS_PLATFORM_DARWIN
    return "brew install " + pkg_name;
#elif defined(ELMOS_PLATFORM_LINUX)
    return "sudo apt install " + pkg_name;
#else
    return "";
#endif
}

}  // namespace

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
    return CheckResult{
        .name = "Homebrew",
        .passed = r.has_value(),
        .required = true,
        .message = "Install from: https://brew.sh",
        .category = "System",
        .install_hint = "/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/"
                        "install/HEAD/install.sh)\"",
    };
#else
    for (const auto& mgr : {"apt-get", "dnf", "apk", "pacman"}) {
        if (exec_->look_path(mgr).has_value()) {
            return CheckResult{
                .name = "Package Manager", .passed = true, .required = true, .category = "System"};
        }
    }
    return CheckResult{.name = "Package Manager",
                       .passed = false,
                       .required = true,
                       .message = "apt-get, dnf, apk, or pacman required",
                       .category = "System"};
#endif
}

auto HealthChecker::check_packages(std::stop_token /*token*/) -> std::vector<CheckResult> {
    std::vector<CheckResult> results;
    for (const auto& pkg : config::required_packages()) {
        bool installed = platform_->packages().is_installed(std::string(pkg.name));
        auto hint = installed ? std::string{} : get_install_hint(platform_, std::string(pkg.name));
        results.push_back(CheckResult{
            .name = std::string(pkg.name),
            .passed = installed,
            .required = pkg.required,
            .message = std::string(pkg.description),
            .category = std::string(pkg.category),
            .install_hint = std::move(hint),
        });
    }
    return results;
}

auto HealthChecker::check_headers() -> std::vector<CheckResult> {
    namespace fs = std::filesystem;

    std::vector<CheckResult> results;

    // Check headers at $HOME/.elmos/sysroot/ (the global sysroot)
    auto sysroot = config::WorkspaceManager::global_elmos_dir() + "/sysroot";

    for (auto hdr : config::required_headers()) {
        auto path = sysroot + "/" + std::string(hdr);
        bool exists = fs_->exists(path);

        // For elf.h, also check if it's empty
        if (exists && hdr == "elf.h") {
            std::error_code ec;
            auto size = std::filesystem::file_size(path, ec);
            if (ec || size == 0)
                exists = false;
        }

        std::string hint;
        if (!exists) {
            hint = "Auto-fix: elmos doctor --fix (downloads from elmos repository)";
        }

        results.push_back(CheckResult{
            .name = "Header: " + std::string(hdr),
            .passed = exists,
            .required = true,
            .category = "Headers",
            .install_hint = std::move(hint),
        });
    }

    auto active_ws = config::WorkspaceManager::get_active_workspace();
    if (!active_ws) {
        results.push_back(CheckResult{
            .name = "Workspace sysroot asm links",
            .passed = false,
            .required = false,
            .message = "No active workspace",
            .category = "Headers",
            .install_hint = "Run: elmos init <name> or elmos pick <name>",
        });
        return results;
    }

    auto asm_generic = fs::path(cfg_->paths.kernel_dir) / "include" / "uapi" / "asm-generic";
    if (!fs::is_directory(asm_generic)) {
        results.push_back(CheckResult{
            .name = "Workspace sysroot asm links",
            .passed = false,
            .required = false,
            .message = "Kernel headers not found at " + asm_generic.string(),
            .category = "Headers",
            .install_hint = "Run: elmos kernel clone, then elmos doctor -f",
        });
        return results;
    }

    auto ws_asm_dir = fs::path(config::WorkspaceManager::workspace_dir(*active_ws)) / "sysroot" /
                      "asm";
    for (const auto& name : required_asm_headers()) {
        auto link_path = ws_asm_dir / name;
        auto expected_target = asm_generic / name;

        bool ok = false;
        std::string message;
        std::error_code ec;

        if (!fs::exists(expected_target, ec)) {
            message = "missing kernel header target: " + expected_target.string();
        }
        else if (!fs::exists(link_path, ec)) {
            message = "missing link: " + link_path.string();
        }
        else if (!fs::is_symlink(link_path, ec)) {
            message = "not a symbolic link: " + link_path.string();
        }
        else {
            auto current_target = fs::read_symlink(link_path, ec);
            if (ec) {
                message = "cannot read symlink: " + link_path.string();
            }
            else {
                auto resolved_target = current_target.is_absolute()
                                           ? current_target
                                           : link_path.parent_path() / current_target;
                if (resolved_target.lexically_normal() == expected_target.lexically_normal()) {
                    ok = true;
                }
                else {
                    message = "link target mismatch: " + link_path.string();
                }
            }
        }

        results.push_back(CheckResult{
            .name = "Workspace asm: " + name,
            .passed = ok,
            .required = false,
            .message = ok ? (link_path.string() + " -> " + expected_target.string())
                          : std::move(message),
            .category = "Headers",
            .install_hint = ok ? std::string{} : "Run: elmos doctor -f",
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
            .category = "Cross-compilers",
            .install_hint = r.has_value() ? std::string{}
                                          : "Build with: elmos toolchain build " + name,
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
            .category = "Cross-compilers",
            .install_hint = r.has_value() ? std::string{}
                                          : "Build with: elmos toolchain build " + name,
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
        .message = "Run: elmos toolchain clone",
        .category = "Toolchains",
        .install_hint = tm_->is_installed() ? std::string{} : "elmos toolchain clone",
    });

    if (tm_->is_installed()) {
        auto tcs = tm_->get_installed_toolchains();
        if (tcs) {
            for (const auto& tc : *tcs) {
                results.push_back({
                    .name = "Toolchain: " + tc.target,
                    .passed = tc.installed,
                    .required = false,
                    .category = "Toolchains",
                });
            }
        }
    }
    return results;
}

auto HealthChecker::fix_packages(std::stop_token token, const std::vector<CheckResult>& failed)
    -> std::vector<std::string> {
    std::vector<std::string> fixed;
    for (const auto& r : failed) {
        if (r.passed || r.category == "Headers" || r.category == "Cross-compilers" ||
            r.category == "Toolchains" || r.category == "System")
            continue;
        auto result = platform_->packages().install(token, r.name);
        if (result) {
            fixed.push_back(r.name);
        }
    }
    return fixed;
}

auto HealthChecker::fix_headers(std::stop_token token) -> std::vector<std::string> {
    namespace fs = std::filesystem;
    std::vector<std::string> fixed;

    // Target: $HOME/.elmos/sysroot/
    auto sysroot_dir = config::WorkspaceManager::global_elmos_dir() + "/sysroot";
    std::error_code ec;
    fs::create_directories(sysroot_dir, ec);

    // Download elf.h from glibc if missing or empty.
    auto elf_h = fs::path(sysroot_dir) / "elf.h";
    bool need_elf = !fs::exists(elf_h);
    if (!need_elf) {
        auto size = fs::file_size(elf_h, ec);
        need_elf = ec || size == 0;
    }

    if (need_elf) {
        static constexpr auto kElfHeaderUrl =
            "https://raw.githubusercontent.com/bminor/glibc/master/elf/elf.h";
        auto r = exec_->run(token, "curl", {"-fsSL", "-o", elf_h.string(), kElfHeaderUrl});
        if (r) {
            fixed.push_back("elf.h");
        }
    }

    // Download byteswap.h and endian.h from elmos repository (refactor branch)
    // if they don't exist at $HOME/.elmos/sysroot/.
    static constexpr auto kRepoRawBase =
        "https://raw.githubusercontent.com/NguyenTrongPhuc552003/elmos/refactor/"
        "include/sysroot/";

    for (const auto* name : {"byteswap.h", "endian.h"}) {
        auto dst = fs::path(sysroot_dir) / name;
        if (fs::exists(dst))
            continue;
        auto url = std::string(kRepoRawBase) + name;
        auto r = exec_->run(token, "curl", {"-fsSL", "-o", dst.string(), url});
        if (r) {
            fixed.push_back(name);
        }
    }

    auto active_ws = config::WorkspaceManager::get_active_workspace();
    if (!active_ws) {
        return fixed;
    }

    auto asm_generic = fs::path(cfg_->paths.kernel_dir) / "include" / "uapi" / "asm-generic";
    if (!fs::is_directory(asm_generic)) {
        return fixed;
    }

    auto ws_asm_dir = fs::path(config::WorkspaceManager::workspace_dir(*active_ws)) / "sysroot" /
                      "asm";
    fs::create_directories(ws_asm_dir, ec);

    for (const auto& name : required_asm_headers()) {
        auto link_path = ws_asm_dir / name;
        auto target_path = asm_generic / name;

        if (!fs::exists(target_path)) {
            continue;
        }

        bool already_correct = false;
        std::error_code read_ec;
        if (fs::exists(link_path) && fs::is_symlink(link_path)) {
            auto current_target = fs::read_symlink(link_path, read_ec);
            if (!read_ec) {
                auto resolved_target = current_target.is_absolute()
                                           ? current_target
                                           : link_path.parent_path() / current_target;
                already_correct =
                    (resolved_target.lexically_normal() == target_path.lexically_normal());
            }
        }

        if (already_correct) {
            continue;
        }

        std::error_code rm_ec;
        fs::remove(link_path, rm_ec);

        std::error_code link_ec;
        fs::create_symlink(target_path, link_path, link_ec);
        if (!link_ec) {
            fixed.push_back("workspace-sysroot/asm/" + name);
        }
    }

    return fixed;
}

}  // namespace elmos::domain::doctor
