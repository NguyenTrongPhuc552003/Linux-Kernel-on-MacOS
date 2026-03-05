#pragma once
// ============================================================================
// config/types.hpp — Configuration structs replacing Go's core/config/types.go
// ============================================================================

#include <elmos/common.hpp>

#include <map>
#include <optional>
#include <string>
#include <vector>

namespace elmos::config {

struct ArchConfig;  // Forward declaration — defined in arch.hpp

struct ImageConfig {
    std::string path;
    std::string volume_name;
    std::string size;
    std::string mount_point;
};

struct BuildConfig {
    std::string arch;
    int jobs = 0;
    bool llvm = true;
    std::string cross_compile;
    bool verbose = false;
};

struct QEMUConfig {
    std::string memory;
    int gdb_port = 0;
    int ssh_port = 0;
    int smp = 0;
};

struct PathsConfig {
    std::string project_root;
    std::string kernel_dir;
    std::string modules_dir;
    std::string apps_dir;
    std::string libraries_dir;
    std::string patches_dir;
    std::string rootfs_dir;
    std::string disk_image;
    std::string debian_mirror;
    std::string toolchains_dir;
};

struct ProfileConfig {
    std::string arch;
    int jobs = 0;
    std::string memory;
    std::string cross_compile;
};

struct BootloaderConfig {
    std::string type;
    std::string repo;
    std::string version;
    std::vector<std::string> patches;
};

struct PluginEntry {
    std::string type;
    std::string path;
    bool enabled = true;
    std::map<std::string, std::string> config;
};

struct PluginsConfig {
    std::vector<std::string> enabled;
    std::vector<std::string> disabled;
    std::map<std::string, PluginEntry> plugins;
    std::vector<std::string> search_paths;
};

// Forward declaration
struct MachineDefinition;

struct Config {
    std::string config_file;

    ImageConfig image;
    BuildConfig build;
    QEMUConfig qemu;
    PathsConfig paths;
    std::map<std::string, ProfileConfig> profiles;

    std::string machine;
    std::map<std::string, std::shared_ptr<MachineDefinition>> machines;
    std::shared_ptr<MachineDefinition> current_machine;

    BootloaderConfig bootloader;
    PluginsConfig plugins;

    auto get_machine(const std::string& name) -> MachineDefinition*;
    auto set_current_machine(std::shared_ptr<MachineDefinition> m) -> VoidResult;
    auto apply_profile(const std::string& name) -> VoidResult;

    // Defined in arch.hpp
    auto get_arch_config() const -> const ::elmos::config::ArchConfig*;

    auto save(const std::string& path) -> VoidResult;
};

}  // namespace elmos::config
