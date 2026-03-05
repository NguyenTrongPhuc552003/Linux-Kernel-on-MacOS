#pragma once
// ============================================================================
// platform/darwin.hpp — macOS platform implementation
// ============================================================================

#include <infra/executor/interface.hpp>
#include <infra/homebrew/resolver.hpp>

#include "interface.hpp"

#include <memory>

#ifdef ELMOS_PLATFORM_DARWIN

namespace elmos::infra::platform {

class DarwinDiskImage final : public DiskImageManager {
public:
    explicit DarwinDiskImage(executor::Executor* exec) : exec_(exec) {}

    auto create(std::stop_token token, const std::string& image_path, int size_gb)
        -> VoidResult override;
    auto mount(std::stop_token token, const std::string& image_path)
        -> Result<std::string> override;
    auto unmount(std::stop_token token, const std::string& mount_point) -> VoidResult override;
    auto is_mounted(std::stop_token token, const std::string& name) -> Result<MountStatus> override;

    void set_executor(executor::Executor* exec) { exec_ = exec; }

private:
    executor::Executor* exec_;
    static auto parse_mount_point(const std::string& output) -> std::string;
    static auto parse_mounted_volume(const std::string& output, const std::string& name)
        -> std::string;
};

class DarwinPackages final : public PackageManager {
public:
    DarwinPackages(executor::Executor* exec, std::shared_ptr<homebrew::Resolver> resolver)
        : exec_(exec), resolver_(std::move(resolver)) {}

    auto is_installed(const std::string& pkg) -> bool override;
    auto install(std::stop_token token, const std::string& pkg) -> VoidResult override;
    auto list_installed() -> Result<std::vector<std::string>> override;
    auto get_bin_path(const std::string& pkg) -> std::string override;
    auto get_lib_path(const std::string& pkg) -> std::string override;
    auto get_include_path(const std::string& pkg) -> std::string override;

    void set_executor(executor::Executor* exec) { exec_ = exec; }

private:
    executor::Executor* exec_;
    std::shared_ptr<homebrew::Resolver> resolver_;
};

class DarwinPaths final : public PathProvider {
public:
    auto workspace_root(const std::string& name) -> std::string override;
    auto cache_dir() -> std::string override;
    auto toolchain_dir() -> std::string override;
};

class DarwinPlatform final : public Platform {
public:
    explicit DarwinPlatform(executor::Executor* exec);

    auto name() const -> std::string override { return "darwin"; }
    auto disk_image() -> DiskImageManager& override { return *disk_img_; }
    auto packages() -> PackageManager& override { return *packages_; }
    auto paths() -> PathProvider& override { return paths_; }
    void set_executor(executor::Executor* exec) override;

private:
    executor::Executor* exec_;
    std::unique_ptr<DarwinDiskImage> disk_img_;
    std::unique_ptr<DarwinPackages> packages_;
    DarwinPaths paths_;
    std::shared_ptr<homebrew::Resolver> resolver_;
};

}  // namespace elmos::infra::platform

#endif  // ELMOS_PLATFORM_DARWIN
