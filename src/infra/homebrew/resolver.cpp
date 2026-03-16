// ============================================================================
// homebrew/resolver.cpp — Homebrew path resolution implementation
// ============================================================================

#include "resolver.hpp"

#include <algorithm>
#include <filesystem>
#include <sstream>

namespace elmos::infra::homebrew {

auto Resolver::get_prefix(const std::string& pkg) -> std::string {
    if (auto it = cache_.find(pkg); it != cache_.end()) {
        return it->second;
    }

    std::stop_source ss;
    auto out = exec_->output(ss.get_token(), "brew", {"--prefix", pkg});
    if (!out)
        return "";

    auto prefix = *out;
    // Trim whitespace
    while (!prefix.empty() &&
           (prefix.back() == '\n' || prefix.back() == ' ' || prefix.back() == '\t')) {
        prefix.pop_back();
    }

    cache_[pkg] = prefix;
    return prefix;
}

auto Resolver::get_bin(const std::string& pkg) -> std::string {
    auto prefix = get_prefix(pkg);
    if (prefix.empty())
        return "";
    return (std::filesystem::path(prefix) / "bin").string();
}

auto Resolver::get_sbin(const std::string& pkg) -> std::string {
    auto prefix = get_prefix(pkg);
    if (prefix.empty())
        return "";
    return (std::filesystem::path(prefix) / "sbin").string();
}

auto Resolver::get_include(const std::string& pkg) -> std::string {
    auto prefix = get_prefix(pkg);
    if (prefix.empty())
        return "";
    return (std::filesystem::path(prefix) / "include").string();
}

auto Resolver::get_lib(const std::string& pkg) -> std::string {
    auto prefix = get_prefix(pkg);
    if (prefix.empty())
        return "";
    return (std::filesystem::path(prefix) / "lib").string();
}

auto Resolver::get_libexec_bin(const std::string& pkg) -> std::string {
    auto prefix = get_prefix(pkg);
    if (prefix.empty())
        return "";
    return (std::filesystem::path(prefix) / "libexec" / "gnubin").string();
}

auto Resolver::build_gnu_tools_env() -> EnvList {
    namespace fs = std::filesystem;

    // GNU packages that provide replacements for BSD tools via libexec/gnubin.
    // NOTE: binutils is intentionally excluded — GNU ar creates SysV-format
    // archives that Apple's linker cannot read ("invalid control bits").
    // Per ct-ng macOS docs, binutils/bin is appended (not prepended) to PATH.
    static const std::vector<std::string> gnu_packages = {
        "coreutils", "findutils", "gnu-sed", "gawk",  "grep", "make",
        "diffutils", "gnu-tar",   "gzip",    "bzip2", "xz",   "gpatch",
    };

    // Build tools that only need their bin/ on PATH
    static const std::vector<std::string> build_packages = {
        "autoconf",   "automake", "libtool",  "m4",      "bison",   "flex",
        "pkg-config", "texinfo",  "help2man", "gettext", "ncurses", "e2fsprogs",
    };

    // Per ct-ng macOS docs and Go reference (addLibraryFlags):
    // LDFLAGS packages need -L<prefix>/lib
    static const std::vector<std::string> ldflags_packages = {
        "bison",
        "flex",
        "ncurses",
        "zlib",
    };
    // CPPFLAGS packages need -I<prefix>/include
    static const std::vector<std::string> cppflags_packages = {
        "binutils",
        "flex",
        "ncurses",
        "zlib",
    };

    std::string path_prefix;

    for (const auto& pkg : gnu_packages) {
        // Prefer libexec/gnubin (provides un-prefixed GNU names)
        auto gnubin = get_libexec_bin(pkg);
        if (!gnubin.empty() && fs::is_directory(gnubin)) {
            path_prefix += gnubin + ":";
        }
        // Also add regular bin (g-prefixed names as fallback)
        auto bin = get_bin(pkg);
        if (!bin.empty() && fs::is_directory(bin)) {
            path_prefix += bin + ":";
        }
    }

    for (const auto& pkg : build_packages) {
        auto bin = get_bin(pkg);
        if (!bin.empty() && fs::is_directory(bin)) {
            path_prefix += bin + ":";
        }
        // e2fsprogs also has sbin
        if (pkg == "e2fsprogs") {
            auto sbin = get_sbin(pkg);
            if (!sbin.empty() && fs::is_directory(sbin)) {
                path_prefix += sbin + ":";
            }
        }
    }

    EnvList result;

    if (!path_prefix.empty()) {
        // Remove trailing colon
        if (path_prefix.back() == ':')
            path_prefix.pop_back();

        const char* sys_path = std::getenv("PATH");
        std::string full_path = path_prefix;
        if (sys_path) {
            full_path += ":";
            full_path += sys_path;
        }

        // Append binutils/bin at END of PATH (per ct-ng macOS docs)
        // so macOS native ar/ranlib take priority over GNU variants
        auto binutils_bin = get_bin("binutils");
        if (!binutils_bin.empty() && fs::is_directory(binutils_bin)) {
            full_path += ":" + binutils_bin;
        }

        result.push_back("PATH=" + full_path);
    }

    // Build LDFLAGS (per ct-ng macOS docs: -L for ncurses, bison, flex, zlib)
    std::string ldflags;
    for (const auto& pkg : ldflags_packages) {
        auto lib = get_lib(pkg);
        if (!lib.empty() && fs::is_directory(lib)) {
            if (!ldflags.empty())
                ldflags += " ";
            ldflags += "-L" + lib;
        }
    }
    if (!ldflags.empty()) {
        result.push_back("LDFLAGS=" + ldflags);
    }

    // Build CPPFLAGS (per ct-ng macOS docs: -I for binutils, ncurses, flex, zlib)
    std::string cppflags;
    for (const auto& pkg : cppflags_packages) {
        auto inc = get_include(pkg);
        if (!inc.empty() && fs::is_directory(inc)) {
            if (!cppflags.empty())
                cppflags += " ";
            cppflags += "-I" + inc;
        }
    }
    if (!cppflags.empty()) {
        result.push_back("CPPFLAGS=" + cppflags);
    }

    // PKG_CONFIG_PATH — add Homebrew's share/pkgconfig
    // Derive brew prefix from any known package prefix (2 levels up from opt/<pkg>)
    auto ncurses_pfx = get_prefix("ncurses");
    if (!ncurses_pfx.empty()) {
        auto brew_base = fs::path(ncurses_pfx).parent_path().parent_path();
        auto share_pkgconfig = (brew_base / "share" / "pkgconfig").string();
        if (fs::is_directory(share_pkgconfig)) {
            const char* existing = std::getenv("PKG_CONFIG_PATH");
            std::string pkg_path = share_pkgconfig;
            if (existing && existing[0] != '\0') {
                pkg_path += ":";
                pkg_path += existing;
            }
            result.push_back("PKG_CONFIG_PATH=" + pkg_path);
        }
    }

    return result;
}

auto Resolver::list_installed() -> Result<std::vector<std::string>> {
    std::stop_source ss;
    auto out = exec_->output(ss.get_token(), "brew", {"list", "--formulae"});
    if (!out)
        return make_error(out.error());

    std::vector<std::string> result;
    std::istringstream stream(*out);
    std::string line;
    while (std::getline(stream, line)) {
        if (!line.empty())
            result.push_back(std::move(line));
    }
    return result;
}

auto Resolver::list_taps() -> Result<std::vector<std::string>> {
    std::stop_source ss;
    auto out = exec_->output(ss.get_token(), "brew", {"tap"});
    if (!out)
        return make_error(out.error());

    std::vector<std::string> result;
    std::istringstream stream(*out);
    std::string line;
    while (std::getline(stream, line)) {
        if (!line.empty())
            result.push_back(std::move(line));
    }
    return result;
}

auto Resolver::is_installed(const std::string& pkg) -> bool {
    auto installed = list_installed();
    if (!installed)
        return false;
    return std::find(installed->begin(), installed->end(), pkg) != installed->end();
}

auto Resolver::is_tapped(const std::string& tap) -> bool {
    auto taps = list_taps();
    if (!taps)
        return false;
    return std::find(taps->begin(), taps->end(), tap) != taps->end();
}

void Resolver::clear_cache() {
    cache_.clear();
}

}  // namespace elmos::infra::homebrew
