// ============================================================================
// domain/rootfs/builder.cpp — Root filesystem creation
// ============================================================================

#include "builder.hpp"

#include <config/types.hpp>
#include <context/context.hpp>

#include <filesystem>

namespace elmos::domain::rootfs {

namespace fs = std::filesystem;

Builder::Builder(context::Context* ctx) : ctx_(ctx) {}

auto Builder::create(std::stop_token token, RootfsOptions opts) -> VoidResult {
    auto r = run_debootstrap(token, opts);
    if (!r)
        return r;

    if (!opts.packages.empty()) {
        r = install_packages(token, opts.packages);
        if (!r)
            return r;
    }

    if (!opts.post_build_script.empty()) {
        r = customize(token, opts.post_build_script);
        if (!r)
            return r;
    }

    return {};
}

auto Builder::run_debootstrap(std::stop_token token, const RootfsOptions& opts) -> VoidResult {
    auto& cfg = ctx_->config();
    auto rootfs_dir = cfg.paths.rootfs_dir;

    auto r = ctx_->fs().mkdir_all(rootfs_dir);
    if (!r)
        return r;

    return ctx_->exec().run(token, "sudo",
                            {
                                "debootstrap",
                                "--arch=" + cfg.build.arch,
                                opts.release,
                                rootfs_dir,
                                cfg.paths.debian_mirror,
                            });
}

auto Builder::install_packages(std::stop_token token, const std::vector<std::string>& packages)
    -> VoidResult {
    auto& cfg = ctx_->config();

    std::vector<std::string> args = {
        "chroot", cfg.paths.rootfs_dir, "apt-get", "install", "-y",
    };
    for (const auto& pkg : packages) {
        args.push_back(pkg);
    }

    return ctx_->exec().run(token, "sudo", args);
}

auto Builder::install_modules(std::stop_token token) -> VoidResult {
    auto& cfg = ctx_->config();
    auto env = ctx_->get_make_env();

    return ctx_->exec().run_with_env(token, env, "sudo",
                                     {
                                         "make",
                                         "-C",
                                         cfg.paths.kernel_dir,
                                         "ARCH=" + cfg.build.arch,
                                         "INSTALL_MOD_PATH=" + cfg.paths.rootfs_dir,
                                         "modules_install",
                                     });
}

auto Builder::customize(std::stop_token token, const std::string& script) -> VoidResult {
    auto& cfg = ctx_->config();
    return ctx_->exec().run(token, "sudo",
                            {
                                "chroot",
                                cfg.paths.rootfs_dir,
                                "/bin/bash",
                                "-c",
                                script,
                            });
}

auto Builder::clean() -> VoidResult {
    auto& cfg = ctx_->config();
    std::error_code ec;
    std::filesystem::remove_all(cfg.paths.rootfs_dir, ec);
    if (ec)
        return make_error(Error::generic("failed to clean rootfs: " + ec.message()));
    return {};
}

}  // namespace elmos::domain::rootfs
