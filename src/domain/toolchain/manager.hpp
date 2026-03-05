#pragma once
// ============================================================================
// domain/toolchain/manager.hpp — Cross-compilation toolchain management
// Replaces Go's core/domain/toolchain/
// ============================================================================

#include <elmos/common.hpp>

#include <infra/executor/interface.hpp>
#include <infra/filesystem/interface.hpp>

#include <stop_token>
#include <string>
#include <vector>

namespace elmos::config {
struct Config;
}

namespace elmos::domain::toolchain {

struct ToolchainInfo {
    std::string target;  // e.g. "aarch64-unknown-linux-gnu"
    std::string config_file;
    bool installed = false;
};

struct ToolchainPaths {
    std::string base_dir;    // ~/.elmos/toolchains
    std::string x_tools;     // ~/.elmos/toolchains/x-tools
    std::string config_dir;  // ~/.elmos/toolchains/configs
    std::string build_dir;   // ~/.elmos/toolchains/build
    std::string ct_ng_dir;   // ~/.elmos/toolchains/crosstool-ng
};

/// Manages cross-compilation toolchains built via crosstool-ng.
class Manager {
public:
    Manager(infra::executor::Executor* exec, infra::filesystem::FileSystem* fs,
            config::Config* cfg);

    auto paths() const -> const ToolchainPaths& { return paths_; }

    auto is_installed() const -> bool;
    auto install(std::stop_token token) -> VoidResult;
    auto build_toolchain(std::stop_token token, const std::string& target) -> VoidResult;
    auto get_installed_toolchains() -> Result<std::vector<ToolchainInfo>>;
    auto get_available_configs() -> Result<std::vector<std::string>>;
    auto get_bin_dir(const std::string& target) -> std::string;

private:
    infra::executor::Executor* exec_;
    infra::filesystem::FileSystem* fs_;
    [[maybe_unused]] config::Config* cfg_;
    ToolchainPaths paths_;

    void init_paths();
};

}  // namespace elmos::domain::toolchain
