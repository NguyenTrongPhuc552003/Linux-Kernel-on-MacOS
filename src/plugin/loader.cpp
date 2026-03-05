// ============================================================================
// plugin/loader.cpp — Plugin registry and loader implementation
// ============================================================================

#include "loader.hpp"

namespace elmos::plugin {

auto builtin_factories() -> std::unordered_map<std::string, PluginFactory>& {
    static std::unordered_map<std::string, PluginFactory> factories;
    return factories;
}

void register_builtin_factory(const std::string& name, PluginFactory factory) {
    builtin_factories()[name] = std::move(factory);
}

// ---- Registry ----

Registry::Registry(HookExecutor* executor, bool verbose) : executor_(executor), verbose_(verbose) {}

auto Registry::load_builtins(context::Context* ctx, const AnyMap& config) -> VoidResult {
    static const std::vector<std::string> builtin_names = {
        "kernel-builder",  "bsp-manager",    "bootloader-builder",
        "caching-backend", "rootfs-builder", "image-assembler",
    };

    for (const auto& name : builtin_names) {
        if (is_disabled(config, name))
            continue;

        auto it = builtin_factories().find(name);
        if (it == builtin_factories().end())
            continue;

        auto plugin = it->second();
        if (!plugin)
            continue;

        auto plugin_cfg = extract_plugin_config(config, name);
        auto result = register_plugin(ctx, std::move(plugin), plugin_cfg);
        if (!result) {
            // Log but don't fail
        }
    }

    return {};
}

auto Registry::register_plugin(context::Context* ctx, std::unique_ptr<Plugin> plugin,
                               const AnyMap& config) -> VoidResult {
    if (!plugin) {
        return make_error(Error::generic("plugin is null"));
    }

    auto name = plugin->name();

    if (plugins_.contains(name)) {
        return make_error(Error::generic("plugin already loaded: " + name));
    }

    auto r = plugin->init(ctx, config);
    if (!r)
        return make_error(Error::wrap("plugin " + name + " init failed", r.error()));

    r = plugin->validate();
    if (!r)
        return make_error(Error::wrap("plugin " + name + " validation failed", r.error()));

    auto hooks = plugin->hooks();
    executor_->register_batch(name, hooks);

    plugins_[name] = std::move(plugin);
    return {};
}

auto Registry::get(const std::string& name) -> Plugin* {
    auto it = plugins_.find(name);
    if (it == plugins_.end())
        return nullptr;
    return it->second.get();
}

auto Registry::has_plugin(const std::string& name) const -> bool {
    return plugins_.contains(name);
}

auto Registry::list_plugins() const -> std::vector<Plugin*> {
    std::vector<Plugin*> result;
    result.reserve(plugins_.size());
    for (const auto& [_, p] : plugins_) {
        result.push_back(p.get());
    }
    return result;
}

auto Registry::count() const -> size_t {
    return plugins_.size();
}

auto Registry::cleanup_all() -> VoidResult {
    for (auto& [name, plugin] : plugins_) {
        auto r = plugin->cleanup();
        if (!r) {
            return make_error(Error::wrap("plugin " + name + " cleanup failed", r.error()));
        }
    }
    return {};
}

auto Registry::is_disabled(const AnyMap& config, const std::string& name) -> bool {
    auto it = config.find(name);
    if (it == config.end())
        return false;
    // AnyMap values are strings — check for "false", "0", or "disabled"
    const auto& val = it->second;
    return val == "false" || val == "0" || val == "disabled";
}

auto Registry::extract_plugin_config(const AnyMap& /*config*/, const std::string& /*name*/)
    -> AnyMap {
    // Plugin-specific config extraction — Phase 3
    // AnyMap is string→string; nested config requires a different approach
    return {};
}

}  // namespace elmos::plugin
