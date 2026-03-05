#pragma once
// ============================================================================
// platform/windows.hpp — Windows (WSL2) platform implementation
// ============================================================================

#include <infra/executor/interface.hpp>

#include "interface.hpp"

#ifdef ELMOS_PLATFORM_WINDOWS

namespace elmos::infra::platform {

auto windows_to_wsl_path(const std::string& win_path) -> std::string;

class WindowsDiskImage final : public DiskImageManager {
public:
    explicit WindowsDiskImage(executor::Executor* exec) : exec_(exec) {}

    auto create(std::stop_token token, const std::string& image_path, int size_gb)
        -> VoidResult override;
    auto mount(std::stop_token token, const std::string& image_path)
        -> Result<std::string> override;
    auto unmount(std::stop_token token, const std::string& mount_point) -> VoidResult override;
    auto is_mounted(std::stop_token token, const std::string& name) -> Result<MountStatus> override;

    void set_executor(executor::Executor* exec) { exec_ = exec; }

private:
    executor::Executor* exec_;
};

class WindowsPackages final : public PackageManager {
public:
    explicit WindowsPackages(executor::Executor* exec) : exec_(exec) {}

    auto is_installed(const std::string& pkg) -> bool override;
    auto install(std::stop_token token, const std::string& pkg) -> VoidResult override;
    auto list_installed() -> Result<std::vector<std::string>> override;
    auto get_bin_path(const std::string& pkg) -> std::string override;
    auto get_lib_path(const std::string& pkg) -> std::string override;
    auto get_include_path(const std::string& pkg) -> std::string override;

    void set_executor(executor::Executor* exec) { exec_ = exec; }

private:
    executor::Executor* exec_;
};

class WindowsPaths final : public PathProvider {
public:
    auto workspace_root(const std::string& name) -> std::string override;
    auto cache_dir() -> std::string override;
    auto toolchain_dir() -> std::string override;
};

class WindowsPlatform final : public Platform {
public:
    explicit WindowsPlatform(executor::Executor* exec);

    auto name() const -> std::string override { return "windows-wsl2"; }
    auto disk_image() -> DiskImageManager& override { return *disk_img_; }
    auto packages() -> PackageManager& override { return *packages_; }
    auto paths() -> PathProvider& override { return paths_; }
    void set_executor(executor::Executor* exec) override;

private:
    executor::Executor* exec_;
    std::unique_ptr<WindowsDiskImage> disk_img_;
    std::unique_ptr<WindowsPackages> packages_;
    WindowsPaths paths_;
};

}  // namespace elmos::infra::platform

#endif  // ELMOS_PLATFORM_WINDOWS
