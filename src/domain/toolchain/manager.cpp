// ============================================================================
// domain/toolchain/manager.cpp — Cross-compilation toolchain management
// ============================================================================

#include "manager.hpp"

#include <config/arch.hpp>
#include <config/types.hpp>

#include <cstdlib>
#include <filesystem>
#include <optional>
#include <regex>
#include <sstream>
#include <thread>
#include <unordered_map>

namespace elmos::domain::toolchain {

namespace fs = std::filesystem;

namespace {

auto trim(std::string s) -> std::string {
    constexpr std::string_view ws = " \t\r\n";
    const auto start = s.find_first_not_of(ws);
    if (start == std::string::npos) {
        return "";
    }
    const auto end = s.find_last_not_of(ws);
    return s.substr(start, end - start + 1);
}

auto strip_quotes(std::string value) -> std::string {
    if (value.size() >= 2 && value.front() == '"' && value.back() == '"') {
        return value.substr(1, value.size() - 2);
    }
    return value;
}

auto normalized_arch(std::string ct_arch, std::string ct_arch_bitness) -> std::string {
    if (ct_arch == "aarch64") {
        return "arm64";
    }
    if (ct_arch == "arm" && ct_arch_bitness == "64") {
        return "arm64";
    }
    return ct_arch;
}

}  // namespace

Manager::Manager(infra::executor::Executor* exec, infra::filesystem::FileSystem* fs_arg,
                 infra::platform::Platform* platform, config::Config* cfg)
    : exec_(exec), fs_(fs_arg), platform_(platform), cfg_(cfg) {
    init_paths();
}

void Manager::init_paths() {
    // Toolchains live inside the workspace mount point, not in ~/.elmos/
    std::string base = cfg_->paths.toolchains_dir;
    if (base.empty()) {
        // Fallback if config not yet loaded
        const char* home = std::getenv("HOME");
        base = home ? (fs::path(home) / "x-tools").string() : "/tmp/elmos/toolchains";
    }

    paths_ = {
        .base_dir = base,
        .x_tools = (fs::path(base) / "x-tools").string(),
        .config_dir = (fs::path(base) / "configs").string(),
        .build_dir = (fs::path(base) / "build").string(),
        .ct_ng_dir = (fs::path(base) / "crosstool-ng").string(),
    };
}

auto Manager::ct_ng_bin() const -> std::string {
    return (fs::path(paths_.base_dir) / "bin" / "ct-ng").string();
}

auto Manager::is_installed() const -> bool {
    return fs_->exists(ct_ng_bin());
}

auto Manager::get_build_env() -> EnvList {
    auto env = platform_->packages().build_gnu_environment();

#ifdef __APPLE__
    // Prefer Homebrew GNU compilers when available to avoid clang++-specific
    // probe noise inside crosstool-ng final GCC stages.
    auto gcc_bin_dir = platform_->packages().get_bin_path("gcc");
    if (!gcc_bin_dir.empty() && fs::is_directory(gcc_bin_dir)) {
        std::string selected_cc;
        std::string selected_cxx;

        auto pick_highest = [](const std::vector<std::string>& candidates,
                               const std::string& stem) -> std::optional<std::string> {
            std::optional<std::pair<int, std::string>> best;
            std::regex versioned(stem + R"(-(\d+)$)");
            std::smatch m;

            for (const auto& path : candidates) {
                auto name = fs::path(path).filename().string();
                int version = -1;
                if (std::regex_match(name, m, versioned)) {
                    version = std::stoi(m[1].str());
                }
                if (!best || version > best->first) {
                    best = {version, path};
                }
            }

            if (!best) {
                return std::nullopt;
            }
            return best->second;
        };

        std::vector<std::string> cc_candidates;
        std::vector<std::string> cxx_candidates;

        for (const auto& entry : fs::directory_iterator(gcc_bin_dir)) {
            if (!entry.is_regular_file()) {
                continue;
            }

            auto name = entry.path().filename().string();
            if (name.starts_with("gcc-")) {
                cc_candidates.push_back(entry.path().string());
            }
            else if (name.starts_with("g++-")) {
                cxx_candidates.push_back(entry.path().string());
            }
        }

        if (auto picked = pick_highest(cc_candidates, "gcc"); picked) {
            selected_cc = *picked;
        }
        if (auto picked = pick_highest(cxx_candidates, "g\\+\\+"); picked) {
            selected_cxx = *picked;
        }

        // If g++-* was not discovered but gcc-* was, infer matching g++-* path.
        if (!selected_cc.empty() && selected_cxx.empty()) {
            auto cc_name = fs::path(selected_cc).filename().string();
            if (cc_name.starts_with("gcc-")) {
                auto inferred = (fs::path(gcc_bin_dir) / ("g++-" + cc_name.substr(4))).string();
                if (fs::exists(inferred)) {
                    selected_cxx = inferred;
                }
            }
        }

        if (!selected_cc.empty()) {
            env.push_back("CC=" + selected_cc);
            env.push_back("HOSTCC=" + selected_cc);
            env.push_back("CC_FOR_BUILD=" + selected_cc);
            env.push_back("BUILD_CC=" + selected_cc);
        }
        if (!selected_cxx.empty()) {
            env.push_back("CXX=" + selected_cxx);
            env.push_back("HOSTCXX=" + selected_cxx);
            env.push_back("CXX_FOR_BUILD=" + selected_cxx);
            env.push_back("BUILD_CXX=" + selected_cxx);
        }
    }
#endif

    // ct-ng configs use ${ELMOS_WORKSPACE} — keep for any residual references
    env.push_back("ELMOS_WORKSPACE=" + cfg_->image.volume_name);

    // CT_PREFIX must be set for ct-ng (Go: getBuildEnv sets CT_PREFIX=paths.XTools)
    env.push_back("CT_PREFIX=" + paths_.x_tools);

    // Include the workspace-local ct-ng bin dir in PATH so ct-ng
    // and its internal scripts can find themselves
    auto ct_ng_bin_dir = (fs::path(paths_.base_dir) / "bin").string();
    if (fs_->is_dir(ct_ng_bin_dir)) {
        // Find existing PATH entry and prepend, or create new one
        bool found = false;
        for (auto& entry : env) {
            if (entry.starts_with("PATH=")) {
                entry = "PATH=" + ct_ng_bin_dir + ":" + entry.substr(5);
                found = true;
                break;
            }
        }
        if (!found) {
            const char* sys_path = std::getenv("PATH");
            std::string path_val = ct_ng_bin_dir;
            if (sys_path) {
                path_val += ":";
                path_val += sys_path;
            }
            env.push_back("PATH=" + path_val);
        }
    }

    return env;
}

auto Manager::install(std::stop_token token) -> VoidResult {
    // Idempotent: skip if already installed
    if (is_installed()) {
        return {};
    }

    // Clone and build crosstool-ng from source
    if (!fs_->exists(paths_.ct_ng_dir)) {
        auto r = fs_->mkdir_all(paths_.ct_ng_dir);
        if (!r)
            return r;

        auto result =
            exec_->run(token, "git",
                       {"clone", "git@github.com:crosstool-ng/crosstool-ng.git", paths_.ct_ng_dir});
        if (!result)
            return result;
    }

    // Get platform-aware build environment (GNU tools PATH on macOS)
    auto build_env = get_build_env();

    auto run_step = [&](const std::string& dir, const std::string& cmd,
                        std::vector<std::string> args) -> VoidResult {
        if (build_env.empty())
            return exec_->run_in_dir(token, dir, cmd, std::move(args));
        return exec_->run_with_env_in_dir(token, build_env, dir, cmd, std::move(args));
    };

    auto r1 = run_step(paths_.ct_ng_dir, "./bootstrap", {});
    if (!r1)
        return r1;

    auto r2 = run_step(paths_.ct_ng_dir, "./configure", {"--prefix=" + paths_.base_dir});
    if (!r2)
        return r2;

    auto r3 = run_step(paths_.ct_ng_dir, "make", {});
    if (!r3)
        return r3;

    return run_step(paths_.ct_ng_dir, "make", {"install"});
}

auto Manager::resolve_target(const std::string& arch_or_target) -> Result<std::string> {
    // 1. Direct match: target.config exists as-is
    auto direct = (fs::path(paths_.config_dir) / (arch_or_target + ".config")).string();
    if (fs_->exists(direct)) {
        return arch_or_target;
    }

    // 2. Derive from ArchConfig::gcc_binary — strip "-gcc" suffix
    const auto* arch_cfg = config::get_arch_config(arch_or_target);
    if (arch_cfg && !arch_cfg->gcc_binary.empty()) {
        auto gcc = arch_cfg->gcc_binary;
        const std::string suffix = "-gcc";
        if (gcc.size() > suffix.size() && gcc.ends_with(suffix)) {
            auto target = gcc.substr(0, gcc.size() - suffix.size());
            auto path = (fs::path(paths_.config_dir) / (target + ".config")).string();
            if (fs_->exists(path)) {
                return target;
            }
        }
    }

    // 3. Fuzzy: scan configs/ for a file whose name starts with a prefix
    //    derived from the arch (e.g. "riscv" matches "riscv64-...")
    auto configs = get_available_configs();
    if (configs) {
        for (const auto& c : *configs) {
            if (c.starts_with(arch_or_target)) {
                return c;
            }
        }
    }

    return make_error(Error::config("no toolchain config found for '" + arch_or_target +
                                    "'. Available configs: " + [&]() {
                                        std::string list;
                                        if (configs) {
                                            for (const auto& c : *configs) {
                                                if (!list.empty())
                                                    list += ", ";
                                                list += c;
                                            }
                                        }
                                        return list.empty() ? "(none)" : list;
                                    }()));
}

auto Manager::build_toolchain(std::stop_token token, const std::string& target) -> VoidResult {
    // Resolve arch name to actual config target
    auto resolved = resolve_target(target);
    if (!resolved)
        return make_error(resolved.error());

    const auto& actual_target = *resolved;
    auto build_dir = (fs::path(paths_.build_dir) / actual_target).string();
    auto r = fs_->mkdir_all(build_dir);
    if (!r)
        return r;

    // Copy config
    auto config_src = (fs::path(paths_.config_dir) / (actual_target + ".config")).string();
    if (!fs_->exists(config_src)) {
        return make_error(Error::config("toolchain config not found: " + config_src));
    }

    auto config_content = fs_->read_file(config_src);
    if (!config_content)
        return make_error(config_content.error());

    // Patch config with absolute paths (Go: patchConfig / patchConfigContent)
    auto patched = patch_config(*config_content);

    auto config_dest = (fs::path(build_dir) / ".config").string();
    auto wr = fs_->write_file(config_dest, patched);
    if (!wr)
        return wr;

    // Ensure tarballs cache directory exists (CT_LOCAL_TARBALLS_DIR)
    auto src_dir = (fs::path(paths_.base_dir) / "src").string();
    fs_->mkdir_all(src_dir);

    // Determine parallel jobs (Go: build.<jobs>, defaults to NumCPU)
    int jobs = cfg_->build.jobs;
    if (jobs <= 0) {
        jobs = static_cast<int>(std::thread::hardware_concurrency());
        if (jobs <= 0)
            jobs = 1;
    }
    auto build_target = "build." + std::to_string(jobs);

    // Build with GNU tools env (includes CT_PREFIX, ELMOS_WORKSPACE, GNU paths)
    auto build_env = get_build_env();
    auto build_result = exec_->run_with_env_in_dir(token, build_env, build_dir, ct_ng_bin(),
                                                   {build_target});
    if (!build_result)
        return build_result;

    // Post-build verification: ensure the toolchain binaries were actually produced
    return verify_toolchain(token, actual_target);
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

auto Manager::get_targets_for_arch(const std::string& arch) -> Result<std::vector<std::string>> {
    auto configs = get_available_configs();
    if (!configs) {
        return make_error(configs.error());
    }

    std::vector<std::string> filtered;
    for (const auto& target : *configs) {
        auto details = get_config_details(target);
        if (!details) {
            continue;
        }

        if (normalized_arch(details->ct_arch, details->ct_arch_bitness) == arch) {
            filtered.push_back(target);
        }
    }

    if (filtered.empty()) {
        auto resolved = resolve_target(arch);
        if (resolved) {
            filtered.push_back(*resolved);
        }
    }

    return filtered;
}

auto Manager::get_config_details(const std::string& target) -> Result<ToolchainConfigDetails> {
    auto config_file = (fs::path(paths_.config_dir) / (target + ".config")).string();
    if (!fs_->exists(config_file)) {
        return make_error(Error::config("toolchain config not found: " + config_file));
    }

    auto content = fs_->read_file(config_file);
    if (!content) {
        return make_error(content.error());
    }

    std::unordered_map<std::string, std::string> kv;
    std::istringstream in(*content);
    std::string line;
    while (std::getline(in, line)) {
        auto candidate = trim(line);
        if (candidate.empty() || candidate[0] == '#') {
            continue;
        }

        auto eq = candidate.find('=');
        if (eq == std::string::npos || eq == 0) {
            continue;
        }

        auto key = trim(candidate.substr(0, eq));
        auto value = strip_quotes(trim(candidate.substr(eq + 1)));
        if (!key.empty()) {
            kv[key] = value;
        }
    }

    auto read_key = [&](const std::string& key) -> std::string {
        auto it = kv.find(key);
        if (it == kv.end()) {
            return "";
        }
        return it->second;
    };

    auto libc = read_key("CT_LIBC");
    std::string libc_version;
    if (libc == "glibc") {
        libc_version = read_key("CT_GLIBC_VERSION");
    }
    else if (libc == "musl") {
        libc_version = read_key("CT_MUSL_VERSION");
    }
    else if (libc == "uclibc-ng") {
        libc_version = read_key("CT_UCLIBC_NG_VERSION");
    }

    auto prefix_dir = read_key("CT_PREFIX_DIR");
    if (!prefix_dir.empty()) {
        const std::string ct_target_var = "${CT_TARGET}";
        auto pos = prefix_dir.find(ct_target_var);
        if (pos != std::string::npos) {
            prefix_dir.replace(pos, ct_target_var.size(), target);
        }

        const std::string ws_var = "${ELMOS_WORKSPACE}";
        pos = prefix_dir.find(ws_var);
        if (pos != std::string::npos) {
            prefix_dir.replace(pos, ws_var.size(), cfg_->image.volume_name);
        }
    }

    return ToolchainConfigDetails{
        .target = target,
        .config_file = config_file,
        .ct_arch = read_key("CT_ARCH"),
        .ct_arch_bitness = read_key("CT_ARCH_BITNESS"),
        .gcc_version = read_key("CT_GCC_VERSION"),
        .libc = libc,
        .libc_version = libc_version,
        .prefix_dir = prefix_dir,
    };
}

auto Manager::get_bin_dir(const std::string& target) -> std::string {
    return (fs::path(paths_.x_tools) / target / "bin").string();
}

auto Manager::patch_config(const std::string& content) const -> std::string {
    std::string result = content;

    // Helper: replace the value of a CT config key (KEY="old") -> (KEY="new")
    auto replace_config_value = [&](const std::string& key, const std::string& new_value) {
        // Pattern: KEY="<anything>"
        std::regex pattern(key + R"(="[^"]*")");
        result = std::regex_replace(result, pattern, key + "=\"" + new_value + "\"");
    };

    // Patch CT_PREFIX_DIR to absolute path (Go: paths.XTools/${CT_TARGET})
    replace_config_value("CT_PREFIX_DIR", paths_.x_tools + "/${CT_TARGET}");

    // Patch CT_LOCAL_TARBALLS_DIR to absolute path (Go: paths.Src)
    auto src_dir = (fs::path(paths_.base_dir) / "src").string();
    replace_config_value("CT_LOCAL_TARBALLS_DIR", src_dir);

    // Also replace any remaining ${ELMOS_WORKSPACE} references with the actual volume name
    std::string placeholder = "${ELMOS_WORKSPACE}";
    std::string::size_type pos = 0;
    while ((pos = result.find(placeholder, pos)) != std::string::npos) {
        result.replace(pos, placeholder.size(), cfg_->image.volume_name);
        pos += cfg_->image.volume_name.size();
    }

#ifdef __APPLE__
    // macOS: disable CT_COMP_TOOLS to avoid Clang/GCC mixing issues
    // (Go equivalent: disableCompTools / patchConfigContent)
    // ct-ng tries to build m4, make, autoconf, automake, libtool as companion
    // tools but these fail on macOS. Use Homebrew-installed versions instead.
    static const std::vector<std::string> comp_tools = {
        "CT_COMP_TOOLS_M4",       "CT_COMP_TOOLS_MAKE",    "CT_COMP_TOOLS_AUTOCONF",
        "CT_COMP_TOOLS_AUTOMAKE", "CT_COMP_TOOLS_LIBTOOL",
    };
    for (const auto& tool : comp_tools) {
        std::regex tool_pat(tool + "=y");
        result = std::regex_replace(result, tool_pat, "# " + tool + " is not set");
    }

    // macOS: GDB's bundled zlib conflicts with macOS SDK _stdio.h headers.
    // Tell GDB to use the host zlib (already built as ct-ng companion lib)
    // instead of compiling its own bundled copy.
    replace_config_value("CT_GDB_CROSS_EXTRA_CONFIG_ARRAY", "--with-system-zlib");
#endif

    return result;
}

auto Manager::verify_toolchain(std::stop_token token, const std::string& target) -> VoidResult {
    auto bin_dir = get_bin_dir(target);
    if (!fs_->is_dir(bin_dir)) {
        return make_error(
            Error::build("toolchain verification failed: bin directory not found at " + bin_dir));
    }

    // Check that the GCC binary exists
    auto gcc_bin = (fs::path(bin_dir) / (target + "-gcc")).string();
    if (!fs_->exists(gcc_bin)) {
        return make_error(
            Error::build("toolchain verification failed: gcc not found at " + gcc_bin));
    }

    // Verify the compiler can report its version (sanity check)
    auto version_result = exec_->output(token, gcc_bin, {"--version"});
    if (!version_result) {
        return make_error(Error::build("toolchain verification failed: " + gcc_bin +
                                       " --version returned error"));
    }

    return {};
}

}  // namespace elmos::domain::toolchain
