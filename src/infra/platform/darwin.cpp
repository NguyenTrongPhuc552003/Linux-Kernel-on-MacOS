// ============================================================================
// platform/darwin.cpp — macOS platform implementation
// ============================================================================

#ifdef ELMOS_PLATFORM_DARWIN

#include "darwin.hpp"

#include <cstdlib>
#include <filesystem>
#include <sstream>

namespace elmos::infra::platform {

// --- DarwinPlatform ---

DarwinPlatform::DarwinPlatform(executor::Executor* exec) : exec_(exec) {
    resolver_ = std::make_shared<homebrew::Resolver>(exec_);
    disk_img_ = std::make_unique<DarwinDiskImage>(exec_);
    packages_ = std::make_unique<DarwinPackages>(exec_, resolver_);
}

void DarwinPlatform::set_executor(executor::Executor* exec) {
    exec_ = exec;
    resolver_ = std::make_shared<homebrew::Resolver>(exec_);
    disk_img_->set_executor(exec);
    packages_ = std::make_unique<DarwinPackages>(exec_, resolver_);
}

// --- DarwinDiskImage ---

auto DarwinDiskImage::create(std::stop_token token, const std::string& image_path, int size_gb)
    -> VoidResult {
    namespace fs = std::filesystem;
    if (fs::exists(image_path))
        return {};

    auto volname = fs::path(image_path).stem().string();
    auto size_arg = std::to_string(size_gb) + "g";

    auto result = exec_->run(token, "hdiutil",
                             {"create", "-size", size_arg, "-fs", "Case-sensitive HFS+", "-type",
                              "SPARSE", "-volname", volname, image_path});

    if (!result) {
        return make_error(Error::wrap("hdiutil create " + image_path, result.error()));
    }
    return {};
}

auto DarwinDiskImage::mount(std::stop_token token, const std::string& image_path)
    -> Result<std::string> {
    auto out = exec_->output(token, "hdiutil", {"attach", "-nobrowse", image_path});
    if (!out) {
        return make_error(Error::wrap("hdiutil attach " + image_path, out.error()));
    }
    auto mp = parse_mount_point(*out);
    if (mp.empty()) {
        return make_error(Error::image("cannot parse mount point from hdiutil output"));
    }
    return mp;
}

auto DarwinDiskImage::unmount(std::stop_token token, const std::string& mount_point) -> VoidResult {
    return exec_->run(token, "hdiutil", {"detach", mount_point});
}

auto DarwinDiskImage::is_mounted(std::stop_token token, const std::string& name)
    -> Result<MountStatus> {
    auto out = exec_->output(token, "hdiutil", {"info"});
    if (!out) {
        return make_error(Error::wrap("hdiutil info", out.error()));
    }
    auto mp = parse_mounted_volume(*out, name);
    if (mp.empty())
        return MountStatus{false, ""};
    return MountStatus{true, mp};
}

auto DarwinDiskImage::parse_mount_point(const std::string& output) -> std::string {
    std::istringstream stream(output);
    std::string line;
    while (std::getline(stream, line)) {
        auto pos = line.find("/Volumes/");
        if (pos != std::string::npos) {
            auto result = line.substr(pos);
            // Trim trailing whitespace
            while (!result.empty() &&
                   (result.back() == ' ' || result.back() == '\t' || result.back() == '\n')) {
                result.pop_back();
            }
            return result;
        }
    }
    return "";
}

auto DarwinDiskImage::parse_mounted_volume(const std::string& output, const std::string& name)
    -> std::string {
    std::string target = "/Volumes/" + name;
    std::istringstream stream(output);
    std::string line;
    while (std::getline(stream, line)) {
        // Trim
        while (!line.empty() && (line.back() == ' ' || line.back() == '\t'))
            line.pop_back();
        while (!line.empty() && (line.front() == ' ' || line.front() == '\t'))
            line.erase(line.begin());

        if (line.ends_with(target))
            return target;
        auto idx = line.find(target + " ");
        if (idx != std::string::npos) {
            return line.substr(idx);
        }
    }
    return "";
}

// --- DarwinPackages ---

auto DarwinPackages::is_installed(const std::string& pkg) -> bool {
    return !resolver_->get_prefix(pkg).empty();
}

auto DarwinPackages::install(std::stop_token token, const std::string& pkg) -> VoidResult {
    return exec_->run(token, "brew", {"install", pkg});
}

auto DarwinPackages::list_installed() -> Result<std::vector<std::string>> {
    std::stop_source ss;
    auto out = exec_->output(ss.get_token(), "brew", {"list", "--formula", "-1"});
    if (!out)
        return make_error(Error::wrap("brew list", out.error()));

    std::vector<std::string> result;
    std::istringstream stream(*out);
    std::string line;
    while (std::getline(stream, line)) {
        if (!line.empty())
            result.push_back(std::move(line));
    }
    return result;
}

auto DarwinPackages::get_bin_path(const std::string& pkg) -> std::string {
    return resolver_->get_bin(pkg);
}

auto DarwinPackages::get_lib_path(const std::string& pkg) -> std::string {
    return resolver_->get_lib(pkg);
}

auto DarwinPackages::get_include_path(const std::string& pkg) -> std::string {
    return resolver_->get_include(pkg);
}

auto DarwinPackages::build_gnu_environment() -> EnvList {
    return resolver_->build_gnu_tools_env();
}

// --- DarwinPaths ---

auto DarwinPaths::workspace_root(const std::string& name) -> std::string {
    return "/Volumes/" + name;
}

auto DarwinPaths::cache_dir() -> std::string {
    if (const char* home = std::getenv("HOME")) {
        return std::string(home) + "/.elmos";
    }
    return "~/.elmos";
}

auto DarwinPaths::toolchain_dir() -> std::string {
    if (const char* home = std::getenv("HOME")) {
        return std::string(home) + "/x-tools";
    }
    return "~/x-tools";
}

}  // namespace elmos::infra::platform

#endif  // ELMOS_PLATFORM_DARWIN
