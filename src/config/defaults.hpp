#pragma once
// ============================================================================
// config/defaults.hpp — Default configuration values
// Replaces Go's core/config/defaults.go
// ============================================================================

#include <string>
#include <string_view>
#include <vector>

namespace elmos::config {

inline constexpr std::string_view kDefaultImageSize = "40G";
inline constexpr std::string_view kDefaultVolumeName = "elmos";
inline constexpr int kMinimumImageSize = 40;
inline constexpr std::string_view kDefaultArch = "arm64";
inline constexpr std::string_view kDefaultCrossPrefix = "llvm-";
inline constexpr std::string_view kDefaultMemory = "2G";
inline constexpr int kDefaultGDBPort = 1234;
inline constexpr int kDefaultSSHPort = 2222;
inline constexpr std::string_view kDefaultDebianMirror = "http://deb.debian.org/debian";
inline constexpr std::string_view kDefaultGlibcVersion = "2.42";

struct RequiredPackage {
    std::string_view name;
    std::string_view description;
    std::string_view category;
    bool required;
};

inline const std::vector<RequiredPackage>& required_packages() {
    static const std::vector<RequiredPackage> pkgs = {
        {"llvm", "LLVM/Clang toolchain", "Build Tools", true},
        {"lld", "LLVM linker", "Build Tools", true},
        {"gnu-sed", "GNU sed (kernel requires it)", "Build Tools", true},
        {"make", "GNU make 4.0+", "Build Tools", true},
        {"libelf", "ELF library", "Build Tools", true},
        {"git", "Git version control", "Build Tools", true},
        {"qemu", "QEMU emulator", "Virtualization", true},
        {"fakeroot", "Fake root for packaging", "Build Tools", true},
        {"e2fsprogs", "ext4 filesystem tools", "Build Tools", true},
        {"wget", "File downloader", "Build Tools", false},
        {"coreutils", "GNU core utilities", "Build Tools", true},
        {"binutils", "GNU binary utilities (objcopy)", "Toolchain Dependencies", false},
        {"gcc", "GNU Compiler Collection", "Toolchain Dependencies", false},
        {"gmp", "GNU Multiple Precision library", "Toolchain Dependencies", false},
        {"mpfr", "GNU MPFR library", "Toolchain Dependencies", false},
        {"libmpc", "GNU MPC library", "Toolchain Dependencies", false},
        {"isl", "Integer Set Library", "Toolchain Dependencies", false},
        {"texinfo", "GNU documentation system", "Toolchain Dependencies", false},
        {"bison", "Parser generator", "Toolchain Dependencies", false},
        {"gawk", "GNU AWK", "Toolchain Dependencies", false},
        {"autoconf", "Autoconf for ct-ng bootstrap", "Toolchain Dependencies", false},
        {"automake", "Automake for ct-ng bootstrap", "Toolchain Dependencies", false},
        {"libtool", "GNU Libtool", "Toolchain Dependencies", false},
        {"ncurses", "Terminal UI library (menuconfig)", "Toolchain Dependencies", false},
        {"xz", "XZ compression", "Toolchain Dependencies", false},
    };
    return pkgs;
}

inline const std::vector<std::string_view>& required_headers() {
    static const std::vector<std::string_view> hdrs = {"elf.h", "byteswap.h"};
    return hdrs;
}

inline const std::vector<std::string_view>& kernel_config_types() {
    static const std::vector<std::string_view> types = {
        "defconfig",    "tinyconfig",     "kvm_guest.config", "menuconfig",  "xconfig",
        "nconfig",      "oldconfig",      "olddefconfig",     "allnoconfig", "allyesconfig",
        "allmodconfig", "localmodconfig", "localyesconfig",
    };
    return types;
}

}  // namespace elmos::config
