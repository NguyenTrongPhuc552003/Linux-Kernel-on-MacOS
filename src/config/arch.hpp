#pragma once
// ============================================================================
// config/arch.hpp — Architecture definitions replacing Go's core/config/arch.go
// ============================================================================

#include <string>
#include <unordered_map>
#include <vector>

namespace elmos::config {

/// Architecture-specific settings for building and emulation.
struct ArchConfig {
    std::string name;
    std::string kernel_arch;
    std::string kernel_image;
    std::vector<std::string> default_targets;

    std::string qemu_binary;
    std::string qemu_machine;
    std::string qemu_cpu;
    std::string qemu_bios;
    std::string console;

    std::string gcc_binary;
    std::string gdb_binary;
    std::string toolchain_pkg;
};

auto get_arch_config(const std::string& arch) -> const ArchConfig*;
auto supported_architectures() -> std::vector<std::string>;
auto is_valid_arch(const std::string& arch) -> bool;

/// Returns the static architecture registry.
auto architectures() -> const std::unordered_map<std::string, ArchConfig>&;

}  // namespace elmos::config
