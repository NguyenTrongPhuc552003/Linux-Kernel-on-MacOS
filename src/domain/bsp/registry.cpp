// ============================================================================
// domain/bsp/registry.cpp — BSP registry client
// ============================================================================

#include "registry.hpp"

#include <yaml-cpp/yaml.h>

#include <filesystem>
#include <fstream>
#include <sstream>

namespace elmos::domain::bsp {

namespace fs = std::filesystem;

RegistryClient::RegistryClient(const std::string& name, const std::string& base_url,
                               const std::string& cache_dir)
    : name_(name), base_url_(base_url),
      cache_dir_((fs::path(cache_dir) / "registry" / name).string()) {}

auto RegistryClient::fetch_machine(const std::string& machine_name)
    -> Result<config::MachineDefinition> {
    // Cache-first
    auto cached = load_from_cache(machine_name);
    if (cached)
        return *cached;

    // Network fetch would go here (cpp-httplib)
    return make_error(
        Error::generic("machine '" + machine_name + "' not found in registry '" + name_ + "'"));
}

auto RegistryClient::list_machines() -> Result<std::vector<MachineEntry>> {
    auto index_path = (fs::path(cache_dir_) / "index.json").string();
    if (!fs::exists(index_path))
        return std::vector<MachineEntry>{};

    // Parse cached index
    std::ifstream file(index_path);
    if (!file)
        return std::vector<MachineEntry>{};

    // Simple JSON parsing with nlohmann-json would go here
    return std::vector<MachineEntry>{};
}

auto RegistryClient::is_cached(const std::string& machine_name) -> bool {
    return fs::exists(fs::path(cache_dir_) / (machine_name + ".yml"));
}

auto RegistryClient::load_from_cache(const std::string& machine_name)
    -> Result<config::MachineDefinition> {
    auto path = (fs::path(cache_dir_) / (machine_name + ".yml")).string();
    if (!fs::exists(path)) {
        return make_error(Error::generic("cache miss for machine '" + machine_name + "'"));
    }

    try {
        auto node = YAML::LoadFile(path);
        config::MachineDefinition def;
        if (auto m = node["machine"]) {
            def.name = m["name"].as<std::string>("");
            def.manufacturer = m["manufacturer"].as<std::string>("");
            def.description = m["description"].as<std::string>("");
        }
        if (def.name.empty())
            def.name = machine_name;
        return def;
    }
    catch (const YAML::Exception& e) {
        return make_error(
            Error::config("failed to parse cached machine: " + std::string(e.what())));
    }
}

auto RegistryClient::save_to_cache(const std::string& machine_name,
                                   const config::MachineDefinition& def) -> VoidResult {
    std::error_code ec;
    fs::create_directories(cache_dir_, ec);
    if (ec)
        return make_error(Error::generic("cannot create cache dir: " + ec.message()));

    YAML::Emitter out;
    out << YAML::BeginMap;
    out << YAML::Key << "machine" << YAML::Value << YAML::BeginMap;
    out << YAML::Key << "name" << YAML::Value << def.name;
    out << YAML::Key << "manufacturer" << YAML::Value << def.manufacturer;
    out << YAML::Key << "description" << YAML::Value << def.description;
    out << YAML::EndMap;
    out << YAML::EndMap;

    auto path = (fs::path(cache_dir_) / (machine_name + ".yml")).string();
    std::ofstream file(path);
    if (!file)
        return make_error(Error::generic("cannot write cache file"));
    file << out.c_str();
    return {};
}

// RegistryManager

RegistryManager::RegistryManager(const std::string& cache_dir) : cache_dir_(cache_dir) {}

void RegistryManager::add_registry(const std::string& name, const std::string& base_url) {
    clients_.push_back(std::make_unique<RegistryClient>(name, base_url, cache_dir_));
}

auto RegistryManager::find_machine(const std::string& name) -> Result<config::MachineDefinition> {
    for (auto& client : clients_) {
        auto r = client->fetch_machine(name);
        if (r)
            return *r;
    }
    return make_error(Error::generic("machine '" + name + "' not found in any registry"));
}

auto RegistryManager::list_all_machines() -> Result<std::vector<MachineEntry>> {
    std::vector<MachineEntry> all;
    for (auto& client : clients_) {
        auto r = client->list_machines();
        if (r) {
            all.insert(all.end(), r->begin(), r->end());
        }
    }
    return all;
}

}  // namespace elmos::domain::bsp
