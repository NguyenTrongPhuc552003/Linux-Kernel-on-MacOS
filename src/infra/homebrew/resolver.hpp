#pragma once
// ============================================================================
// homebrew/resolver.hpp — Homebrew path resolution (macOS only)
// Replaces Go's core/infra/homebrew/resolver.go + types.go
// ============================================================================

#include <infra/executor/interface.hpp>

#include <string>
#include <unordered_map>
#include <vector>

namespace elmos::infra::homebrew {

/// Resolves Homebrew package installation paths.
/// Used on macOS to locate tools installed via `brew`.
class Resolver {
public:
    explicit Resolver(executor::Executor* exec) : exec_(exec) {}

    auto get_prefix(const std::string& pkg) -> std::string;
    auto get_bin(const std::string& pkg) -> std::string;
    auto get_sbin(const std::string& pkg) -> std::string;
    auto get_include(const std::string& pkg) -> std::string;
    auto get_lib(const std::string& pkg) -> std::string;
    auto get_libexec_bin(const std::string& pkg) -> std::string;

    /// Build a complete environment with all GNU tools in PATH for build commands.
    auto build_gnu_tools_env() -> EnvList;

    auto list_installed() -> Result<std::vector<std::string>>;
    auto list_taps() -> Result<std::vector<std::string>>;
    auto is_installed(const std::string& pkg) -> bool;
    auto is_tapped(const std::string& tap) -> bool;

    void clear_cache();

private:
    executor::Executor* exec_;
    std::unordered_map<std::string, std::string> cache_;
};

}  // namespace elmos::infra::homebrew
