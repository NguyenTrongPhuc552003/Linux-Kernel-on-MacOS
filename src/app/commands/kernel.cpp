// ============================================================================
// app/commands/kernel.cpp — Kernel management commands
// ============================================================================

#include <app/app.hpp>
#include <config/workspaces.hpp>
#include <domain/builder/kernel.hpp>

#include "commands.hpp"

#include <filesystem>
#include <format>
#include <fstream>
#include <stop_token>
#include <unordered_set>
#include <vector>

namespace elmos::app::commands {

namespace {

auto setup_sysroot_headers(App& app, std::stop_token token) -> VoidResult {
    namespace fs = std::filesystem;

    auto kernel_dir = fs::path(app.config().paths.kernel_dir);
    auto project_sysroot = fs::path(app.config().paths.project_root) / "include" / "sysroot";

    std::vector<fs::path> sysroots;
    std::unordered_set<std::string> seen;
    auto add_sysroot = [&](const fs::path& path) {
        auto normalized = path.lexically_normal().string();
        if (normalized.empty() || seen.contains(normalized)) {
            return;
        }
        seen.insert(normalized);
        sysroots.push_back(path);
    };

    add_sysroot(fs::path(app.config().paths.libraries_dir));
    auto active_ws = config::WorkspaceManager::get_active_workspace();
    if (active_ws) {
        add_sysroot(fs::path(config::WorkspaceManager::workspace_dir(*active_ws)) / "sysroot");
    }

    auto global_sysroot = fs::path(config::WorkspaceManager::global_elmos_dir()) / "sysroot";

    auto find_source_header = [&](const std::string& name) -> fs::path {
        std::vector<fs::path> candidates = {
            project_sysroot / name,
            global_sysroot / name,
        };
        for (const auto& candidate : candidates) {
            if (fs::exists(candidate)) {
                return candidate;
            }
        }
        return {};
    };

    auto asm_generic = kernel_dir / "include" / "uapi" / "asm-generic";
    for (const auto& sysroot_dir : sysroots) {
        auto sysroot_asm = sysroot_dir / "asm";
        std::error_code ec;
        fs::create_directories(sysroot_asm, ec);
        if (ec) {
            return make_error(Error::generic("failed to create directory " + sysroot_asm.string() +
                                             ": " + ec.message()));
        }

        // Keep byteswap/endian in sync for all sysroots used by kernel build.
        for (const auto* name : {"byteswap.h", "endian.h"}) {
            auto src = find_source_header(name);
            if (src.empty()) {
                continue;
            }

            auto dst = sysroot_dir / name;
            bool same_path = false;
            std::error_code eq_ec;
            same_path = fs::equivalent(src, dst, eq_ec);
            if (!same_path) {
                fs::copy_file(src, dst, fs::copy_options::overwrite_existing, ec);
                if (ec) {
                    return make_error(
                        Error::generic("failed to copy " + src.string() + " to " + dst.string()));
                }
            }
        }

        for (const auto* name : {"bitsperlong.h", "int-ll64.h", "posix_types.h", "types.h"}) {
            auto link_path = sysroot_asm / name;
            auto target_path = asm_generic / name;
            std::error_code rm_ec;
            fs::remove(link_path, rm_ec);

            std::error_code link_ec;
            fs::create_symlink(target_path, link_path, link_ec);
            if (link_ec) {
                return make_error(Error::generic("failed to create symlink " + link_path.string() +
                                                 " -> " + target_path.string()));
            }
        }

        // Refresh elf.h if missing or empty.
        auto elf_h = sysroot_dir / "elf.h";
        bool need_elf = !fs::exists(elf_h);
        if (!need_elf) {
            std::error_code sz_ec;
            auto size = fs::file_size(elf_h, sz_ec);
            need_elf = sz_ec || size == 0;
        }

        if (need_elf) {
            static constexpr auto kElfHeaderUrl =
                "https://raw.githubusercontent.com/bminor/glibc/master/elf/elf.h";
            auto r = app.exec().run(token, "curl", {"-fsSL", "-o", elf_h.string(), kElfHeaderUrl});
            if (!r) {
                return r;
            }
        }
    }

    return {};
}

}  // namespace

void register_kernel(App& app, CLI::App& cli) {
    auto* kernel = cli.add_subcommand("kernel", "Kernel configuration commands");

    // kernel config [type]
    auto* config_cmd = kernel->add_subcommand("config", "Configure the kernel");
    auto* config_type = config_cmd
                            ->add_option("type", "Config type (defconfig/menuconfig/tinyconfig)")
                            ->default_val("defconfig");
    config_cmd->callback([&app, config_type] {
        auto type = config_type->as<std::string>();
        app.printer().step("Running kernel {}...", type);
        std::stop_source ss;
        if (auto r = app.kernel_builder().configure(ss.get_token(), type); !r) {
            app.printer().error("Configuration failed: {}", r.error().message());
            return;
        }
        app.printer().success("Kernel configured!");
    });

    // kernel clone [url]
    auto* clone_cmd = kernel->add_subcommand("clone", "Clone the Linux kernel source");
    auto* clone_url =
        clone_cmd->add_option("url", "Git URL")
            ->default_val("https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git");
    clone_cmd->callback([&app, clone_url] {
        if (app.context().kernel_exists()) {
            app.printer().info("Kernel source already exists");
            return;
        }
        auto url = clone_url->as<std::string>();
        app.printer().step("Cloning kernel from {}...", url);
        std::stop_source ss;
        if (auto r = app.exec().run(ss.get_token(), "git",
                                    {"clone", url, app.config().paths.kernel_dir});
            !r) {
            app.printer().error("Clone failed: {}", r.error().message());
            return;
        }
        app.printer().success("Kernel cloned!");

        app.printer().step("Setting up sysroot headers...");

        if (auto r = setup_sysroot_headers(app, ss.get_token()); !r) {
            app.printer().error("Sysroot setup failed: {}", r.error().message());
            return;
        }

        app.printer().success("Sysroot ready");
    });

    // kernel build [targets...]
    auto* build_cmd = kernel->add_subcommand("build", "Build the Linux kernel");
    auto* jobs_opt = build_cmd->add_option("-j,--jobs", "Parallel build jobs")->default_val(0);
    build_cmd->callback([&app, jobs_opt] {
        int jobs = jobs_opt->as<int>();
        app.printer().step("Building kernel for {}...", app.config().build.arch);
        std::stop_source ss;

        // Keep sysroot headers/symlinks healthy for existing workspaces too.
        if (auto r = setup_sysroot_headers(app, ss.get_token()); !r) {
            app.printer().error("Sysroot setup failed: {}", r.error().message());
            return;
        }

        if (auto r = app.kernel_builder().build(ss.get_token(),
                                                domain::builder::BuildOptions{.jobs = jobs});
            !r) {
            app.printer().error("Build failed: {}", r.error().message());
            return;
        }
        app.printer().success("Build complete!");
    });

    // kernel clean
    auto* clean_cmd = kernel->add_subcommand("clean", "Clean kernel build artifacts");
    clean_cmd->callback([&app] {
        app.printer().step("Cleaning...");
        std::stop_source ss;
        if (auto r = app.kernel_builder().clean(ss.get_token()); !r) {
            app.printer().error("Clean failed: {}", r.error().message());
            return;
        }
        app.printer().success("Cleaned!");
    });

    // kernel status
    auto* status_cmd = kernel->add_subcommand("status", "Show kernel source status");
    status_cmd->callback([&app] {
        if (!app.context().kernel_exists()) {
            app.printer().info("Kernel source not found");
            app.printer().print("  Run 'elmos kernel clone' to download");
            return;
        }
        app.printer().success("Kernel source found at {}", app.config().paths.kernel_dir);
        if (app.context().has_config()) {
            app.printer().print("  ✓ Kernel configured");
        }
        else {
            app.printer().print("  ○ Not configured");
        }
        if (app.context().has_kernel_image()) {
            app.printer().print("  ✓ Kernel image built");
        }
        else {
            app.printer().print("  ○ Not built");
        }
    });

    // kernel pull
    auto pull_callback = [&app] {
        if (!app.context().kernel_exists()) {
            app.printer().info("Kernel source not found. Clone first.");
            return;
        }
        app.printer().step("Updating kernel source...");
        std::stop_source ss;
        if (auto r = app.exec().run(ss.get_token(), "git",
                                    {"-C", app.config().paths.kernel_dir, "pull"});
            !r) {
            app.printer().error("Pull failed: {}", r.error().message());
            return;
        }
        app.printer().success("Kernel updated!");
    };
    auto* pull_cmd = kernel->add_subcommand("pull", "Update kernel source (pull latest)");
    pull_cmd->callback(pull_callback);

    // kernel update — alias for pull (per WORKFLOW.md)
    auto* update_cmd = kernel->add_subcommand("update", "Update kernel source (alias for pull)");
    update_cmd->callback(pull_callback);

    // kernel switch [ref]
    auto* switch_cmd = kernel->add_subcommand("switch", "Switch branch/tag");
    auto* ref_opt = switch_cmd->add_option("ref", "Branch or tag to switch to");
    switch_cmd->callback([&app, ref_opt] {
        if (!app.context().kernel_exists()) {
            app.printer().info("Kernel source not found. Clone first.");
            return;
        }
        if (ref_opt->count() == 0) {
            app.printer().info("Usage: elmos kernel switch <ref>");
            return;
        }
        auto ref = ref_opt->as<std::string>();
        app.printer().step("Switching to {}...", ref);
        std::stop_source ss;
        if (auto r = app.exec().run(ss.get_token(), "git",
                                    {"-C", app.config().paths.kernel_dir, "checkout", ref});
            !r) {
            app.printer().error("Switch failed: {}", r.error().message());
            return;
        }
        app.printer().success("Switched to {}", ref);
    });

    // kernel reset
    auto* reset_cmd = kernel->add_subcommand("reset", "Reset kernel source (reclone)");
    reset_cmd->callback([&app] {
        std::stop_source ss;
        auto& dir = app.config().paths.kernel_dir;
        if (app.context().kernel_exists()) {
            app.printer().step("Removing existing kernel source...");
            std::filesystem::remove_all(dir);
        }
        std::string url = "https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git";
        app.printer().step("Cloning kernel from {}...", url);
        if (auto r = app.exec().run(ss.get_token(), "git", {"clone", url, dir}); !r) {
            app.printer().error("Clone failed: {}", r.error().message());
            return;
        }
        app.printer().success("Kernel reset complete!");

        app.printer().step("Setting up sysroot headers...");

        if (auto r = setup_sysroot_headers(app, ss.get_token()); !r) {
            app.printer().error("Sysroot setup failed: {}", r.error().message());
            return;
        }

        app.printer().success("Sysroot ready");
    });

    // kernel show — show build result
    auto* show_cmd = kernel->add_subcommand("show", "Show kernel build result");
    show_cmd->callback([&app] {
        auto& cfg = app.config();
        app.printer().step("Kernel build information:");
        app.printer().print("  Architecture: {}", cfg.build.arch);
        app.printer().print("  Kernel dir:   {}", cfg.paths.kernel_dir);

        if (!app.context().kernel_exists()) {
            app.printer().info("  Kernel source not cloned");
            return;
        }

        if (app.context().has_config()) {
            app.printer().print("  ✓ Kernel configured");
        }
        else {
            app.printer().print("  ○ Not configured");
        }

        if (app.context().has_kernel_image()) {
            auto image = app.context().get_kernel_image();
            app.printer().print("  ✓ Kernel image: {}", image);
        }
        else {
            app.printer().print("  ○ Kernel not built");
        }

        // Check for dtbs
        auto dtbs_dir = std::filesystem::path(cfg.paths.kernel_dir) / "arch" / cfg.build.arch /
                        "boot" / "dts";
        if (std::filesystem::exists(dtbs_dir)) {
            app.printer().print("  ✓ DTBs dir: {}", dtbs_dir.string());
        }

        // Check for modules
        auto modules_dir = std::filesystem::path(cfg.paths.kernel_dir) / "modules";
        if (std::filesystem::exists(modules_dir)) {
            app.printer().print("  ✓ Modules built");
        }
    });

    // Show help when 'elmos kernel' is run with no subcommand
    kernel->callback([kernel] {
        if (kernel->get_subcommands().empty()) {
            std::cout << kernel->help();
        }
    });

    // kernel install
    auto* install_cmd = kernel->add_subcommand("install", "Install kernel artifacts");
    install_cmd->callback([&app] {
        if (!app.context().has_kernel_image()) {
            app.printer().warn("Kernel not built yet. Run 'elmos kernel build' first.");
            return;
        }
        auto install_dir = std::filesystem::path(app.config().image.mount_point) / "kernel";
        std::filesystem::create_directories(install_dir);
        app.printer().step("Installing kernel artifacts to {}", install_dir.string());
        // Symlink kernel image
        auto image = app.context().get_kernel_image();
        auto target = install_dir / std::filesystem::path(image).filename();
        std::filesystem::remove(target);
        std::filesystem::create_symlink(image, target);
        app.printer().success("Kernel artifacts installed!");
    });
}

}  // namespace elmos::app::commands
