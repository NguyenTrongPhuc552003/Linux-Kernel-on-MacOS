// ============================================================================
// domain/builder/kernel.cpp — Kernel build orchestration
// ============================================================================

#include "kernel.hpp"

#include <config/arch.hpp>
#include <config/defaults.hpp>
#include <config/types.hpp>
#include <context/context.hpp>
#include <domain/toolchain/manager.hpp>

#include <filesystem>
#include <string>

namespace elmos::domain::builder {

namespace fs = std::filesystem;

KernelBuilder::KernelBuilder(context::Context* ctx, toolchain::Manager* tm) : ctx_(ctx), tm_(tm) {}

auto KernelBuilder::get_toolchain_env() -> Result<std::pair<EnvList, std::string>> {
    auto env = ctx_->get_make_env();
    auto& cfg = ctx_->config();
    std::string cross = cfg.build.cross_compile;

    auto arch_cfg = config::get_arch_config(cfg.build.arch);
    if (!arch_cfg || arch_cfg->gcc_binary.empty()) {
        return std::pair{env, cross};
    }

    // Extract target tuple from gcc binary
    auto binary = arch_cfg->gcc_binary;
    if (!binary.ends_with("-gcc")) {
        return std::pair{env, cross};
    }
    auto target = binary.substr(0, binary.size() - 4);
    auto bin_dir = tm_->get_bin_dir(target);

    if (!ctx_->fs().is_dir(bin_dir)) {
        return std::pair{env, cross};
    }

    // Prepend toolchain bin dir to PATH and update CROSS_COMPILE
    cross = target + "-";
    EnvList new_env;
    bool path_updated = false;

    for (auto& e : env) {
        if (e.starts_with("PATH=")) {
            new_env.push_back("PATH=" + bin_dir + ":" + e.substr(5));
            path_updated = true;
        }
        else if (e.starts_with("CROSS_COMPILE=")) {
            new_env.push_back("CROSS_COMPILE=" + cross);
        }
        else {
            new_env.push_back(e);
        }
    }
    if (!path_updated) {
        new_env.push_back("PATH=" + bin_dir);
    }

    return std::pair{new_env, cross};
}

auto KernelBuilder::build(std::stop_token token, BuildOptions opts) -> VoidResult {
    auto& cfg = ctx_->config();

    int jobs = opts.jobs > 0 ? opts.jobs : cfg.build.jobs;

    auto tc = get_toolchain_env();
    if (!tc)
        return make_error(tc.error());
    auto& [env, cross] = *tc;

    std::vector<std::string> args = {
        "-C",     cfg.paths.kernel_dir,     "-j" + std::to_string(jobs), "ARCH=" + cfg.build.arch,
        "LLVM=1", "CROSS_COMPILE=" + cross,
    };
    for (const auto& t : opts.targets) {
        args.push_back(t);
    }

    return ctx_->exec().run_with_env(token, env, "make", args);
}

auto KernelBuilder::configure(std::stop_token token, const std::string& config_type) -> VoidResult {
    // Validate config type
    bool valid = false;
    for (auto ct : config::kernel_config_types()) {
        if (ct == config_type) {
            valid = true;
            break;
        }
    }
    if (!valid) {
        return make_error(Error::config("invalid config type: " + config_type));
    }

    auto& cfg = ctx_->config();
    auto tc = get_toolchain_env();
    if (!tc)
        return make_error(tc.error());
    auto& [env, cross] = *tc;

    std::vector<std::string> args = {
        "-C",     cfg.paths.kernel_dir,     "ARCH=" + cfg.build.arch,
        "LLVM=1", "CROSS_COMPILE=" + cross, config_type,
    };

    auto r = ctx_->exec().run_with_env(token, env, "make", args);
    if (!r)
        return r;

    if (config_type == "kvm_guest.config") {
        return force_graphics_config(token);
    }
    return {};
}

auto KernelBuilder::force_graphics_config(std::stop_token token) -> VoidResult {
    auto& cfg = ctx_->config();
    auto script = (fs::path(cfg.paths.kernel_dir) / "scripts" / "config").string();
    auto config_file = (fs::path(cfg.paths.kernel_dir) / ".config").string();

    std::vector<std::string> args = {
        "--file",   config_file,
        "--enable", "CONFIG_DRM",
        "--enable", "CONFIG_DRM_VIRTIO_GPU",
        "--enable", "CONFIG_FB",
        "--enable", "CONFIG_FRAMEBUFFER_CONSOLE",
    };

    auto r = ctx_->exec().run(token, script, args);
    if (!r)
        return r;

    // Finalize with olddefconfig
    auto env = ctx_->get_make_env();
    return ctx_->exec().run_with_env(token, env, "make",
                                     {
                                         "-C",
                                         cfg.paths.kernel_dir,
                                         "ARCH=" + cfg.build.arch,
                                         "LLVM=1",
                                         "olddefconfig",
                                     });
}

auto KernelBuilder::clean(std::stop_token token) -> VoidResult {
    auto& cfg = ctx_->config();
    auto env = ctx_->get_make_env();
    return ctx_->exec().run_silent(token, env, "make",
                                   {
                                       "-C",
                                       cfg.paths.kernel_dir,
                                       "ARCH=" + cfg.build.arch,
                                       "LLVM=1",
                                       "distclean",
                                   });
}

auto KernelBuilder::enable_kvm_config(std::stop_token token) -> VoidResult {
    auto& cfg = ctx_->config();
    auto env = ctx_->get_make_env();
    return ctx_->exec().run_with_env(token, env, "make",
                                     {
                                         "-C",
                                         cfg.paths.kernel_dir,
                                         "ARCH=" + cfg.build.arch,
                                         "LLVM=1",
                                         "olddefconfig",
                                     });
}

auto KernelBuilder::get_default_targets() -> std::vector<std::string> {
    return ctx_->get_default_targets();
}

auto KernelBuilder::has_config() -> bool {
    return ctx_->has_config();
}
auto KernelBuilder::has_kernel_image() -> bool {
    return ctx_->has_kernel_image();
}

}  // namespace elmos::domain::builder
