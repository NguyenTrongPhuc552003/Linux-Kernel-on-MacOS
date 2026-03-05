// ============================================================================
// domain/toolchain/manager.cpp — Cross-compilation toolchain management
// ============================================================================

#include "manager.hpp"

#include <config/types.hpp>

#include <cstdlib>
#include <filesystem>

namespace elmos::domain::toolchain {

namespace fs = std::filesystem;

Manager::Manager(infra::executor::Executor* exec, infra::filesystem::FileSystem* fs_arg,
                 config::Config* cfg)
    : exec_(exec), fs_(fs_arg), cfg_(cfg) {
    init_paths();
}

void Manager::init_paths() {
    const char* home = std::getenv("HOME");
    std::string base = home ? fs::path(home) / ".elmos" / "toolchains" : "/tmp/elmos/toolchains";

    paths_ = {
        .base_dir = base,
        .x_tools = (fs::path(base) / "x-tools").string(),
        .config_dir = (fs::path(base) / "configs").string(),
        .build_dir = (fs::path(base) / "build").string(),
        .ct_ng_dir = (fs::path(base) / "crosstool-ng").string(),
    };
}

auto Manager::is_installed() const -> bool {
    auto r = exec_->look_path("ct-ng");
    return r.has_value();
}

auto Manager::install(std::stop_token token) -> VoidResult {
    // Clone and build crosstool-ng from source
    if (!fs_->exists(paths_.ct_ng_dir)) {
        auto r = fs_->mkdir_all(paths_.ct_ng_dir);
        if (!r)
            return r;

        auto result = exec_->run(
            token, "git",
            {"clone", "https://github.com/crosstool-ng/crosstool-ng.git", paths_.ct_ng_dir});
        if (!result)
            return result;
    }

    auto r1 = exec_->run_in_dir(token, paths_.ct_ng_dir, "./bootstrap", {});
    if (!r1)
        return r1;

    auto r2 = exec_->run_in_dir(token, paths_.ct_ng_dir, "./configure",
                                {"--prefix=" + paths_.base_dir});
    if (!r2)
        return r2;

    auto r3 = exec_->run_in_dir(token, paths_.ct_ng_dir, "make", {});
    if (!r3)
        return r3;

    return exec_->run_in_dir(token, paths_.ct_ng_dir, "make", {"install"});
}

auto Manager::build_toolchain(std::stop_token token, const std::string& target) -> VoidResult {
    auto build_dir = (fs::path(paths_.build_dir) / target).string();
    auto r = fs_->mkdir_all(build_dir);
    if (!r)
        return r;

    // Copy config
    auto config_src = (fs::path(paths_.config_dir) / (target + ".config")).string();
    if (!fs_->exists(config_src)) {
        return make_error(Error::config("toolchain config not found: " + config_src));
    }

    auto config_content = fs_->read_file(config_src);
    if (!config_content)
        return make_error(config_content.error());

    auto config_dest = (fs::path(build_dir) / ".config").string();
    auto wr = fs_->write_file(config_dest, *config_content);
    if (!wr)
        return wr;

    // Build
    EnvList env = {"CT_PREFIX=" + paths_.x_tools};
    return exec_->run_with_env_in_dir(token, env, build_dir, "ct-ng", {"build"});
}

auto Manager::get_installed_toolchains() -> Result<std::vector<ToolchainInfo>> {
    std::vector<ToolchainInfo> toolchains;

    if (!fs_->exists(paths_.x_tools))
        return toolchains;

    auto entries = fs_->read_dir(paths_.x_tools);
    if (!entries)
        return make_error(entries.error());

    for (const auto& entry : *entries) {
        if (!entry.is_directory)
            continue;
        auto bin_dir = (fs::path(paths_.x_tools) / entry.name / "bin").string();
        toolchains.push_back({
            .target = entry.name,
            .installed = fs_->is_dir(bin_dir),
        });
    }

    return toolchains;
}

auto Manager::get_available_configs() -> Result<std::vector<std::string>> {
    std::vector<std::string> configs;
    if (!fs_->exists(paths_.config_dir))
        return configs;

    auto entries = fs_->read_dir(paths_.config_dir);
    if (!entries)
        return make_error(entries.error());

    for (const auto& entry : *entries) {
        if (!entry.is_directory && entry.name.ends_with(".config")) {
            configs.push_back(entry.name.substr(0, entry.name.size() - 7));
        }
    }

    return configs;
}

auto Manager::get_bin_dir(const std::string& target) -> std::string {
    return (fs::path(paths_.x_tools) / target / "bin").string();
}

}  // namespace elmos::domain::toolchain
