#pragma once
// ============================================================================
// config/workspaces.hpp — Workspace management
// Replaces Go's core/config/workspaces.go
// ============================================================================

#include <elmos/common.hpp>

#include <filesystem>
#include <string>
#include <vector>

namespace elmos::config {

class WorkspaceManager {
public:
    explicit WorkspaceManager(const std::string& root_path = ".");

    auto get_root() const -> std::string;
    auto get_elmos_dir() const -> std::string;
    auto get_config_dir() const -> std::string;
    auto get_cache_dir() const -> std::string;
    auto get_machine_dir() const -> std::string;
    auto get_machine_def_path(const std::string& machine_id) const -> std::string;
    auto get_plugins_dir() const -> std::string;
    auto get_default_config_path() const -> std::string;
    auto get_plugin_config_path() const -> std::string;
    auto get_cache_path(const std::string& fingerprint) const -> std::string;
    auto get_cache_index_path() const -> std::string;

    auto initialize() -> VoidResult;
    auto exists() const -> bool;
    auto ensure_initialized() -> VoidResult;
    auto clean_cache() -> VoidResult;
    auto query_machines() -> Result<std::vector<std::string>>;

    static auto find_workspace_root() -> Result<std::string>;

private:
    std::string root_path_;
};

}  // namespace elmos::config
