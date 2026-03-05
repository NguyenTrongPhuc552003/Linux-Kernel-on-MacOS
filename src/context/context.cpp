// ============================================================================
// context/context.cpp — Build context implementation
// ============================================================================

#include "context.hpp"

#include <config/arch.hpp>
#include <infra/executor/env.hpp>

#include <cstdlib>
#include <filesystem>
#include <sstream>

namespace elmos::context {

Context::Context(config::Config* cfg, infra::executor::Executor* exec,
                 infra::filesystem::FileSystem* filesystem)
    : config_(cfg), exec_(exec), fs_(filesystem) {
    platform_owned_ = infra::platform::create_platform();
    platform_ = platform_owned_.get();
    platform_->set_executor(exec_);

#ifdef ELMOS_PLATFORM_DARWIN
    brew_ = std::make_unique<infra::homebrew::Resolver>(exec_);
#endif
}

void Context::set_platform(std::unique_ptr<infra::platform::Platform> p) {
    platform_owned_ = std::move(p);
    platform_ = platform_owned_.get();
    platform_->set_executor(exec_);
}

auto Context::is_mounted(std::stop_token token) -> bool {
    auto name = config_->image.volume_name;
    auto result = platform_->disk_image().is_mounted(token, name);
    if (!result)
        return false;
    return result->mounted;
}

auto Context::ensure_mounted(std::stop_token token) -> VoidResult {
    if (!is_mounted(token)) {
        return make_error(Error::image("kernel volume not mounted"));
    }
    return {};
}

auto Context::get_actual_mount_point(std::stop_token token) -> Result<std::string> {
    // Fast path
    if (fs_->is_dir(config_->image.mount_point)) {
        return config_->image.mount_point;
    }
    auto name = config_->image.volume_name;
    auto result = platform_->disk_image().is_mounted(token, name);
    if (!result)
        return make_error(result.error());
    if (!result->mounted) {
        return make_error(Error::image("image not mounted: " + config_->image.path));
    }
    return result->mount_point;
}

auto Context::kernel_exists() -> bool {
    auto git_dir = (fs::path(config_->paths.kernel_dir) / ".git").string();
    return fs_->exists(git_dir);
}

auto Context::has_config() -> bool {
    auto config_file = (fs::path(config_->paths.kernel_dir) / ".config").string();
    return fs_->exists(config_file);
}

auto Context::get_kernel_image() -> std::string {
    auto arch_cfg = config_->get_arch_config();
    if (!arch_cfg)
        return "";
    return (fs::path(config_->paths.kernel_dir) / "arch" / arch_cfg->kernel_arch / "boot" /
            arch_cfg->kernel_image)
        .string();
}

auto Context::get_vmlinux() -> std::string {
    return (fs::path(config_->paths.kernel_dir) / "vmlinux").string();
}

auto Context::has_kernel_image() -> bool {
    return fs_->exists(get_kernel_image());
}

auto Context::get_default_targets() -> std::vector<std::string> {
    auto arch_cfg = config_->get_arch_config();
    if (!arch_cfg)
        return {"Image", "dtbs", "modules"};
    return arch_cfg->default_targets;
}

auto Context::get_make_env() -> EnvList {
    auto& cfg = *config_;
    auto env_map = infra::executor::get_current_env();

    std::string new_path;
    if (auto it = env_map.find("PATH"); it != env_map.end()) {
        new_path = it->second;
    }

#ifdef ELMOS_PLATFORM_DARWIN
    if (brew_) {
        new_path = prepend_brew_tool_paths(new_path);
    }
#endif

    EnvList env;
    for (const auto& [key, val] : env_map) {
        if (key != "PATH") {
            env.push_back(key + "=" + val);
        }
    }
    env.push_back("PATH=" + new_path);
    env.push_back("ARCH=" + cfg.build.arch);
    env.push_back("LLVM=1");
    env.push_back("CROSS_COMPILE=" + cfg.build.cross_compile);

    auto hostcflags = build_host_cflags();
    if (!hostcflags.empty()) {
        env.push_back("HOSTCFLAGS=" + hostcflags);
    }

    return env;
}

auto Context::prepend_brew_tool_paths(const std::string& current_path) -> std::string {
    if (!brew_)
        return current_path;
    std::string path = current_path;
    std::stop_source ss;
    auto token = ss.get_token();

    auto prepend = [&](const std::string& p) {
        if (!p.empty())
            path = p + ":" + path;
    };

    auto r1 = brew_->get_libexec_bin("gnu-sed");
    if (!r1.empty())
        prepend(r1);
    auto r2 = brew_->get_libexec_bin("coreutils");
    if (!r2.empty())
        prepend(r2);
    auto r3 = brew_->get_bin("llvm");
    if (!r3.empty())
        prepend(r3);
    auto r4 = brew_->get_bin("lld");
    if (!r4.empty())
        prepend(r4);
    auto r5 = brew_->get_sbin("e2fsprogs");
    if (!r5.empty())
        prepend(r5);

    return path;
}

auto Context::build_host_cflags() -> std::string {
#ifndef ELMOS_PLATFORM_DARWIN
    if (!config_->paths.libraries_dir.empty()) {
        return "-I" + config_->paths.libraries_dir;
    }
    return "";
#else
    std::vector<std::string> flags;
    if (!config_->paths.libraries_dir.empty()) {
        flags.push_back("-I" + config_->paths.libraries_dir);
    }
    if (brew_) {
        auto r = brew_->get_include("libelf");
        if (!r.empty()) {
            flags.push_back("-I" + r);
        }
    }
    flags.push_back("-D_UUID_T");
    flags.push_back("-D__GETHOSTUUID_H");
    flags.push_back("-D_DARWIN_C_SOURCE");
    flags.push_back("-D_FILE_OFFSET_BITS=64");

    std::string result;
    for (size_t i = 0; i < flags.size(); ++i) {
        if (i > 0)
            result += " ";
        result += flags[i];
    }
    return result;
#endif
}

}  // namespace elmos::context
