// ============================================================================
// domain/builder/app.cpp — Userspace application build orchestration
// ============================================================================

#include "app.hpp"

#include <config/arch.hpp>
#include <config/types.hpp>
#include <context/context.hpp>
#include <domain/toolchain/manager.hpp>

#include <filesystem>

namespace elmos::domain::builder {

namespace fs = std::filesystem;

AppBuilder::AppBuilder(context::Context* ctx, toolchain::Manager* tm) : ctx_(ctx), tm_(tm) {}

auto AppBuilder::build(std::stop_token token, const std::string& name) -> VoidResult {
    auto apps = get_apps(name);
    if (!apps)
        return make_error(apps.error());
    if (apps->empty())
        return {};

    auto env = ctx_->get_make_env();
    auto compiler = get_cross_compiler(ctx_->config().build.cross_compile);

    for (const auto& app : *apps) {
        auto r = build_app(token, app, compiler, env);
        if (!r)
            return r;
    }
    return {};
}

auto AppBuilder::build_app(std::stop_token token, const AppInfo& app, const std::string& compiler,
                           const EnvList& env) -> VoidResult {
    auto& cfg = ctx_->config();
    auto makefile = (fs::path(app.path) / "Makefile").string();

    if (ctx_->fs().exists(makefile)) {
        return ctx_->exec().run_with_env_in_dir(token, env, app.path, "make",
                                                {
                                                    "CC=" + compiler,
                                                    "ARCH=" + cfg.build.arch,
                                                });
    }

    auto src = (fs::path(app.path) / (app.name + ".c")).string();
    if (!ctx_->fs().exists(src)) {
        return make_error(Error::generic("no source file found for " + app.name));
    }

    auto out = (fs::path(app.path) / app.name).string();
    return ctx_->exec().run_with_env(token, env, compiler, {"-static", "-o", out, src});
}

auto AppBuilder::clean(std::stop_token token, const std::string& name) -> VoidResult {
    auto apps = get_apps(name);
    if (!apps)
        return make_error(apps.error());

    for (const auto& app : *apps) {
        auto makefile = (fs::path(app.path) / "Makefile").string();
        if (ctx_->fs().exists(makefile)) {
            ctx_->exec().run_in_dir(token, app.path, "make", {"clean"});
        }
        else {
            ctx_->fs().remove((fs::path(app.path) / app.name).string());
        }
    }
    return {};
}

auto AppBuilder::get_apps(const std::string& name) -> Result<std::vector<AppInfo>> {
    auto& cfg = ctx_->config();
    if (!ctx_->fs().exists(cfg.paths.apps_dir)) {
        return std::vector<AppInfo>{};
    }

    if (!name.empty()) {
        auto path = (fs::path(cfg.paths.apps_dir) / name).string();
        if (!ctx_->fs().exists(path)) {
            return make_error(Error::generic("app not found: " + name));
        }
        return std::vector{get_app_info(name, path)};
    }

    auto entries = ctx_->fs().read_dir(cfg.paths.apps_dir);
    if (!entries)
        return make_error(entries.error());

    std::vector<AppInfo> apps;
    for (const auto& entry : *entries) {
        if (!entry.is_directory)
            continue;
        auto path = (fs::path(cfg.paths.apps_dir) / entry.name).string();
        auto src = (fs::path(path) / (entry.name + ".c")).string();
        auto makefile = (fs::path(path) / "Makefile").string();
        if (!ctx_->fs().exists(src) && !ctx_->fs().exists(makefile))
            continue;
        apps.push_back(get_app_info(entry.name, path));
    }
    return apps;
}

auto AppBuilder::get_app_info(const std::string& name, const std::string& path) -> AppInfo {
    auto bin = (fs::path(path) / name).string();
    return {.name = name, .path = path, .built = ctx_->fs().exists(bin)};
}

auto AppBuilder::get_cross_compiler(const std::string& prefix) -> std::string {
    if (!prefix.empty())
        return prefix + "gcc";
    auto arch_cfg = config::get_arch_config(ctx_->config().build.arch);
    if (arch_cfg && !arch_cfg->gcc_binary.empty()) {
        std::stop_source ss;
        auto r = ctx_->exec().look_path(arch_cfg->gcc_binary);
        if (r)
            return *r;
    }
    return "clang";
}

auto AppBuilder::create_app(const std::string& name) -> VoidResult {
    auto& cfg = ctx_->config();
    auto app_path = (fs::path(cfg.paths.apps_dir) / name).string();

    if (ctx_->fs().exists(app_path)) {
        return make_error(Error::generic("app already exists: " + name));
    }

    auto r = ctx_->fs().mkdir_all(app_path);
    if (!r)
        return r;

    std::string c_name = name;
    for (auto& c : c_name) {
        if (c == '-')
            c = '_';
    }

    std::string src = "#include <stdio.h>\n\nint main(void) {\n"
                      "    printf(\"Hello from " +
                      name + "!\\n\");\n    return 0;\n}\n";

    auto wr = ctx_->fs().write_file((fs::path(app_path) / (c_name + ".c")).string(), src);
    if (!wr)
        return wr;

    std::string makefile = "CC ?= gcc\nTARGET := " + c_name +
                           "\n\n"
                           "all: $(TARGET)\n\n$(TARGET): $(TARGET).c\n\t$(CC) -static -o $@ $<\n\n"
                           "clean:\n\trm -f $(TARGET)\n\n.PHONY: all clean\n";

    return ctx_->fs().write_file((fs::path(app_path) / "Makefile").string(), makefile);
}

}  // namespace elmos::domain::builder
