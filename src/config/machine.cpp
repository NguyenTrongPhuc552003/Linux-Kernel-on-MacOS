// ============================================================================
// config/machine.cpp — Machine definition implementation
// ============================================================================

#include "machine.hpp"

namespace elmos::config {

auto MachineDefinition::validate() const -> VoidResult {
    if (name.empty()) {
        return make_error(Error::config("machine definition: name is required"));
    }
    if (kernel.arch.empty()) {
        return make_error(
            Error::config("machine definition " + name + ": kernel.arch is required"));
    }
    if (!is_valid_arch(kernel.arch)) {
        return make_error(Error::config("machine definition " + name +
                                        ": unsupported architecture " + kernel.arch));
    }
    if (bootloader.type.empty()) {
        return make_error(
            Error::config("machine definition " + name + ": bootloader.type is required"));
    }
    if (bootloader.type != "u-boot" && bootloader.type != "barebox") {
        return make_error(Error::config("machine definition " + name +
                                        ": unsupported bootloader type " + bootloader.type));
    }
    if (qemu.system.empty()) {
        return make_error(
            Error::config("machine definition " + name + ": qemu.system is required"));
    }
    return {};
}

auto MachineDefinition::to_string() const -> std::string {
    return name + " (" + kernel.arch + ") by " + manufacturer;
}

auto MachineDefinition::get_arch_config() const -> const ArchConfig* {
    return config::get_arch_config(kernel.arch);
}

auto default_machine() -> MachineDefinition {
    return MachineDefinition{
        .name = "rock5b_plus",
        .manufacturer = "Radxa",
        .description = "Radxa ROCK 5B+ with Rockchip RK3588",
        .tags = {"arm64", "armv8", "rockchip", "rk3588", "production-ready"},
        .kernel =
            {
                .arch = "arm64",
                .defconfig = "rock5b_defconfig",
                .device_trees = {"rk3588-rock-5b.dtb"},
            },
        .bootloader =
            {
                .type = "u-boot",
                .repo = "https://github.com/u-boot/u-boot.git",
                .version = "v2024.01",
                .defconfig = "rock5b_plus_defconfig",
                .binary = "u-boot.itb",
                .firmware =
                    {
                        {.name = "DDR Initialization",
                         .src = "rk3588_ddr.bin",
                         .blob_name = "rk3588_ddr.bin",
                         .required = true},
                        {.name = "Miniloader",
                         .src = "rk3588_miniloader.elf",
                         .blob_name = "rk3588_miniloader.elf",
                         .required = true},
                    },
                .flash = {.method = "rkdevtool", .offset_mb = 0, .device_pattern = "/dev/sd*"},
            },
        .qemu =
            {
                .system = "qemu-system-aarch64",
                .machine = "virt",
                .cpu = "cortex-a72",
                .memory = "2G",
                .network = "user",
                .console = "ttyAMA0",
                .smp = 2,
            },
        .rootfs =
            {
                .extra_packages = {"u-boot-tools", "openssh-server"},
                .kernel_modules = {"rk_crypto"},
            },
    };
}

}  // namespace elmos::config
