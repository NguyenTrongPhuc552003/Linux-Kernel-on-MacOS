// ============================================================================
// app/commands/kernel.cpp — Kernel management commands
// ============================================================================

#include <app/app.hpp>
#include <domain/builder/kernel.hpp>

#include "commands.hpp"

#include <format>
#include <stop_token>

namespace elmos::app::commands {

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
    });

    // kernel build [targets...]
    auto* build_cmd = kernel->add_subcommand("build", "Build the Linux kernel");
    int jobs = 0;
    build_cmd->add_option("-j,--jobs", jobs, "Parallel build jobs");
    build_cmd->callback([&app, &jobs] {
        app.printer().step("Building kernel for {}...", app.config().build.arch);
        std::stop_source ss;
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
    auto* pull_cmd = kernel->add_subcommand("pull", "Update kernel source");
    pull_cmd->callback([&app] {
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
    });

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
