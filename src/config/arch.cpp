// ============================================================================
// config/arch.cpp — Architecture definitions
// ============================================================================

#include "arch.hpp"

namespace elmos::config {

auto architectures() -> const std::unordered_map<std::string, ArchConfig>& {
    static const std::unordered_map<std::string, ArchConfig> archs = {
        {"arm64",
         {
             .name = "arm64",
             .kernel_arch = "arm64",
             .kernel_image = "Image",
             .default_targets = {"Image", "dtbs", "modules"},
             .qemu_binary = "qemu-system-aarch64",
             .qemu_machine = "virt",
             .qemu_cpu = "cortex-a72",
             .qemu_bios = "",
             .console = "ttyAMA0",
             .gcc_binary = "aarch64-unknown-linux-gnu-gcc",
             .gdb_binary = "aarch64-unknown-linux-gnu-gdb",
             .toolchain_pkg = "",
         }},
        {"arm",
         {
             .name = "arm",
             .kernel_arch = "arm",
             .kernel_image = "zImage",
             .default_targets = {"zImage", "dtbs", "modules"},
             .qemu_binary = "qemu-system-arm",
             .qemu_machine = "virt,highmem=off",
             .qemu_cpu = "cortex-a15",
             .qemu_bios = "",
             .console = "ttyAMA0",
             .gcc_binary = "arm-cortex_a15-linux-gnueabihf-gcc",
             .gdb_binary = "arm-cortex_a15-linux-gnueabihf-gdb",
             .toolchain_pkg = "",
         }},
        {"riscv",
         {
             .name = "riscv",
             .kernel_arch = "riscv",
             .kernel_image = "Image",
             .default_targets = {"Image", "dtbs", "modules"},
             .qemu_binary = "qemu-system-riscv64",
             .qemu_machine = "virt",
             .qemu_cpu = "rv64",
             .qemu_bios = "-bios default",
             .console = "ttyS0",
             .gcc_binary = "riscv64-unknown-linux-gnu-gcc",
             .gdb_binary = "riscv64-unknown-linux-gnu-gdb",
             .toolchain_pkg = "",
         }},
    };
    return archs;
}

auto get_arch_config(const std::string& arch) -> const ArchConfig* {
    const auto& archs = architectures();
    auto it = archs.find(arch);
    if (it == archs.end())
        return nullptr;
    return &it->second;
}

auto supported_architectures() -> std::vector<std::string> {
    std::vector<std::string> result;
    for (const auto& [name, _] : architectures()) {
        result.push_back(name);
    }
    return result;
}

auto is_valid_arch(const std::string& arch) -> bool {
    return architectures().contains(arch);
}

}  // namespace elmos::config
