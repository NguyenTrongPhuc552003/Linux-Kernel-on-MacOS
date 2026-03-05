#pragma once
// ============================================================================
// config/machine.hpp — Machine definition types
// Replaces Go's core/config/machine.go
// ============================================================================

#include <elmos/common.hpp>

#include "arch.hpp"

#include <string>
#include <vector>

namespace elmos::config {

struct FirmwareBlob {
    std::string name;
    std::string src;
    std::string blob_name;
    std::string checksum;
    bool required = false;
};

struct FlashConfig {
    std::string method;
    int offset_mb = 0;
    std::string device_pattern;
    std::string pre_flash_cmd;
    std::string post_flash_cmd;
};

struct MachineKernelConfig {
    std::string arch;
    std::string defconfig;
    std::vector<std::string> device_trees;
    std::vector<std::string> config_options;
    std::vector<std::string> patches;
};

struct MachineBootloaderConfig {
    std::string type;
    std::string repo;
    std::string version;
    std::string defconfig;
    std::string binary;
    std::vector<FirmwareBlob> firmware;
    std::vector<std::string> patches;
    FlashConfig flash;
};

struct MachineQEMUConfig {
    std::string system;
    std::string machine;
    std::string cpu;
    std::string memory;
    std::string network;
    std::string console;
    int smp = 0;
    std::vector<std::string> extra_args;
};

struct DeviceNode {
    std::string path;
    std::string type;
    int major = 0;
    int minor = 0;
    std::string permissions;
};

struct MachineRootfsConfig {
    std::vector<std::string> extra_packages;
    std::vector<std::string> kernel_modules;
    std::string post_build_script;
    std::vector<DeviceNode> devices;
};

struct MachineDefinition {
    std::string name;
    std::string manufacturer;
    std::string description;
    std::vector<std::string> tags;

    MachineKernelConfig kernel;
    MachineBootloaderConfig bootloader;
    MachineQEMUConfig qemu;
    MachineRootfsConfig rootfs;

    auto validate() const -> VoidResult;
    auto to_string() const -> std::string;
    auto get_arch_config() const -> const ArchConfig*;
};

/// Returns the default rock5b_plus machine definition.
auto default_machine() -> MachineDefinition;

}  // namespace elmos::config
