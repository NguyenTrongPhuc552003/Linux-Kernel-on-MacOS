#pragma once
// ============================================================================
// domain/bsp/registry.hpp — BSP registry client for machine definitions
// ============================================================================

#include <elmos/common.hpp>

#include <config/machine.hpp>

#include <memory>
#include <string>
#include <vector>

namespace elmos::domain::bsp {

struct MachineEntry {
    std::string name;
    std::string manufacturer;
    std::string description;
    std::vector<std::string> tags;
};

/// Fetches machine definitions from a BSP registry (offline-first with cache).
class RegistryClient {
public:
    RegistryClient(const std::string& name, const std::string& base_url,
                   const std::string& cache_dir);

    auto fetch_machine(const std::string& machine_name) -> Result<config::MachineDefinition>;
    auto list_machines() -> Result<std::vector<MachineEntry>>;
    auto is_cached(const std::string& machine_name) -> bool;

private:
    std::string name_;
    std::string base_url_;
    std::string cache_dir_;

    auto load_from_cache(const std::string& machine_name) -> Result<config::MachineDefinition>;
    auto save_to_cache(const std::string& machine_name, const config::MachineDefinition& def)
        -> VoidResult;
};

/// Orchestrates multiple BSP registries.
class RegistryManager {
public:
    explicit RegistryManager(const std::string& cache_dir);

    void add_registry(const std::string& name, const std::string& base_url);
    auto find_machine(const std::string& name) -> Result<config::MachineDefinition>;
    auto list_all_machines() -> Result<std::vector<MachineEntry>>;

private:
    std::vector<std::unique_ptr<RegistryClient>> clients_;
    std::string cache_dir_;
};

}  // namespace elmos::domain::bsp
