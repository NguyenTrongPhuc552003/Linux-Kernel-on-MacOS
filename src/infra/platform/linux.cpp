// ============================================================================
// platform/linux.cpp — Linux platform implementation
// ============================================================================

#ifdef ELMOS_PLATFORM_LINUX

#include "linux.hpp"

#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <unordered_map>

namespace elmos::infra::platform {

namespace fs = std::filesystem;

// --- Package mapping: canonical name -> distro-specific name ---
struct PkgMapping {
    std::string debian;
    std::string fedora;
    std::string alpine;
    std::string arch;
};

// NOLINTNEXTLINE(cert-err58-cpp) — static init is safe for const data
static const std::unordered_map<std::string, PkgMapping> kLinuxPackageMap = {
    {"aarch64-elf-gcc",
     {"gcc-aarch64-linux-gnu", "gcc-aarch64-linux-gnu", "aarch64-linux-musl-cross", ""}},
    {"arm-none-eabi-gcc", {"gcc-arm-none-eabi", "arm-none-eabi-gcc", "arm-none-eabi-gcc", ""}},
    {"dtc", {"device-tree-compiler", "dtc", "dtc", "dtc"}},
    {"qemu", {"qemu-system-arm", "qemu-system-arm", "qemu-system-arm", "qemu"}},
    {"u-boot-tools", {"u-boot-tools", "uboot-tools", "u-boot-tools", "u-boot-tools"}},
    {"debootstrap", {"debootstrap", "debootstrap", "debootstrap", "debootstrap"}},
    {"e2fsprogs", {"e2fsprogs", "e2fsprogs", "e2fsprogs", "e2fsprogs"}},
    {"llvm", {"clang", "clang", "clang", "clang"}},
    {"lld", {"lld", "lld", "lld", "lld"}},
    {"gnu-sed", {"sed", "sed", "sed", "sed"}},
    {"make", {"make", "make", "make", "make"}},
    {"libelf", {"libelf-dev", "elfutils-libelf-devel", "elfutils-dev", "libelf"}},
    {"git", {"git", "git", "git", "git"}},
    {"fakeroot", {"fakeroot", "fakeroot", "fakeroot", "fakeroot"}},
    {"wget", {"wget", "wget", "wget", "wget"}},
    {"coreutils", {"coreutils", "coreutils", "coreutils", "coreutils"}},
    {"binutils", {"binutils", "binutils", "binutils", "binutils"}},
    {"gcc", {"gcc", "gcc", "gcc", "gcc"}},
    {"gmp", {"libgmp-dev", "gmp-devel", "gmp-dev", "gmp"}},
    {"mpfr", {"libmpfr-dev", "mpfr-devel", "mpfr-dev", "mpfr"}},
    {"libmpc", {"libmpc-dev", "libmpc-devel", "mpc1-dev", "libmpc"}},
    {"isl", {"libisl-dev", "isl-devel", "isl-dev", "isl"}},
    {"texinfo", {"texinfo", "texinfo", "texinfo", "texinfo"}},
    {"bison", {"bison", "bison", "bison", "bison"}},
    {"gawk", {"gawk", "gawk", "gawk", "gawk"}},
    {"autoconf", {"autoconf", "autoconf", "autoconf", "autoconf"}},
    {"automake", {"automake", "automake", "automake", "automake"}},
    {"libtool", {"libtool", "libtool", "libtool", "libtool"}},
    {"ncurses", {"libncurses-dev", "ncurses-devel", "ncurses-dev", "ncurses"}},
    {"xz", {"xz-utils", "xz", "xz", "xz"}},
    {"go-task", {"", "", "", ""}},  // Not in distro repos
};

static auto binary_exists(const std::string& name) -> bool {
    // Check common PATH locations
    for (const auto& dir : {"/usr/bin", "/usr/local/bin", "/usr/sbin", "/bin", "/sbin"}) {
        if (fs::exists(fs::path(dir) / name))
            return true;
    }
    return false;
}

auto detect_linux_family() -> LinuxFamily {
    if (binary_exists("apt-get"))
        return LinuxFamily::Debian;
    if (binary_exists("dnf"))
        return LinuxFamily::Fedora;
    if (binary_exists("apk"))
        return LinuxFamily::Alpine;
    if (binary_exists("pacman"))
        return LinuxFamily::Arch;
    return LinuxFamily::Generic;
}

static auto is_orbstack() -> bool {
    return fs::exists("/opt/orbstack-guest");
}

// --- LinuxPlatform ---

LinuxPlatform::LinuxPlatform(executor::Executor* exec)
    : exec_(exec), family_(detect_linux_family()) {
    disk_img_ = std::make_unique<LinuxDiskImage>(exec_);
    packages_ = std::make_unique<LinuxPackages>(exec_, family_);
    orbstack_ = is_orbstack();
}

auto LinuxPlatform::name() const -> std::string {
    std::string result;
    switch (family_) {
    case LinuxFamily::Debian:
        result = "linux-debian";
        break;
    case LinuxFamily::Fedora:
        result = "linux-fedora";
        break;
    case LinuxFamily::Alpine:
        result = "linux-alpine";
        break;
    case LinuxFamily::Arch:
        result = "linux-arch";
        break;
    default:
        result = "linux";
        break;
    }
    if (orbstack_)
        result += "-orbstack";
    return result;
}

void LinuxPlatform::set_executor(executor::Executor* exec) {
    exec_ = exec;
    disk_img_->set_executor(exec);
    packages_->set_executor(exec);
}

// --- LinuxDiskImage ---

auto LinuxDiskImage::create(std::stop_token token, const std::string& image_path, int size_gb)
    -> VoidResult {
    if (fs::exists(image_path))
        return {};

    // Ensure parent directory exists
    std::error_code ec;
    fs::create_directories(fs::path(image_path).parent_path(), ec);
    if (ec) {
        return make_error(Error::image("create parent dir: " + ec.message()));
    }

    auto size_arg = std::to_string(size_gb) + "G";
    auto r1 = exec_->run(token, "truncate", {"-s", size_arg, image_path});
    if (!r1)
        return make_error(Error::wrap("linux disk image: create sparse file", r1.error()));

    auto r2 = exec_->run(token, "mkfs.ext4", {"-F", image_path});
    if (!r2)
        return make_error(Error::wrap("linux disk image: format ext4", r2.error()));

    return {};
}

auto LinuxDiskImage::mount(std::stop_token token, const std::string& image_path)
    -> Result<std::string> {
    auto out = exec_->output(token, "sudo", {"losetup", "--find", "--show", image_path});
    if (!out)
        return make_error(Error::wrap("losetup --find", out.error()));

    auto loop_dev = *out;
    // Trim whitespace
    while (!loop_dev.empty() && (loop_dev.back() == '\n' || loop_dev.back() == ' ')) {
        loop_dev.pop_back();
    }

    auto name = fs::path(image_path).stem().string();
    auto mount_point = "/mnt/" + name;

    auto r1 = exec_->run(token, "sudo", {"mkdir", "-p", mount_point});
    if (!r1)
        return make_error(Error::wrap("mkdir " + mount_point, r1.error()));

    auto r2 = exec_->run(token, "sudo", {"mount", loop_dev, mount_point});
    if (!r2)
        return make_error(Error::wrap("mount " + loop_dev, r2.error()));

    // Make writable by current user
    if (const char* user = std::getenv("USER")) {
        std::string ownership = std::string(user) + ":" + user;
        exec_->run(token, "sudo", {"chown", ownership, mount_point});
    }

    return mount_point;
}

auto LinuxDiskImage::unmount(std::stop_token token, const std::string& mount_point) -> VoidResult {
    auto loop_dev = find_loop_device(mount_point);

    auto r1 = exec_->run(token, "sudo", {"umount", mount_point});
    if (!r1)
        return make_error(Error::wrap("umount " + mount_point, r1.error()));

    if (!loop_dev.empty()) {
        exec_->run(token, "sudo", {"losetup", "-d", loop_dev});
    }

    exec_->run(token, "sudo", {"rmdir", mount_point});
    return {};
}

auto LinuxDiskImage::is_mounted(std::stop_token /*token*/, const std::string& name)
    -> Result<MountStatus> {
    std::ifstream mounts("/proc/mounts");
    if (!mounts) {
        return make_error(Error::image("cannot read /proc/mounts"));
    }

    auto target = "/mnt/" + name;
    std::string line;
    while (std::getline(mounts, line)) {
        std::istringstream iss(line);
        std::string dev, mp;
        iss >> dev >> mp;
        if (mp == target) {
            return MountStatus{true, target};
        }
    }
    return MountStatus{false, ""};
}

auto LinuxDiskImage::find_loop_device(const std::string& mount_point) -> std::string {
    std::ifstream mounts("/proc/mounts");
    if (!mounts)
        return "";

    std::string line;
    while (std::getline(mounts, line)) {
        std::istringstream iss(line);
        std::string dev, mp;
        iss >> dev >> mp;
        if (mp == mount_point && dev.starts_with("/dev/loop")) {
            return dev;
        }
    }
    return "";
}

// --- LinuxPackages ---

auto LinuxPackages::resolve_package_name(const std::string& pkg) -> std::string {
    auto it = kLinuxPackageMap.find(pkg);
    if (it == kLinuxPackageMap.end())
        return pkg;
    const auto& m = it->second;
    if (m.debian.empty() && m.fedora.empty() && m.alpine.empty() && m.arch.empty()) {
        return "";  // No distro package
    }
    switch (family_) {
    case LinuxFamily::Debian:
        return m.debian.empty() ? pkg : m.debian;
    case LinuxFamily::Fedora:
        return m.fedora.empty() ? pkg : m.fedora;
    case LinuxFamily::Alpine:
        return m.alpine.empty() ? pkg : m.alpine;
    case LinuxFamily::Arch:
        return m.arch.empty() ? pkg : m.arch;
    default:
        return pkg;
    }
}

auto LinuxPackages::is_installed(const std::string& pkg) -> bool {
    auto native = resolve_package_name(pkg);
    if (native.empty())
        return binary_exists(pkg);
    return is_single_package_installed(native);
}

auto LinuxPackages::is_single_package_installed(const std::string& native) -> bool {
    std::stop_source ss;
    auto token = ss.get_token();
    switch (family_) {
    case LinuxFamily::Debian: {
        auto out = exec_->output(token, "dpkg", {"-s", native});
        return out.has_value() && out->find("Status: install ok installed") != std::string::npos;
    }
    case LinuxFamily::Fedora:
        return exec_->run(token, "rpm", {"-q", native}).has_value();
    case LinuxFamily::Alpine:
        return exec_->run(token, "apk", {"info", "-e", native}).has_value();
    case LinuxFamily::Arch:
        return exec_->run(token, "pacman", {"-Q", native}).has_value();
    default:
        return binary_exists(native);
    }
}

auto LinuxPackages::install(std::stop_token token, const std::string& pkg) -> VoidResult {
    auto native = resolve_package_name(pkg);
    if (native.empty()) {
        return make_error(
            Error::dependency("package \"" + pkg + "\" has no distro mapping; install manually"));
    }
    switch (family_) {
    case LinuxFamily::Debian:
        return exec_->run(token, "sudo", {"apt-get", "install", "-y", native});
    case LinuxFamily::Fedora:
        return exec_->run(token, "sudo", {"dnf", "install", "-y", native});
    case LinuxFamily::Alpine:
        return exec_->run(token, "sudo", {"apk", "add", "--no-cache", native});
    case LinuxFamily::Arch:
        return exec_->run(token, "sudo", {"pacman", "-S", "--noconfirm", native});
    default:
        return make_error(Error::platform("package install not supported on this Linux variant"));
    }
}

auto LinuxPackages::list_installed() -> Result<std::vector<std::string>> {
    std::stop_source ss;
    auto token = ss.get_token();
    Result<std::string> out = make_error(Error::platform("listing packages not supported"));

    switch (family_) {
    case LinuxFamily::Debian:
        out = exec_->output(token, "dpkg", {"--get-selections"});
        break;
    case LinuxFamily::Fedora:
        out = exec_->output(token, "rpm", {"-qa", "--queryformat", "%{NAME}\n"});
        break;
    case LinuxFamily::Alpine:
        out = exec_->output(token, "apk", {"list", "--installed"});
        break;
    default:
        return make_error(Error::platform("listing packages not supported on this Linux variant"));
    }

    if (!out)
        return make_error(out.error());

    std::vector<std::string> result;
    std::istringstream stream(*out);
    std::string line;
    while (std::getline(stream, line)) {
        if (!line.empty()) {
            // Extract first field
            auto space = line.find_first_of(" \t");
            result.push_back(space != std::string::npos ? line.substr(0, space) : line);
        }
    }
    return result;
}

auto LinuxPackages::get_bin_path(const std::string& pkg) -> std::string {
    auto native = resolve_package_name(pkg);
    for (const auto& dir : {"/usr/bin", "/usr/local/bin", "/usr/sbin", "/bin", "/sbin"}) {
        if (fs::exists(fs::path(dir) / native)) {
            return dir;
        }
    }
    return "";
}

auto LinuxPackages::get_lib_path(const std::string& pkg) -> std::string {
    for (const auto& candidate :
         {"/usr/lib", "/usr/lib/x86_64-linux-gnu", "/usr/lib/aarch64-linux-gnu"}) {
        if (fs::exists(fs::path(candidate) / pkg)) {
            return candidate;
        }
    }
    return "/usr/lib";
}

auto LinuxPackages::get_include_path(const std::string& /*pkg*/) -> std::string {
    return "/usr/include";
}

// --- LinuxPaths ---

auto LinuxPaths::workspace_root(const std::string& name) -> std::string {
    return "/mnt/" + name;
}

auto LinuxPaths::cache_dir() -> std::string {
    if (const char* home = std::getenv("HOME")) {
        return std::string(home) + "/.elmos";
    }
    return "~/.elmos";
}

auto LinuxPaths::toolchain_dir() -> std::string {
    if (const char* home = std::getenv("HOME")) {
        return std::string(home) + "/x-tools";
    }
    return "~/x-tools";
}

}  // namespace elmos::infra::platform

#endif  // ELMOS_PLATFORM_LINUX
