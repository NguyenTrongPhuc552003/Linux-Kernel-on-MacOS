// ============================================================================
// app/commands/qemu.cpp — QEMU emulator commands
// ============================================================================

#include <app/app.hpp>
#include <domain/emulator/qemu.hpp>

#include "commands.hpp"

#include <cstdlib>
#include <filesystem>
#include <stop_token>

namespace elmos::app::commands {

namespace {

void show_qemu_components(App& app) {
    namespace fs = std::filesystem;

    auto& cfg = app.config();
    auto arch_cfg = cfg.get_arch_config();

    app.printer().step("QEMU setup:");
    app.printer().print("  Architecture: {}", cfg.build.arch);

    if (arch_cfg) {
        app.printer().print("  QEMU binary:  {}", arch_cfg->qemu_binary);
        app.printer().print("  Machine:      {}", arch_cfg->qemu_machine);
        app.printer().print("  CPU:          {}", arch_cfg->qemu_cpu);
        app.printer().print("  Memory/SMP:   {} / {}", cfg.qemu.memory, cfg.qemu.smp);
        app.printer().print("  GDB/SSH port: {} / {}", cfg.qemu.gdb_port, cfg.qemu.ssh_port);
        app.printer().print("  {} QEMU executable in PATH",
                            app.qemu_runner().is_available({}) ? "✓" : "○");
    }
    else {
        app.printer().print("  ○ Unknown architecture configuration");
    }

    auto kernel_image = app.context().get_kernel_image();
    auto vmlinux = app.context().get_vmlinux();
    auto disk_image = cfg.paths.disk_image;
    auto rootfs_dir = cfg.paths.rootfs_dir;
    auto bootloader_dir = fs::path(cfg.image.mount_point) / "bootloader";

    app.printer().print("  {} Kernel image: {}", fs::exists(kernel_image) ? "✓" : "○",
                        kernel_image);
    app.printer().print("  {} vmlinux debug symbols: {}", fs::exists(vmlinux) ? "✓" : "○", vmlinux);
    app.printer().print("  {} Rootfs directory: {}", fs::is_directory(rootfs_dir) ? "✓" : "○",
                        rootfs_dir);
    app.printer().print("  {} Rootfs disk image: {}", fs::exists(disk_image) ? "✓" : "○",
                        disk_image);
    if (fs::is_directory(rootfs_dir) && !fs::exists(disk_image)) {
        app.printer().print("  ○ Disk image will be assembled automatically on 'elmos qemu run'");
    }

    bool has_bootloader = fs::exists(bootloader_dir / "u-boot.bin") ||
                          fs::exists(bootloader_dir / "u-boot.itb") ||
                          fs::exists(bootloader_dir / "u-boot");
    app.printer().print("  {} Bootloader artifact: {}", has_bootloader ? "✓" : "○",
                        bootloader_dir.string());
    if (has_bootloader) {
        const auto& arch = cfg.build.arch;
        std::string boot_mode;
        if (arch == "arm64") {
            boot_mode = "U-Boot via -bios (arm64 virt firmware slot)";
        }
        else if (arch == "arm") {
            boot_mode = "U-Boot via -kernel (arm32 virt kernel slot)";
        }
        else if (arch == "riscv") {
            boot_mode = "OpenSBI (-bios default) + U-Boot via -kernel (RISC-V S-mode)";
        }
        else {
            boot_mode = "U-Boot detected (arch unsupported — will fall back to direct boot)";
        }
        app.printer().print("  \u2139 Boot mode: {}", boot_mode);
        app.printer().print(
            "  \u2139 Disk image must have a FAT32 boot partition with extlinux.conf");
    }
    else {
        app.printer().print("  \u2139 Boot mode: direct kernel boot (-kernel Linux image)");
    }
    app.printer().print("  {} Modules dir: {}", fs::is_directory(cfg.paths.modules_dir) ? "✓" : "○",
                        cfg.paths.modules_dir);
    app.printer().print("  {} Apps dir: {}", fs::is_directory(cfg.paths.apps_dir) ? "✓" : "○",
                        cfg.paths.apps_dir);
}

}  // namespace

void register_qemu(App& app, CLI::App& cli) {
    auto* qemu = cli.add_subcommand("qemu", "Run kernel in emulator");

    // qemu show — show QEMU component readiness
    auto* show_cmd = qemu->add_subcommand("show", "Show QEMU components and readiness");
    show_cmd->callback([&app] { show_qemu_components(app); });

    // qemu status — check running QEMU processes
    auto* status_cmd = qemu->add_subcommand("status", "Check running QEMU processes");
    status_cmd->callback([&app] {
        std::stop_source ss;
        auto r = app.exec().output(ss.get_token(), "pgrep", {"-la", "qemu"});
        if (!r || r->empty()) {
            app.printer().info("No QEMU processes running");
            return;
        }
        app.printer().step("Running QEMU processes:");
        app.printer().print("{}", *r);
    });

    // qemu clean <pid>||all — kill QEMU processes
    auto* clean_cmd = qemu->add_subcommand("clean", "Kill QEMU process(es)");
    auto* clean_target = clean_cmd->add_option("target", "Process ID or 'all'")->required();
    clean_cmd->callback([&app, clean_target] {
        auto target = clean_target->as<std::string>();
        std::stop_source ss;
        if (target == "all") {
            app.printer().step("Killing all QEMU processes...");
            auto r = app.exec().run(ss.get_token(), "pkill", {"-f", "qemu"});
            if (!r) {
                app.printer().info("No QEMU processes to kill");
                return;
            }
            app.printer().success("All QEMU processes terminated");
        }
        else {
            app.printer().step("Killing QEMU process {}...", target);
            auto r = app.exec().run(ss.get_token(), "kill", {target});
            if (!r) {
                app.printer().error("Failed to kill process {}: {}", target, r.error().message());
                return;
            }
            app.printer().success("Process {} terminated", target);
        }
    });

    // qemu run [-d|--debug] [-g|--graphical] [--force-direct-kernel]
    auto* run_cmd = qemu->add_subcommand("run", "Boot kernel in QEMU");
    auto* debug_flag = run_cmd->add_flag("-d,--debug", "Enable GDB server for debugging");
    auto* graphical_flag = run_cmd->add_flag("-g,--graphical", "Enable graphical output");
    auto* force_direct_flag = run_cmd->add_flag(
        "--force-direct-kernel", "Boot kernel directly (-kernel Image) even if U-Boot is present");
    auto* append_opt =
        run_cmd->add_option("--append", "Extra kernel cmdline parameters")->default_val("");
    auto* initrd_opt = run_cmd->add_option("--initrd", "Custom initrd image path")->default_val("");
    auto* extra_args_opt = run_cmd->add_option("--extra-arg", "Extra QEMU argument")
                               ->expected(0, -1)
                               ->default_val(std::vector<std::string>{});
    run_cmd->callback([&app, debug_flag, graphical_flag, force_direct_flag, append_opt, initrd_opt,
                       extra_args_opt] {
        if (!app.context().has_kernel_image()) {
            app.printer().warn("Kernel not built. Run 'elmos kernel build' first.");
            return;
        }
        bool gdb = debug_flag->count() > 0;
        bool graphic = graphical_flag->count() > 0;
        bool force_direct = force_direct_flag->count() > 0;
        auto append = append_opt->as<std::string>();
        auto initrd = initrd_opt->as<std::string>();
        auto extra_args = extra_args_opt->as<std::vector<std::string>>();

        // When U-Boot is present, the disk image must have a FAT32 boot
        // partition so that U-Boot can find the kernel via extlinux.conf.
        bool has_uboot = !force_direct && app.qemu_runner().has_bootloader();

        std::stop_source ss;
        if (initrd.empty()) {
            auto disk_ready = app.rootfs_builder().ensure_disk_image(ss.get_token(), "", false,
                                                                     has_uboot);
            if (!disk_ready) {
                app.printer().error("Rootfs disk image preparation failed: {}",
                                    disk_ready.error().message());
                return;
            }
            if (*disk_ready) {
                app.printer().success("Rootfs disk image ready at {}",
                                      app.config().paths.disk_image);
            }
        }

        if (gdb)
            app.printer().step("Launching QEMU with GDB server on :{}...",
                               app.config().qemu.gdb_port);
        else if (has_uboot)
            app.printer().step("Launching QEMU with U-Boot bootloader...");
        else
            app.printer().step("Launching QEMU...");

        const domain::emulator::RunOptions opts{.gdb = gdb,
                                                .graphic = graphic,
                                                .initrd = initrd,
                                                .append = append,
                                                .extra_args = extra_args,
                                                .force_direct_kernel = force_direct};

        auto cmd = app.qemu_runner().build_command(opts);
        if (!cmd) {
            app.printer().error("QEMU failed: {}", cmd.error().message());
            return;
        }

        const auto& [binary, args] = *cmd;
        app.printer().print("QEMU argv:");
        app.printer().print("  {}", binary);
        for (const auto& arg : args) {
            app.printer().print("  {}", arg);
        }

        if (auto r = app.exec().run(ss.get_token(), binary, args); !r) {
            app.printer().error("QEMU failed: {}", r.error().message());
        }
    });

    // Show help when 'elmos qemu' is run with no subcommand
    qemu->callback([qemu] {
        if (qemu->get_subcommands().empty()) {
            std::cout << qemu->help();
        }
    });
}

}  // namespace elmos::app::commands
