#pragma once
// ============================================================================
// domain/toolchain/manager.hpp — Cross-compilation toolchain management
// Replaces Go's core/domain/toolchain/
// ============================================================================

#include <elmos/common.hpp>

#include <config/arch.hpp>
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

struct ToolchainInfo {
    std::string target;  // e.g. "aarch64-unknown-linux-gnu"
    std::string config_file;
    bool installed = false;
};

struct ToolchainConfigDetails {
    std::string target;
    std::string config_file;
    std::string ct_arch;
    std::string ct_arch_bitness;
    std::string gcc_version;
    std::string libc;
    std::string libc_version;
    std::string prefix_dir;
};

struct ToolchainPaths {
    std::string base_dir;    // <workspace>/toolchains
    std::string x_tools;     // <workspace>/toolchains/x-tools
    std::string config_dir;  // <workspace>/toolchains/configs
    std::string build_dir;   // <workspace>/toolchains/build
    std::string ct_ng_dir;   // <workspace>/toolchains/crosstool-ng
};

/// Manages cross-compilation toolchains built via crosstool-ng.
class Manager {
public:
    Manager(infra::executor::Executor* exec, infra::filesystem::FileSystem* fs,
            infra::platform::Platform* platform, config::Config* cfg);

    auto paths() const -> const ToolchainPaths& { return paths_; }

    /// Full path to the workspace-local ct-ng binary.
    auto ct_ng_bin() const -> std::string;

    auto is_installed() const -> bool;
    auto install(std::stop_token token) -> VoidResult;

    /// Resolve an arch name ("riscv") or full target ("riscv64-unknown-linux-gnu")
    /// to the actual config file base name that exists in configs/.
    auto resolve_target(const std::string& arch_or_target) -> Result<std::string>;

    auto build_toolchain(std::stop_token token, const std::string& target) -> VoidResult;
    auto get_installed_toolchains() -> Result<std::vector<ToolchainInfo>>;
    auto get_available_configs() -> Result<std::vector<std::string>>;
    auto get_targets_for_arch(const std::string& arch) -> Result<std::vector<std::string>>;
    auto get_config_details(const std::string& target) -> Result<ToolchainConfigDetails>;
    auto get_bin_dir(const std::string& target) -> std::string;

    /// Verify a built toolchain by checking key binaries exist and the
    /// compiler can produce output. Returns an error describing what failed.
    auto verify_toolchain(std::stop_token token, const std::string& target) -> VoidResult;

private:
    infra::executor::Executor* exec_;
    infra::filesystem::FileSystem* fs_;
    infra::platform::Platform* platform_;
    [[maybe_unused]] config::Config* cfg_;
    ToolchainPaths paths_;

    void init_paths();
    auto get_build_env() -> EnvList;

    /// Patch .config content: replace ${ELMOS_WORKSPACE} variable references
    /// and set CT_PREFIX_DIR / CT_LOCAL_TARBALLS_DIR to absolute paths.
    /// (Go equivalent: patchConfig / patchConfigContent)
    auto patch_config(const std::string& content) const -> std::string;
};

}  // namespace elmos::domain::toolchain
