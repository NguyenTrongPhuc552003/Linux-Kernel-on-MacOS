// ============================================================================
// app/commands/rootfs.cpp — Root filesystem management
// ============================================================================

#include <app/app.hpp>
#include <domain/rootfs/builder.hpp>

#include "commands.hpp"

#include <stop_token>

namespace elmos::app::commands {

void register_rootfs(App& app, CLI::App& cli) {
    auto* rootfs = cli.add_subcommand("rootfs", "Manage root filesystem");

    // rootfs build
    auto* build_cmd = rootfs->add_subcommand("build", "Build root filesystem");
    std::string size = "5G";
    build_cmd->add_option("-s,--size", size, "Rootfs size");
    build_cmd->callback([&app, &size] {
        app.printer().step("Building rootfs ({})...", size);
        std::stop_source ss;
        if (auto r = app.rootfs_builder().create(ss.get_token(), domain::rootfs::RootfsOptions{});
            !r) {
            app.printer().error("Rootfs build failed: {}", r.error().message());
            return;
        }
        app.printer().success("Rootfs built!");
    });

    // rootfs clean
    auto* clean_cmd = rootfs->add_subcommand("clean", "Remove root filesystem");
    clean_cmd->callback([&app] {
        if (auto r = app.rootfs_builder().clean(); !r) {
            app.printer().error("Clean failed: {}", r.error().message());
            return;
        }
        app.printer().success("Rootfs cleaned!");
    });

    // rootfs status
    auto* status_cmd = rootfs->add_subcommand("status", "Show rootfs status");
    status_cmd->callback([&app] {
        auto rootfs_dir = app.config().paths.rootfs_dir;
        if (std::filesystem::exists(rootfs_dir)) {
            app.printer().success("Rootfs exists at {}", rootfs_dir);
        }
        else {
            app.printer().info("No rootfs found. Run 'elmos rootfs build'");
        }
    });
}

}  // namespace elmos::app::commands
