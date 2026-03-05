// ============================================================================
// config/saver.cpp — Configuration saving and profile management
// Replaces Go's core/config/saver.go
// ============================================================================

#include "arch.hpp"
#include "defaults.hpp"
#include "machine.hpp"
#include "types.hpp"

#include <yaml-cpp/yaml.h>

#include <filesystem>
#include <fstream>

namespace elmos::config {

auto Config::get_machine(const std::string& name) -> MachineDefinition* {
    auto it = machines.find(name);
    if (it == machines.end())
        return nullptr;
    return it->second.get();
}

auto Config::set_current_machine(std::shared_ptr<MachineDefinition> m) -> VoidResult {
    if (!m)
        return make_error(Error::config("machine definition cannot be nil"));
    auto r = m->validate();
    if (!r)
        return r;
    current_machine = std::move(m);
    build.arch = current_machine->kernel.arch;
    machine = current_machine->name;
    return {};
}

auto Config::apply_profile(const std::string& name) -> VoidResult {
    auto it = profiles.find(name);
    if (it == profiles.end()) {
        return make_error(Error::config("profile not found: " + name));
    }
    const auto& profile = it->second;
    if (!profile.arch.empty())
        build.arch = profile.arch;
    if (profile.jobs > 0)
        build.jobs = profile.jobs;
    if (!profile.memory.empty())
        qemu.memory = profile.memory;
    if (!profile.cross_compile.empty())
        build.cross_compile = profile.cross_compile;
    return {};
}

auto Config::get_arch_config() const -> const ArchConfig* {
    return config::get_arch_config(build.arch);
}

auto Config::save(const std::string& path) -> VoidResult {
    YAML::Emitter out;
    out << YAML::BeginMap;

    // Image
    out << YAML::Key << "image" << YAML::Value << YAML::BeginMap;
    if (!image.path.empty())
        out << YAML::Key << "path" << YAML::Value << image.path;
    if (!image.volume_name.empty())
        out << YAML::Key << "volume_name" << YAML::Value << image.volume_name;
    if (!image.size.empty())
        out << YAML::Key << "size" << YAML::Value << image.size;
    if (!image.mount_point.empty())
        out << YAML::Key << "mount_point" << YAML::Value << image.mount_point;
    out << YAML::EndMap;

    // Build
    out << YAML::Key << "build" << YAML::Value << YAML::BeginMap;
    out << YAML::Key << "arch" << YAML::Value << build.arch;
    out << YAML::Key << "jobs" << YAML::Value << build.jobs;
    out << YAML::Key << "llvm" << YAML::Value << build.llvm;
    if (!build.cross_compile.empty())
        out << YAML::Key << "cross_compile" << YAML::Value << build.cross_compile;
    out << YAML::Key << "verbose" << YAML::Value << build.verbose;
    out << YAML::EndMap;

    // QEMU
    out << YAML::Key << "qemu" << YAML::Value << YAML::BeginMap;
    out << YAML::Key << "memory" << YAML::Value << qemu.memory;
    out << YAML::Key << "gdb_port" << YAML::Value << qemu.gdb_port;
    out << YAML::Key << "ssh_port" << YAML::Value << qemu.ssh_port;
    out << YAML::Key << "smp" << YAML::Value << qemu.smp;
    out << YAML::EndMap;

    // Paths (only non-default values)
    out << YAML::Key << "paths" << YAML::Value << YAML::BeginMap;
    if (!paths.project_root.empty())
        out << YAML::Key << "project_root" << YAML::Value << paths.project_root;
    out << YAML::EndMap;

    out << YAML::EndMap;

    // Write to file
    namespace fs = std::filesystem;
    if (auto parent = fs::path(path).parent_path(); !parent.empty()) {
        std::error_code ec;
        fs::create_directories(parent, ec);
        if (ec) {
            return make_error(Error::config("cannot create config directory: " + ec.message()));
        }
    }

    std::ofstream file(path, std::ios::out | std::ios::trunc);
    if (!file) {
        return make_error(Error::config("cannot write config file: " + path));
    }
    file << out.c_str();
    return {};
}

}  // namespace elmos::config
