// ============================================================================
// domain/builder/module.cpp — Kernel module build orchestration
// ============================================================================

#include "module.hpp"

#include <config/types.hpp>
#include <context/context.hpp>
#include <domain/toolchain/manager.hpp>

#include "kernel.hpp"

#include <filesystem>
#include <sstream>

namespace elmos::domain::builder {

namespace fs = std::filesystem;

ModuleBuilder::ModuleBuilder(context::Context* ctx, toolchain::Manager* tm) : ctx_(ctx), tm_(tm) {}

auto ModuleBuilder::build(std::stop_token token, const std::string& name) -> VoidResult {
    auto modules = get_modules(name);
    if (!modules)
        return make_error(modules.error());
    for (const auto& mod : *modules) {
        auto r = build_module(token, mod);
        if (!r)
            return r;
    }
    return {};
}

auto ModuleBuilder::build_module(std::stop_token token, const ModuleInfo& mod) -> VoidResult {
    auto& cfg = ctx_->config();
    auto env = ctx_->get_make_env();

    return ctx_->exec().run_with_env(token, env, "make",
                                     {
                                         "-C",
                                         cfg.paths.kernel_dir,
                                         "M=" + mod.path,
                                         "ARCH=" + cfg.build.arch,
                                         "LLVM=1",
                                         "modules",
                                     });
}

auto ModuleBuilder::clean(std::stop_token token, const std::string& name) -> VoidResult {
    auto modules = get_modules(name);
    if (!modules)
        return make_error(modules.error());
    auto& cfg = ctx_->config();
    for (const auto& mod : *modules) {
        ctx_->exec().run(token, "make",
                         {
                             "-C",
                             cfg.paths.kernel_dir,
                             "M=" + mod.path,
                             "ARCH=" + cfg.build.arch,
                             "clean",
                         });
    }
    return {};
}

auto ModuleBuilder::get_modules(const std::string& name) -> Result<std::vector<ModuleInfo>> {
    auto& cfg = ctx_->config();
    if (!ctx_->fs().exists(cfg.paths.modules_dir)) {
        return std::vector<ModuleInfo>{};
    }

    if (!name.empty()) {
        auto path = (fs::path(cfg.paths.modules_dir) / name).string();
        if (!ctx_->fs().exists(path)) {
            return make_error(Error::generic("module not found: " + name));
        }
        return std::vector{get_module_info(name, path)};
    }

    auto entries = ctx_->fs().read_dir(cfg.paths.modules_dir);
    if (!entries)
        return make_error(entries.error());

    std::vector<ModuleInfo> modules;
    for (const auto& entry : *entries) {
        if (!entry.is_directory)
            continue;
        auto path = (fs::path(cfg.paths.modules_dir) / entry.name).string();
        auto make_path = (fs::path(path) / "Makefile").string();
        if (!ctx_->fs().exists(make_path))
            continue;
        modules.push_back(get_module_info(entry.name, path));
    }
    return modules;
}

auto ModuleBuilder::get_module_info(const std::string& name, const std::string& path)
    -> ModuleInfo {
    ModuleInfo info{.name = name, .path = path};
    auto ko = (fs::path(path) / (name + ".ko")).string();
    info.built = ctx_->fs().exists(ko);

    auto src = (fs::path(path) / (name + ".c")).string();
    auto content = ctx_->fs().read_file(src);
    if (content) {
        info.description = extract_description(*content);
    }
    return info;
}

auto ModuleBuilder::extract_description(const std::string& content) -> std::string {
    std::istringstream stream(content);
    std::string line;
    while (std::getline(stream, line)) {
        if (line.find("MODULE_DESCRIPTION") != std::string::npos) {
            auto start = line.find('"');
            auto end = line.rfind('"');
            if (start != std::string::npos && end > start) {
                return line.substr(start + 1, end - start - 1);
            }
        }
    }
    return "";
}

auto ModuleBuilder::prepare_headers(std::stop_token token) -> VoidResult {
    auto& cfg = ctx_->config();
    auto env = ctx_->get_make_env();
    return ctx_->exec().run_with_env(token, env, "make",
                                     {
                                         "-C",
                                         cfg.paths.kernel_dir,
                                         "-j" + std::to_string(cfg.build.jobs),
                                         "ARCH=" + cfg.build.arch,
                                         "LLVM=1",
                                         "modules_prepare",
                                     });
}

auto ModuleBuilder::create_module(const std::string& name) -> VoidResult {
    auto& cfg = ctx_->config();
    auto mod_path = (fs::path(cfg.paths.modules_dir) / name).string();

    if (ctx_->fs().exists(mod_path)) {
        return make_error(Error::generic("module already exists: " + name));
    }

    auto r = ctx_->fs().mkdir_all(mod_path);
    if (!r)
        return r;

    // Create minimal module source
    std::string c_name = name;
    for (auto& c : c_name) {
        if (c == '-')
            c = '_';
    }

    std::string src = "#include <linux/init.h>\n#include <linux/module.h>\n\n"
                      "static int __init " +
                      c_name +
                      "_init(void) {\n"
                      "    pr_info(\"" +
                      name +
                      " loaded\\n\");\n    return 0;\n}\n\n"
                      "static void __exit " +
                      c_name +
                      "_exit(void) {\n"
                      "    pr_info(\"" +
                      name +
                      " unloaded\\n\");\n}\n\n"
                      "module_init(" +
                      c_name +
                      "_init);\n"
                      "module_exit(" +
                      c_name +
                      "_exit);\n\n"
                      "MODULE_LICENSE(\"GPL\");\n"
                      "MODULE_DESCRIPTION(\"A simple kernel module\");\n";

    auto wr = ctx_->fs().write_file((fs::path(mod_path) / (c_name + ".c")).string(), src);
    if (!wr)
        return wr;

    std::string makefile = "obj-m := " + c_name + ".o\n";
    return ctx_->fs().write_file((fs::path(mod_path) / "Makefile").string(), makefile);
}

}  // namespace elmos::domain::builder
