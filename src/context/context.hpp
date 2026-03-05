#pragma once
// ============================================================================
// context/context.hpp — Build context replacing Go's core/context/context.go
// ============================================================================

#include <elmos/common.hpp>

#include <config/types.hpp>
#include <infra/executor/interface.hpp>
#include <infra/filesystem/interface.hpp>
#include <infra/homebrew/resolver.hpp>
#include <infra/platform/interface.hpp>

#include <filesystem>
#include <memory>
#include <stop_token>
#include <string>
#include <vector>

namespace elmos::context {

namespace fs = std::filesystem;

/// Build context holding configuration and all infrastructure dependencies.
/// Replaces Go's Context struct — passed by reference, never by value.
class Context {
public:
    Context(config::Config* cfg, infra::executor::Executor* exec,
            infra::filesystem::FileSystem* filesystem);

    // Accessors
    auto config() -> config::Config& { return *config_; }
    auto config() const -> const config::Config& { return *config_; }
    auto exec() -> infra::executor::Executor& { return *exec_; }
    auto fs() -> infra::filesystem::FileSystem& { return *fs_; }
    auto platform() -> infra::platform::Platform& { return *platform_; }
    auto brew() -> infra::homebrew::Resolver* { return brew_.get(); }

    bool verbose = false;

    // State queries
    auto is_mounted(std::stop_token token = {}) -> bool;
    auto ensure_mounted(std::stop_token token = {}) -> VoidResult;
    auto get_actual_mount_point(std::stop_token token = {}) -> Result<std::string>;
    auto kernel_exists() -> bool;
    auto has_config() -> bool;
    auto get_kernel_image() -> std::string;
    auto get_vmlinux() -> std::string;
    auto has_kernel_image() -> bool;
    auto get_default_targets() -> std::vector<std::string>;

    // Environment construction for make commands
    auto get_make_env() -> EnvList;

    // Platform ownership
    void set_platform(std::unique_ptr<infra::platform::Platform> p);

private:
    config::Config* config_;
    infra::executor::Executor* exec_;
    infra::filesystem::FileSystem* fs_;
    std::unique_ptr<infra::platform::Platform> platform_owned_;
    infra::platform::Platform* platform_ = nullptr;
    std::unique_ptr<infra::homebrew::Resolver> brew_;

    auto prepend_brew_tool_paths(const std::string& current_path) -> std::string;
    auto build_host_cflags() -> std::string;
};

}  // namespace elmos::context
