// ============================================================================
// domain/emulator/qemu.cpp — QEMU integration
// ============================================================================

#include "qemu.hpp"

#include <config/arch.hpp>
#include <config/types.hpp>
#include <context/context.hpp>

#include <algorithm>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <unordered_set>

namespace elmos::domain::emulator {

namespace fs = std::filesystem;

namespace {

auto trim(std::string s) -> std::string {
    constexpr std::string_view ws = " \t\r\n";
    const auto start = s.find_first_not_of(ws);
    if (start == std::string::npos) {
        return "";
    }
    const auto end = s.find_last_not_of(ws);
    return s.substr(start, end - start + 1);
}

auto bootloader_is_riscv_smode(const fs::path& bootloader_dir) -> bool {
    std::ifstream cfg(bootloader_dir / ".config");
    if (!cfg) {
        return false;
    }

    std::string line;
    while (std::getline(cfg, line)) {
        if (line == "CONFIG_RISCV_SMODE=y") {
            return true;
        }
    }
    return false;
}

void append_bios_args(std::vector<std::string>& args, const std::string& bios_field) {
    auto bios = trim(bios_field);
    if (bios.empty()) {
        return;
    }

    if (bios.starts_with("-bios ")) {
        bios = trim(bios.substr(6));
    }

    if (!bios.empty()) {
        args.push_back("-bios");
        args.push_back(bios);
    }
}

auto read_kernel_config(context::Context* ctx) -> std::unordered_set<std::string> {
    auto config_path = fs::path(ctx->config().paths.kernel_dir) / ".config";
    if (!ctx->fs().exists(config_path)) {
        return {};
    }

    std::ifstream in(config_path);
    if (!in) {
        return {};
    }

    std::unordered_set<std::string> options;
    std::string line;
    while (std::getline(in, line)) {
        if (line.empty() || line[0] == '#') {
            continue;
        }
        auto equals = line.find('=');
        if (equals == std::string::npos) {
            continue;
        }
        auto value = line.substr(equals + 1);
        if (value == "y") {
            options.insert(line.substr(0, equals));
        }
    }

    return options;
}

auto kernel_has_option(const std::unordered_set<std::string>& options, const std::string& name)
    -> bool {
    return options.find(name) != options.end();
}

auto default_block_device(const std::string& arch,
                          const std::unordered_set<std::string>& kernel_options) -> std::string {
    if (arch == "arm" || arch == "riscv") {
        if (kernel_has_option(kernel_options, "CONFIG_VIRTIO_MMIO")) {
            return "virtio-blk-device";
        }
        return "virtio-blk-pci";
    }
    return "virtio-blk-pci";
}

auto default_net_device(const std::string& arch,
                        const std::unordered_set<std::string>& kernel_options) -> std::string {
    if (arch == "arm" || arch == "riscv") {
        if (kernel_has_option(kernel_options, "CONFIG_VIRTIO_MMIO")) {
            return "virtio-net-device";
        }
        return "virtio-net-pci";
    }
    return "virtio-net-pci";
}

auto default_9p_device(const std::string& arch,
                       const std::unordered_set<std::string>& kernel_options) -> std::string {
    if (arch == "arm" || arch == "riscv") {
        if (kernel_has_option(kernel_options, "CONFIG_VIRTIO_MMIO")) {
            return "virtio-9p-device";
        }
        return "virtio-9p-pci";
    }
    return "virtio-9p-pci";
}

auto effective_smp_count(int configured_smp, const std::unordered_set<std::string>& kernel_options)
    -> int {
    if (kernel_has_option(kernel_options, "CONFIG_BROKEN_ON_SMP") ||
        !kernel_has_option(kernel_options, "CONFIG_SMP")) {
        return 1;
    }
    return std::max(1, configured_smp);
}

void append_9p_share_if_exists(std::vector<std::string>& args, const std::string& id,
                               const std::string& path, const std::string& tag,
                               const std::string& device) {
    if (path.empty() || !fs::is_directory(path)) {
        return;
    }

    args.push_back("-fsdev");
    args.push_back("local,id=" + id + ",path=" + path + ",security_model=none");
    args.push_back("-device");
    args.push_back(device + ",fsdev=" + id + ",mount_tag=" + tag);
}

}  // namespace

QEMURunner::QEMURunner(context::Context* ctx) : ctx_(ctx) {}

auto QEMURunner::is_available(std::stop_token token) -> bool {
    auto arch_cfg = ctx_->config().get_arch_config();
    if (!arch_cfg)
        return false;
    return ctx_->exec().look_path(arch_cfg->qemu_binary).has_value();
}

auto QEMURunner::find_bootloader() const -> std::string {
    auto& cfg = ctx_->config();
    auto bl_dir = fs::path(cfg.image.mount_point) / "bootloader";

    // Priority: u-boot.bin (flat binary) > u-boot.itb (FIT image) > u-boot (ELF)
    for (const char* name : {"u-boot.bin", "u-boot.itb", "u-boot"}) {
        auto candidate = bl_dir / name;
        if (ctx_->fs().exists(candidate)) {
            return candidate.string();
        }
    }
    return {};
}

auto QEMURunner::has_bootloader() const -> bool {
    return !find_bootloader().empty();
}

auto QEMURunner::build_command(const RunOptions& opts)
    -> Result<std::pair<std::string, std::vector<std::string>>> {
    auto& cfg = ctx_->config();
    const auto& arch = cfg.build.arch;

    auto arch_cfg = cfg.get_arch_config();
    if (!arch_cfg) {
        return make_error(Error::config("no architecture config for " + arch));
    }

    auto qemu_path = ctx_->exec().look_path(arch_cfg->qemu_binary);
    if (!qemu_path) {
        return make_error(Error::dependency("QEMU not found: " + arch_cfg->qemu_binary +
                                            " (install qemu package first)"));
    }

    const bool has_disk_image = (!cfg.paths.disk_image.empty() &&
                                 ctx_->fs().exists(cfg.paths.disk_image));

    // Determine boot mode: U-Boot or direct kernel.
    auto uboot_path = opts.force_direct_kernel ? std::string{} : find_bootloader();
    const bool use_uboot = !uboot_path.empty();

    // Direct-kernel boot always needs the kernel image and rootfs.
    if (!use_uboot) {
        auto kernel_image = ctx_->get_kernel_image();
        if (kernel_image.empty() || !ctx_->fs().exists(kernel_image)) {
            return make_error(Error::build("kernel image not found: " + kernel_image +
                                           " (run 'elmos kernel build')"));
        }
        if (!has_disk_image && opts.initrd.empty()) {
            return make_error(Error::build("rootfs disk image not found: " + cfg.paths.disk_image +
                                           " (create/populate disk image or pass --initrd)"));
        }
    }
    else if (!has_disk_image) {
        // U-Boot mode still needs a disk image (it reads the kernel from it).
        return make_error(
            Error::build("rootfs disk image not found: " + cfg.paths.disk_image +
                         " (run 'elmos rootfs build' or 'elmos qemu run' to auto-create it)"));
    }

    auto kernel_options = read_kernel_config(ctx_);

    std::vector<std::string> args;

    // ── Core hardware ──────────────────────────────────────────────────────
    args.push_back("-machine");
    args.push_back(arch_cfg->qemu_machine);
    args.push_back("-cpu");
    args.push_back(arch_cfg->qemu_cpu);
    args.push_back("-m");
    args.push_back(cfg.qemu.memory);
    args.push_back("-smp");
    args.push_back(std::to_string(effective_smp_count(cfg.qemu.smp, kernel_options)));

    // ── Boot-mode: U-Boot or direct kernel ────────────────────────────────
    if (use_uboot) {
        // U-Boot boot — architecture-specific firmware placement:
        //
        //   arm64 : -bios u-boot.bin          (ARM virt exposes a firmware slot)
        //   arm32 : -kernel u-boot            (ARM virt uses kernel slot for BL)
        //   riscv : -bios default(OpenSBI)    (M-mode)
        //           -kernel u-boot.bin        (S-mode payload handed to U-Boot)
        //
        // In all cases -append is OMITTED; U-Boot reads bootargs from its own
        // environment / extlinux.conf on the FAT boot partition.
        if (arch == "arm64") {
            args.push_back("-bios");
            args.push_back(uboot_path);
        }
        else if (arch == "arm") {
            args.push_back("-kernel");
            args.push_back(uboot_path);
        }
        else if (arch == "riscv") {
            const auto bootloader_dir = fs::path(cfg.image.mount_point) / "bootloader";
            const bool smode_uboot = bootloader_is_riscv_smode(bootloader_dir);

            if (smode_uboot) {
                // OpenSBI runs in M-mode, U-Boot runs in S-mode payload.
                append_bios_args(args, arch_cfg->qemu_bios);  // -bios default
                args.push_back("-kernel");
                args.push_back(uboot_path);
            }
            else {
                // Existing non-S-mode U-Boot configs are a common source of
                // OpenSBI handoff stalls or kernel-entry faults in this flow.
                return make_error(Error::config(
                    "RISC-V U-Boot is not configured for S-mode (missing CONFIG_RISCV_SMODE=y). "
                    "Run 'elmos bootloader config' then 'elmos bootloader build' to switch to "
                    "qemu-riscv64_smode_defconfig, or use '--force-direct-kernel'"));
            }
        }
        else {
            // Unknown arch: emit warning via direct-kernel fallback.
            return make_error(Error::config("U-Boot boot mode is not defined for architecture '" +
                                            arch +
                                            "'; use --force-direct-kernel or add arch support"));
        }
    }
    else {
        // Direct kernel boot (legacy / development mode).
        auto kernel_image = ctx_->get_kernel_image();
        args.push_back("-kernel");
        args.push_back(kernel_image);
        append_bios_args(args, arch_cfg->qemu_bios);
    }

    // ── Disk image ─────────────────────────────────────────────────────────
    if (has_disk_image) {
        args.push_back("-drive");
        args.push_back("file=" + cfg.paths.disk_image + ",format=raw,if=none,id=rootfs0");
        args.push_back("-device");
        args.push_back(default_block_device(arch, kernel_options) + ",drive=rootfs0");
    }

    // ── Display / serial ──────────────────────────────────────────────────
    if (!opts.graphic) {
        args.push_back("-nographic");
        args.push_back("-serial");
        args.push_back("mon:stdio");
    }
    else {
        args.push_back("-device");
        args.push_back("virtio-gpu-pci");
        args.push_back("-device");
        args.push_back("virtio-keyboard-pci");
        args.push_back("-device");
        args.push_back("virtio-mouse-pci");
    }

    // ── Kernel command-line (-append) — direct boot only ──────────────────
    // In U-Boot mode the bootargs come from U-Boot's extlinux.conf / environment.
    if (!use_uboot) {
        std::string console = arch_cfg->console;
        std::string append_str = has_disk_image ? "root=/dev/vda rw rootwait rootfstype=ext4 "
                                                  "earlycon "
                                                : "rw earlycon ";
        append_str += opts.graphic ? "console=tty0" : "console=" + console;
        if (!opts.append.empty()) {
            append_str += " ";
            append_str += opts.append;
        }
        args.push_back("-append");
        args.push_back(append_str);
    }

    // ── Network ───────────────────────────────────────────────────────────
    args.push_back("-netdev");
    args.push_back("user,id=net0,hostfwd=tcp::" + std::to_string(cfg.qemu.ssh_port) + "-:22");
    args.push_back("-device");
    args.push_back(default_net_device(arch, kernel_options) + ",netdev=net0");

    // ── 9P host shares (modules + apps) ───────────────────────────────────
    auto ninep = default_9p_device(arch, kernel_options);
    append_9p_share_if_exists(args, "moddev", cfg.paths.modules_dir, "modules_mount", ninep);
    append_9p_share_if_exists(args, "appdev", cfg.paths.apps_dir, "apps_mount", ninep);

    // ── Initrd (direct boot only — U-Boot loads its own ramdisk) ─────────
    if (!opts.initrd.empty() && !use_uboot) {
        args.push_back("-initrd");
        args.push_back(opts.initrd);
    }

    // ── GDB server ────────────────────────────────────────────────────────
    if (opts.gdb) {
        args.push_back("-gdb");
        args.push_back("tcp::" + std::to_string(cfg.qemu.gdb_port));
        args.push_back("-S");
    }

    // ── Extra pass-through arguments ──────────────────────────────────────
    for (const auto& a : opts.extra_args) {
        args.push_back(a);
    }

    return std::pair{*qemu_path, args};
}

auto QEMURunner::run(std::stop_token token, RunOptions opts) -> VoidResult {
    auto cmd = build_command(opts);
    if (!cmd)
        return make_error(cmd.error());
    auto& [binary, args] = *cmd;
    return ctx_->exec().run(token, binary, args);
}

}  // namespace elmos::domain::emulator
