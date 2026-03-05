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
