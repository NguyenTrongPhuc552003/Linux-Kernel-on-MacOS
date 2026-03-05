#pragma once
// ============================================================================
// platform/linux.hpp — Linux platform implementation
// ============================================================================

#include <infra/executor/interface.hpp>

#include "interface.hpp"

#ifdef ELMOS_PLATFORM_LINUX

namespace elmos::infra::platform {

enum class LinuxFamily {
    Debian,   // apt-get
    Fedora,   // dnf
    Alpine,   // apk
    Arch,     // pacman
    Generic,  // fallback
};

auto detect_linux_family() -> LinuxFamily;

class LinuxDiskImage final : public DiskImageManager {
public:
    explicit LinuxDiskImage(executor::Executor* exec) : exec_(exec) {}

    auto create(std::stop_token token, const std::string& image_path, int size_gb)
        -> VoidResult override;
    auto mount(std::stop_token token, const std::string& image_path)
        -> Result<std::string> override;
    auto unmount(std::stop_token token, const std::string& mount_point) -> VoidResult override;
    auto is_mounted(std::stop_token token, const std::string& name) -> Result<MountStatus> override;

    void set_executor(executor::Executor* exec) { exec_ = exec; }

private:
    executor::Executor* exec_;
    auto find_loop_device(const std::string& mount_point) -> std::string;
};

class LinuxPackages final : public PackageManager {
public:
    LinuxPackages(executor::Executor* exec, LinuxFamily family) : exec_(exec), family_(family) {}

    auto is_installed(const std::string& pkg) -> bool override;
    auto install(std::stop_token token, const std::string& pkg) -> VoidResult override;
    auto list_installed() -> Result<std::vector<std::string>> override;
    auto get_bin_path(const std::string& pkg) -> std::string override;
    auto get_lib_path(const std::string& pkg) -> std::string override;
    auto get_include_path(const std::string& pkg) -> std::string override;

    void set_executor(executor::Executor* exec) { exec_ = exec; }

private:
    executor::Executor* exec_;
    LinuxFamily family_;

    auto resolve_package_name(const std::string& pkg) -> std::string;
    auto is_single_package_installed(const std::string& native) -> bool;
};

class LinuxPaths final : public PathProvider {
public:
    auto workspace_root(const std::string& name) -> std::string override;
    auto cache_dir() -> std::string override;
    auto toolchain_dir() -> std::string override;
};

class LinuxPlatform final : public Platform {
public:
    explicit LinuxPlatform(executor::Executor* exec);

    auto name() const -> std::string override;
    auto disk_image() -> DiskImageManager& override { return *disk_img_; }
    auto packages() -> PackageManager& override { return *packages_; }
    auto paths() -> PathProvider& override { return paths_; }
    void set_executor(executor::Executor* exec) override;

    void set_orbstack(bool val) { orbstack_ = val; }

private:
    executor::Executor* exec_;
    LinuxFamily family_;
    bool orbstack_ = false;
    std::unique_ptr<LinuxDiskImage> disk_img_;
    std::unique_ptr<LinuxPackages> packages_;
    LinuxPaths paths_;
};

}  // namespace elmos::infra::platform

#endif  // ELMOS_PLATFORM_LINUX
