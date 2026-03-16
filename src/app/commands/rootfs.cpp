// ============================================================================
// app/commands/rootfs.cpp — Root filesystem management
// ============================================================================

#include <app/app.hpp>
#include <domain/rootfs/builder.hpp>

#include "commands.hpp"

#include <filesystem>
#include <fstream>
#include <stop_token>

namespace elmos::app::commands {

namespace {

void show_rootfs_status(App& app) {
    namespace fs = std::filesystem;

    auto rootfs_dir = fs::path(app.config().paths.rootfs_dir);
    app.printer().step("Rootfs status:");
    app.printer().print("  Architecture: {}", app.config().build.arch);
    app.printer().print("  Rootfs dir:   {}", rootfs_dir.string());
    app.printer().print("  Disk image:   {}", app.config().paths.disk_image);

    if (!fs::exists(rootfs_dir)) {
        app.printer().info("  No rootfs found. Run 'elmos rootfs clone' or 'elmos rootfs build'");
        if (!fs::exists(app.config().paths.disk_image)) {
            app.printer().print("  ○ No disk image assembled yet");
        }
        return;
    }

    app.printer().print("  ✓ Rootfs directory exists");

    auto debootstrap_marker = rootfs_dir / "debootstrap" / "debootstrap";
    auto os_release = rootfs_dir / "etc" / "os-release";

    if (fs::exists(os_release)) {
        app.printer().print("  ✓ Bootstrap completed");
    }
    else if (fs::exists(debootstrap_marker)) {
        app.printer().print("  ○ Foreign bootstrap staged");
    }
    else {
        app.printer().print("  ○ Rootfs not built");
    }

    auto tools_dir = fs::path(app.config().paths.project_root) / "tools" / "debootstrap" /
                     ".local" / "usr" / "sbin" / "debootstrap";
    if (fs::exists(tools_dir)) {
        app.printer().print("  ✓ Debootstrap installed: {}", tools_dir.string());
    }

    if (fs::exists(app.config().paths.disk_image)) {
        app.printer().print("  ✓ Disk image assembled");
        // Heuristic: a partitioned disk image for U-Boot is larger than a
        // simple ext4 image of the same rootfs, and its first 512 bytes
        // contain MBR signature 0x55AA at offsets 510-511.
        std::ifstream img(app.config().paths.disk_image, std::ios::binary);
        if (img) {
            img.seekg(510);
            unsigned char sig[2] = {0, 0};
            img.read(reinterpret_cast<char*>(sig), 2);
            if (sig[0] == 0x55 && sig[1] == 0xAA) {
                app.printer().print(
                    "  \u2139 Disk image has MBR partition table (U-Boot boot mode)");
            }
            else {
                app.printer().print(
                    "  \u2139 Disk image is a flat ext4 image (direct kernel boot mode)");
            }
        }
    }
    else {
        app.printer().print("  ○ Disk image missing (auto-created by 'elmos qemu run')");
    }
}

void show_rootfs_result(App& app) {
    namespace fs = std::filesystem;

    auto rootfs_dir = fs::path(app.config().paths.rootfs_dir);
    app.printer().step("Rootfs build result:");
    app.printer().print("  Architecture: {}", app.config().build.arch);
    app.printer().print("  Rootfs dir:   {}", rootfs_dir.string());
    app.printer().print("  Disk image:   {}", app.config().paths.disk_image);

    if (!fs::exists(rootfs_dir)) {
        app.printer().info("  No rootfs found. Run 'elmos rootfs clone' then 'elmos rootfs build'");
        return;
    }

    auto os_release = rootfs_dir / "etc" / "os-release";
    auto debootstrap_marker = rootfs_dir / "debootstrap" / "debootstrap";

    if (fs::exists(os_release)) {
        app.printer().print("  ✓ Build completed");
        app.printer().print("  ✓ Metadata file: {}", os_release.string());

        auto init_bin = rootfs_dir / "sbin" / "init";
        auto shell_bin = rootfs_dir / "bin" / "sh";
        app.printer().print("  {} init binary", fs::exists(init_bin) ? "✓" : "○");
        app.printer().print("  {} shell binary", fs::exists(shell_bin) ? "✓" : "○");
    }
    else if (fs::exists(debootstrap_marker)) {
        app.printer().print("  ○ Build partially completed (foreign bootstrap staged)");
        app.printer().print("  ○ Missing {}", os_release.string());
    }
    else {
        app.printer().print("  ○ Rootfs directory exists but no build markers found");
    }

    app.printer().print("  {} Disk image", fs::exists(app.config().paths.disk_image) ? "✓" : "○");
    if (fs::exists(app.config().paths.disk_image)) {
        std::ifstream img(app.config().paths.disk_image, std::ios::binary);
        if (img) {
            img.seekg(510);
            unsigned char sig[2] = {0, 0};
            img.read(reinterpret_cast<char*>(sig), 2);
            if (sig[0] == 0x55 && sig[1] == 0xAA) {
                app.printer().print("  ℹ Partitioned disk (U-Boot boot mode)");
            }
            else {
                app.printer().print("  ℹ Flat ext4 image (direct kernel boot mode)");
            }
        }
    }
}

}  // namespace

void register_rootfs(App& app, CLI::App& cli) {
    auto* rootfs = cli.add_subcommand("rootfs", "Manage root filesystem");

    // rootfs clone — set up rootfs directory and ensure debootstrap is available
    auto* clone_cmd = rootfs->add_subcommand("clone", "Set up rootfs directory in workspace");
    clone_cmd->callback([&app] {
        auto rootfs_dir = app.config().paths.rootfs_dir;

        // Ensure rootfs directory exists
        if (!std::filesystem::exists(rootfs_dir)) {
            std::filesystem::create_directories(rootfs_dir);
            app.printer().step("Created rootfs directory at {}", rootfs_dir);
        }
        else {
            app.printer().info("Rootfs directory already exists at {}", rootfs_dir);
        }

        // Ensure debootstrap is available at tools/debootstrap.
        // Cannot use 'git submodule' since the workspace mount point is typically
        // not a git repository (e.g. /Volumes/elmos). Clone directly instead.
        auto project_root = app.config().paths.project_root;
        auto debootstrap_dir = std::filesystem::path(project_root) / "tools" / "debootstrap";
        auto debootstrap_bin = debootstrap_dir / "debootstrap";
        auto debootstrap_install = debootstrap_dir / ".local";
        auto debootstrap_install_bin = debootstrap_install / "usr" / "sbin" / "debootstrap";

        if (!std::filesystem::exists(debootstrap_bin)) {
            app.printer().step("Cloning debootstrap...");
            std::stop_source ss;

            // Remove any partial directory before cloning
            if (std::filesystem::exists(debootstrap_dir)) {
                std::error_code ec;
                std::filesystem::remove_all(debootstrap_dir, ec);
            }

            // Clone debootstrap directly from upstream Debian repository
            static constexpr auto kDebootstrapRepo =
                "https://salsa.debian.org/installer-team/debootstrap.git";
            if (auto r = app.exec().run(
                    ss.get_token(), "git",
                    {"clone", "--depth=1", kDebootstrapRepo, debootstrap_dir.string()});
                !r) {
                app.printer().error("Failed to clone debootstrap: {}", r.error().message());
                app.printer().info("You can manually clone: git clone {} {}", kDebootstrapRepo,
                                   debootstrap_dir.string());
                return;
            }

            // Build debootstrap from source (generate devices tarball etc.)
            app.printer().step("Building debootstrap from source...");
            if (auto r = app.exec().run_in_dir(ss.get_token(), debootstrap_dir.string(), "make",
                                               {"devices.tar.gz"});
                !r) {
                // Non-fatal: the debootstrap script works without it
                app.printer().info("Skipped devices.tar.gz generation (non-fatal)");
            }

            // Ensure the debootstrap script is executable
            std::error_code perm_ec;
            std::filesystem::permissions(debootstrap_bin,
                                         std::filesystem::perms::owner_exec |
                                             std::filesystem::perms::group_exec |
                                             std::filesystem::perms::others_exec,
                                         std::filesystem::perm_options::add, perm_ec);
        }

        // Build + install debootstrap from source into a local prefix.
        if (std::filesystem::exists(debootstrap_bin) &&
            !std::filesystem::exists(debootstrap_install_bin)) {
            app.printer().step("Installing debootstrap from source...");
            std::stop_source ss;

            if (auto r = app.exec().run_in_dir(ss.get_token(), debootstrap_dir.string(), "make",
                                               {"clean"});
                !r) {
                app.printer().warn("debootstrap clean step failed: {}", r.error().message());
            }

            if (auto r =
                    app.exec().run_in_dir(ss.get_token(), debootstrap_dir.string(), "make",
                                          {"install", "DESTDIR=" + debootstrap_install.string()});
                !r) {
                app.printer().error("Failed to install debootstrap: {}", r.error().message());
                return;
            }
        }

        if (std::filesystem::exists(debootstrap_install_bin)) {
            app.printer().success("Rootfs ready (debootstrap installed at {})",
                                  debootstrap_install_bin.string());
        }
        else if (std::filesystem::exists(debootstrap_bin)) {
            app.printer().success("Rootfs ready (debootstrap at {})", debootstrap_bin.string());
        }
        else {
            app.printer().warn("debootstrap not found at {}", debootstrap_bin.string());
            app.printer().info("Manual setup: git clone "
                               "https://salsa.debian.org/installer-team/debootstrap.git {}",
                               debootstrap_dir.string());
        }
    });

    // rootfs build
    auto* build_cmd = rootfs->add_subcommand("build", "Build root filesystem");
    auto* size_opt = build_cmd->add_option("-s,--size", "Rootfs size")->default_val("5G");
    auto* uboot_flag = build_cmd->add_flag(
        "-u,--uboot", "Create a partitioned disk image (FAT32 boot + ext4 rootfs) "
                      "required for U-Boot bootloader");
    build_cmd->callback([&app, size_opt, uboot_flag] {
        bool with_boot_partition = uboot_flag->count() > 0;
        auto size = size_opt->as<std::string>();
        if (with_boot_partition) {
            app.printer().step("Building rootfs ({}, U-Boot partitioned disk)...", size);
        }
        else {
            app.printer().step("Building rootfs ({})...", size);
        }
        std::stop_source ss;
        if (auto r = app.rootfs_builder().create(
                ss.get_token(),
                domain::rootfs::RootfsOptions{.assemble_disk_image = true,
                                              .disk_size = size,
                                              .with_boot_partition = with_boot_partition});
            !r) {
            app.printer().error("Rootfs build failed: {}", r.error().message());
            return;
        }
        if (with_boot_partition) {
            app.printer().success("Rootfs built with FAT32 boot + ext4 rootfs partitions at {}",
                                  app.config().paths.disk_image);
        }
        else {
            app.printer().success("Rootfs built and disk image assembled at {}",
                                  app.config().paths.disk_image);
        }
    });

    auto* status_cmd = rootfs->add_subcommand("status", "Show rootfs status");
    status_cmd->callback([&app] { show_rootfs_status(app); });

    auto* show_cmd = rootfs->add_subcommand("show", "Show rootfs build result");
    show_cmd->callback([&app] { show_rootfs_result(app); });

    // rootfs clean
    auto* clean_cmd = rootfs->add_subcommand("clean", "Remove root filesystem");
    clean_cmd->callback([&app] {
        if (auto r = app.rootfs_builder().clean(); !r) {
            app.printer().error("Clean failed: {}", r.error().message());
            return;
        }
        app.printer().success("Rootfs directory and disk image cleaned!");
    });

    // Show help when 'elmos rootfs' is run with no subcommand
    rootfs->callback([rootfs] {
        if (rootfs->get_subcommands().empty()) {
            std::cout << rootfs->help();
        }
    });
}

}  // namespace elmos::app::commands
