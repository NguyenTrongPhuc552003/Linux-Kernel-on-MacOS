#pragma once
// ============================================================================
// platform/interface.hpp — OS-specific abstractions
// Replaces Go's core/infra/platform/interface.go
// ============================================================================

#include <elmos/common.hpp>

#include <stop_token>
#include <string>
#include <vector>

namespace elmos::infra::executor {
class Executor;
}

namespace elmos::infra::platform {

/// DiskImageManager abstracts sparse disk image lifecycle.
class DiskImageManager {
public:
    virtual ~DiskImageManager() = default;

    virtual auto create(std::stop_token token, const std::string& image_path, int size_gb)
        -> VoidResult = 0;

    virtual auto mount(std::stop_token token, const std::string& image_path)
        -> Result<std::string> = 0;

    virtual auto unmount(std::stop_token token, const std::string& mount_point) -> VoidResult = 0;

    struct MountStatus {
        bool mounted;
        std::string mount_point;
    };
    virtual auto is_mounted(std::stop_token token, const std::string& name)
        -> Result<MountStatus> = 0;
};

/// PackageManager abstracts querying and installing system packages.
class PackageManager {
public:
    virtual ~PackageManager() = default;

    virtual auto is_installed(const std::string& pkg) -> bool = 0;
    virtual auto install(std::stop_token token, const std::string& pkg) -> VoidResult = 0;
    virtual auto list_installed() -> Result<std::vector<std::string>> = 0;
    virtual auto get_bin_path(const std::string& pkg) -> std::string = 0;
    virtual auto get_lib_path(const std::string& pkg) -> std::string = 0;
    virtual auto get_include_path(const std::string& pkg) -> std::string = 0;

    /// Build environment variables with GNU tools in PATH for build commands.
    /// On macOS, prepends Homebrew GNU tool paths. On Linux/WSL, returns empty
    /// since GNU tools are the system default.
    virtual auto build_gnu_environment() -> EnvList = 0;
};

/// PathProvider supplies OS-specific default path conventions.
class PathProvider {
public:
    virtual ~PathProvider() = default;

    virtual auto workspace_root(const std::string& name) -> std::string = 0;
    virtual auto cache_dir() -> std::string = 0;
    virtual auto toolchain_dir() -> std::string = 0;
};

/// Platform is the root abstraction for all OS-specific operations.
class Platform {
public:
    virtual ~Platform() = default;

    virtual auto name() const -> std::string = 0;
    virtual auto disk_image() -> DiskImageManager& = 0;
    virtual auto packages() -> PackageManager& = 0;
    virtual auto paths() -> PathProvider& = 0;
    virtual void set_executor(executor::Executor* exec) = 0;
};

/// Create the platform implementation for the current OS.
auto create_platform(executor::Executor* exec = nullptr) -> std::unique_ptr<Platform>;

}  // namespace elmos::infra::platform
