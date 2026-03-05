// ============================================================================
// platform/windows.cpp — Windows (WSL2) platform implementation
// ============================================================================

#ifdef ELMOS_PLATFORM_WINDOWS

#include "windows.hpp"

#include <algorithm>
#include <cstdlib>
#include <filesystem>
#include <sstream>

namespace elmos::infra::platform {

auto windows_to_wsl_path(const std::string& win_path) -> std::string {
    if (win_path.size() >= 2 && win_path[1] == ':') {
        char drive = static_cast<char>(std::tolower(static_cast<unsigned char>(win_path[0])));
        auto rest = win_path.substr(2);
        std::replace(rest.begin(), rest.end(), '\\', '/');
        return "/mnt/" + std::string(1, drive) + rest;
    }
    auto result = win_path;
    std::replace(result.begin(), result.end(), '\\', '/');
    return result;
}

// --- WindowsPlatform ---

WindowsPlatform::WindowsPlatform(executor::Executor* exec) : exec_(exec) {
    disk_img_ = std::make_unique<WindowsDiskImage>(exec_);
    packages_ = std::make_unique<WindowsPackages>(exec_);
}

void WindowsPlatform::set_executor(executor::Executor* exec) {
    exec_ = exec;
    disk_img_->set_executor(exec);
    packages_->set_executor(exec);
}

// --- WindowsDiskImage ---

auto WindowsDiskImage::create(std::stop_token token, const std::string& image_path, int size_gb)
    -> VoidResult {
    auto wsl_path = windows_to_wsl_path(image_path);
    auto size_bytes = std::to_string(static_cast<int64_t>(size_gb) * 1024 * 1024 * 1024);
    return exec_->run(token, "wsl.exe", {"--", "fallocate", "-l", size_bytes, wsl_path});
}

auto WindowsDiskImage::mount(std::stop_token token, const std::string& image_path)
    -> Result<std::string> {
    auto wsl_path = windows_to_wsl_path(image_path);
    auto name = std::filesystem::path(image_path).stem().string();
    auto mount_point = "/mnt/elmos/" + name;

    auto r1 = exec_->run(token, "wsl.exe", {"--", "mkdir", "-p", mount_point});
    if (!r1)
        return make_error(Error::wrap("wsl mkdir", r1.error()));

    auto out = exec_->output(token, "wsl.exe", {"--", "losetup", "--find", "--show", wsl_path});
    if (!out)
        return make_error(Error::wrap("wsl losetup", out.error()));

    auto loop_dev = *out;
    while (!loop_dev.empty() && (loop_dev.back() == '\n' || loop_dev.back() == ' ')) {
        loop_dev.pop_back();
    }

    auto r2 = exec_->run(token, "wsl.exe", {"--", "mount", loop_dev, mount_point});
    if (!r2) {
        exec_->run(token, "wsl.exe", {"--", "losetup", "-d", loop_dev});
        return make_error(Error::wrap("wsl mount", r2.error()));
    }

    return mount_point;
}

auto WindowsDiskImage::unmount(std::stop_token token, const std::string& mount_point)
    -> VoidResult {
    auto r = exec_->run(token, "wsl.exe", {"--", "umount", mount_point});
    if (!r)
        return make_error(Error::wrap("wsl umount", r.error()));

    auto out = exec_->output(token, "wsl.exe", {"--", "losetup", "-j", mount_point});
    if (out && !out->empty()) {
        auto colon = out->find(':');
        if (colon != std::string::npos) {
            auto loop_dev = out->substr(0, colon);
            while (!loop_dev.empty() && loop_dev.front() == ' ')
                loop_dev.erase(loop_dev.begin());
            exec_->run(token, "wsl.exe", {"--", "losetup", "-d", loop_dev});
        }
    }
    return {};
}

auto WindowsDiskImage::is_mounted(std::stop_token token, const std::string& name)
    -> Result<MountStatus> {
    auto target = "/mnt/elmos/" + name;
    auto r = exec_->run(token, "wsl.exe", {"--", "grep", "-q", " " + target + " ", "/proc/mounts"});
    if (!r)
        return MountStatus{false, ""};
    return MountStatus{true, target};
}

// --- WindowsPackages ---

auto WindowsPackages::is_installed(const std::string& pkg) -> bool {
    std::stop_source ss;
    auto out = exec_->output(ss.get_token(), "wsl.exe", {"--", "dpkg", "-s", pkg});
    return out.has_value() && out->find("Status: install ok installed") != std::string::npos;
}

auto WindowsPackages::install(std::stop_token token, const std::string& pkg) -> VoidResult {
    return exec_->run(token, "wsl.exe", {"--", "apt-get", "install", "-y", pkg});
}

auto WindowsPackages::list_installed() -> Result<std::vector<std::string>> {
    std::stop_source ss;
    auto out = exec_->output(ss.get_token(), "wsl.exe", {"--", "dpkg", "--get-selections"});
    if (!out)
        return make_error(Error::wrap("wsl dpkg", out.error()));

    std::vector<std::string> result;
    std::istringstream stream(*out);
    std::string line;
    while (std::getline(stream, line)) {
        if (!line.empty()) {
            auto space = line.find_first_of(" \t");
            result.push_back(space != std::string::npos ? line.substr(0, space) : line);
        }
    }
    return result;
}

auto WindowsPackages::get_bin_path(const std::string& pkg) -> std::string {
    std::stop_source ss;
    auto out = exec_->output(ss.get_token(), "wsl.exe", {"--", "which", pkg});
    if (!out || out->empty())
        return "";
    auto path = *out;
    while (!path.empty() && (path.back() == '\n' || path.back() == ' '))
        path.pop_back();
    return std::filesystem::path(path).parent_path().string();
}

auto WindowsPackages::get_lib_path(const std::string& /*pkg*/) -> std::string {
    return "/usr/lib";
}

auto WindowsPackages::get_include_path(const std::string& /*pkg*/) -> std::string {
    return "/usr/include";
}

// --- WindowsPaths ---

auto WindowsPaths::workspace_root(const std::string& name) -> std::string {
    return "\\\\wsl$\\Ubuntu\\mnt\\elmos\\" + name;
}

auto WindowsPaths::cache_dir() -> std::string {
    if (const char* home = std::getenv("USERPROFILE")) {
        return std::string(home) + "\\.elmos";
    }
    return "~\\.elmos";
}

auto WindowsPaths::toolchain_dir() -> std::string {
    if (const char* home = std::getenv("USERPROFILE")) {
        return std::string(home) + "\\x-tools";
    }
    return "~\\x-tools";
}

}  // namespace elmos::infra::platform

#endif  // ELMOS_PLATFORM_WINDOWS
