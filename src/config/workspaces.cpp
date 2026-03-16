// ============================================================================
// config/workspaces.cpp — Workspace management implementation
// ============================================================================

#include "workspaces.hpp"

#include <algorithm>
#include <cstdlib>
#include <fstream>

#if defined(ELMOS_PLATFORM_DARWIN) || defined(ELMOS_PLATFORM_LINUX)
#include <sys/types.h>
#include <unistd.h>

#include <pwd.h>
#endif

namespace elmos::config {

namespace fs = std::filesystem;

namespace {

auto resolve_registry_home() -> std::string {
#if defined(ELMOS_PLATFORM_DARWIN) || defined(ELMOS_PLATFORM_LINUX)
    // When running under sudo, keep workspace registry tied to the invoking user.
    if (const char* sudo_user = std::getenv("SUDO_USER"); sudo_user && *sudo_user) {
        if (auto* pw = ::getpwnam(sudo_user); pw && pw->pw_dir && *pw->pw_dir) {
            return pw->pw_dir;
        }
    }

    if (auto* pw = ::getpwuid(::getuid()); pw && pw->pw_dir && *pw->pw_dir) {
        return pw->pw_dir;
    }
#endif

    if (const char* home = std::getenv("HOME"); home && *home) {
        return home;
    }

    return "/tmp";
}

}  // namespace

WorkspaceManager::WorkspaceManager(const std::string& root_path)
    : root_path_(root_path.empty() ? "." : root_path) {}

auto WorkspaceManager::get_root() const -> std::string {
    return root_path_;
}
auto WorkspaceManager::get_elmos_dir() const -> std::string {
    return (fs::path(root_path_) / ".elmos").string();
}
auto WorkspaceManager::get_config_dir() const -> std::string {
    return (fs::path(get_elmos_dir()) / "config").string();
}
auto WorkspaceManager::get_cache_dir() const -> std::string {
    return (fs::path(get_elmos_dir()) / "build-cache").string();
}
auto WorkspaceManager::get_machine_dir() const -> std::string {
    return (fs::path(root_path_) / "machine").string();
}
auto WorkspaceManager::get_machine_def_path(const std::string& machine_id) const -> std::string {
    return (fs::path(get_machine_dir()) / (machine_id + ".yml")).string();
}
auto WorkspaceManager::get_plugins_dir() const -> std::string {
    return (fs::path(root_path_) / "plugins").string();
}
auto WorkspaceManager::get_default_config_path() const -> std::string {
    return (fs::path(root_path_) / "default.yml").string();
}
auto WorkspaceManager::get_plugin_config_path() const -> std::string {
    return (fs::path(get_elmos_dir()) / "plugins.yml").string();
}
auto WorkspaceManager::get_cache_path(const std::string& fingerprint) const -> std::string {
    return (fs::path(get_cache_dir()) / fingerprint).string();
}
auto WorkspaceManager::get_cache_index_path() const -> std::string {
    return (fs::path(get_cache_dir()) / "index.json").string();
}

auto WorkspaceManager::initialize() -> VoidResult {
    std::error_code ec;
    for (const auto& dir : {get_elmos_dir(), get_config_dir(), get_cache_dir(), get_machine_dir(),
                            get_plugins_dir()}) {
        fs::create_directories(dir, ec);
        if (ec) {
            return make_error(
                Error::config("failed to create directory " + dir + ": " + ec.message()));
        }
    }
    return {};
}

auto WorkspaceManager::exists() const -> bool {
    std::error_code ec;
    return fs::is_directory(get_elmos_dir(), ec);
}

auto WorkspaceManager::ensure_initialized() -> VoidResult {
    if (!exists()) {
        return make_error(Error::config("workspace not initialized at " + root_path_));
    }
    for (const auto& dir : {get_elmos_dir(), get_cache_dir()}) {
        std::error_code ec;
        if (!fs::is_directory(dir, ec)) {
            return make_error(Error::config("workspace directory missing: " + dir));
        }
    }
    return {};
}

auto WorkspaceManager::clean_cache() -> VoidResult {
    std::error_code ec;
    fs::remove_all(get_cache_dir(), ec);
    if (ec) {
        return make_error(Error::config("failed to clean cache: " + ec.message()));
    }
    fs::create_directories(get_cache_dir(), ec);
    return {};
}

auto WorkspaceManager::query_machines() -> Result<std::vector<std::string>> {
    auto machine_dir = get_machine_dir();
    std::error_code ec;
    if (!fs::exists(machine_dir, ec))
        return std::vector<std::string>{};

    std::vector<std::string> machines;
    for (const auto& entry : fs::directory_iterator(machine_dir, ec)) {
        if (!entry.is_directory() && entry.path().extension() == ".yml") {
            machines.push_back(entry.path().stem().string());
        }
    }
    if (ec) {
        return make_error(Error::config("failed to read machine directory: " + ec.message()));
    }
    return machines;
}

auto WorkspaceManager::find_workspace_root() -> Result<std::string> {
    std::error_code ec;
    auto cwd = fs::current_path(ec);
    if (ec) {
        return make_error(Error::config("cannot get working directory: " + ec.message()));
    }

    auto current = cwd;
    while (true) {
        if (fs::is_directory(current / ".elmos", ec)) {
            return current.string();
        }
        auto parent = current.parent_path();
        if (parent == current)
            break;
        current = parent;
    }

    return make_error(
        Error::config("workspace root (.elmos/) not found in current directory or any parent"));
}

// ── Global workspace registry ───────────────────────────────────────────────

auto WorkspaceManager::global_elmos_dir() -> std::string {
    return resolve_registry_home() + "/.elmos";
}

auto WorkspaceManager::workspace_config_path(const std::string& name) -> std::string {
    return global_elmos_dir() + "/workspaces/" + name + "/config.yaml";
}

auto WorkspaceManager::get_active_workspace() -> Result<std::string> {
    auto path = global_elmos_dir() + "/active";
    std::ifstream f(path);
    if (!f)
        return make_error(Error::config("no active workspace — run 'elmos init <name>' first"));
    std::string name;
    std::getline(f, name);
    while (!name.empty() && (name.back() == '\n' || name.back() == '\r' || name.back() == ' '))
        name.pop_back();
    if (name.empty())
        return make_error(Error::config("no active workspace — run 'elmos init <name>' first"));
    return name;
}

auto WorkspaceManager::set_active_workspace(const std::string& name) -> VoidResult {
    auto dir = global_elmos_dir() + "/workspaces";
    std::error_code ec;
    fs::create_directories(dir, ec);

    auto cfg_path = workspace_config_path(name);
    if (!fs::exists(cfg_path)) {
        return make_error(
            Error::config("workspace '" + name + "' not found — run 'elmos init " + name + "'"));
    }

    auto path = global_elmos_dir() + "/active";
    std::ofstream f(path, std::ios::trunc);
    if (!f)
        return make_error(Error::config("cannot write " + path));
    f << name;
    return {};
}

auto WorkspaceManager::workspace_dir(const std::string& name) -> std::string {
    return global_elmos_dir() + "/workspaces/" + name;
}

auto WorkspaceManager::list_workspaces() -> Result<std::vector<std::string>> {
    auto dir = global_elmos_dir() + "/workspaces";
    std::error_code ec;
    if (!fs::is_directory(dir, ec))
        return std::vector<std::string>{};

    std::vector<std::string> names;
    for (const auto& entry : fs::directory_iterator(dir, ec)) {
        // Each workspace is a subdirectory containing config.yaml
        if (entry.is_directory() && fs::exists(entry.path() / "config.yaml", ec))
            names.push_back(entry.path().filename().string());
    }
    std::sort(names.begin(), names.end());
    return names;
}

}  // namespace elmos::config
