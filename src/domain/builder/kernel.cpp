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

namespace {
void append_host_make_flags(context::Context* ctx, std::vector<std::string>& args) {
    auto host_flags = ctx->get_host_cflags();
    if (host_flags.empty()) {
        return;
    }

    // Keep C and preprocessor host flags aligned for Linux host tools.
    args.push_back("HOSTCFLAGS=" + host_flags);
    args.push_back("HOSTCPPFLAGS=" + host_flags);
}

auto qemu_boot_symbols() -> const std::vector<std::string>& {
    static const std::vector<std::string> symbols = {
        "CONFIG_DEVTMPFS",        "CONFIG_DEVTMPFS_MOUNT",  "CONFIG_EXT4_FS",
        "CONFIG_TMPFS",           "CONFIG_TMPFS_POSIX_ACL", "CONFIG_PCI",
        "CONFIG_VIRTIO_PCI",      "CONFIG_VIRTIO_MMIO",     "CONFIG_VIRTIO_BLK",
        "CONFIG_NET_9P",          "CONFIG_NET_9P_VIRTIO",   "CONFIG_9P_FS",
        "CONFIG_9P_FS_POSIX_ACL",
    };
    return symbols;
}
}  // namespace

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

    auto apply_qemu_boot = force_qemu_boot_config(token);
    if (!apply_qemu_boot) {
        return make_error(apply_qemu_boot.error());
    }

    std::vector<std::string> args = {
        "-C",     cfg.paths.kernel_dir,     "-j" + std::to_string(jobs), "ARCH=" + cfg.build.arch,
        "LLVM=1", "CROSS_COMPILE=" + cross,
    };

    // Pass HOSTCFLAGS as a make command-line argument to override the kernel
    // Makefile's own HOSTCFLAGS assignment. Environment variables alone are
    // overridden by Makefile simple-assignments; command-line variables take
    // highest precedence. This ensures the sysroot include path (with elf.h,
    // byteswap.h polyfills) reaches host tool compilations on macOS.
    append_host_make_flags(ctx_, args);

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

    append_host_make_flags(ctx_, args);

    auto r = ctx_->exec().run_with_env(token, env, "make", args);
    if (!r)
        return r;

    auto apply_qemu_boot = force_qemu_boot_config(token);
    if (!apply_qemu_boot) {
        return make_error(apply_qemu_boot.error());
    }

    if (config_type == "kvm_guest.config") {
        return force_graphics_config(token);
    }
    return {};
}

auto KernelBuilder::force_graphics_config(std::stop_token token) -> VoidResult {
    auto& cfg = ctx_->config();
    auto script = (fs::path(cfg.paths.kernel_dir) / "scripts" / "config").string();
    auto config_file = (fs::path(cfg.paths.kernel_dir) / ".config").string();

    std::vector<std::string> script_args = {
        "--file",   config_file,
        "--enable", "CONFIG_DRM",
        "--enable", "CONFIG_DRM_VIRTIO_GPU",
        "--enable", "CONFIG_FB",
        "--enable", "CONFIG_FRAMEBUFFER_CONSOLE",
    };

    auto r = ctx_->exec().run(token, script, script_args);
    if (!r)
        return r;

    // Finalize with olddefconfig
    auto env = ctx_->get_make_env();
    std::vector<std::string> args = {
        "-C", cfg.paths.kernel_dir, "ARCH=" + cfg.build.arch, "LLVM=1", "olddefconfig",
    };
    append_host_make_flags(ctx_, args);
    return ctx_->exec().run_with_env(token, env, "make", args);
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
    std::vector<std::string> args = {
        "-C", cfg.paths.kernel_dir, "ARCH=" + cfg.build.arch, "LLVM=1", "olddefconfig",
    };
    append_host_make_flags(ctx_, args);
    return ctx_->exec().run_with_env(token, env, "make", args);
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

auto KernelBuilder::force_qemu_boot_config(std::stop_token token) -> VoidResult {
    auto& cfg = ctx_->config();
    auto script = (fs::path(cfg.paths.kernel_dir) / "scripts" / "config").string();
    auto config_file = (fs::path(cfg.paths.kernel_dir) / ".config").string();

    if (!ctx_->fs().exists(script) || !ctx_->fs().exists(config_file)) {
        return {};
    }

    std::vector<std::string> script_args = {"--file", config_file};
    for (const auto& sym : qemu_boot_symbols()) {
        script_args.push_back("--enable");
        script_args.push_back(sym);
    }

    auto script_result = ctx_->exec().run(token, script, script_args);
    if (!script_result) {
        return make_error(Error::wrap("force QEMU boot kernel config", script_result.error()));
    }

    auto env = ctx_->get_make_env();
    std::vector<std::string> finalize_args = {
        "-C", cfg.paths.kernel_dir, "ARCH=" + cfg.build.arch, "LLVM=1", "olddefconfig",
    };
    append_host_make_flags(ctx_, finalize_args);

    auto finalize = ctx_->exec().run_with_env(token, env, "make", finalize_args);
    if (!finalize) {
        return make_error(Error::wrap("finalize QEMU boot kernel config", finalize.error()));
    }

    return {};
}

}  // namespace elmos::domain::builder
