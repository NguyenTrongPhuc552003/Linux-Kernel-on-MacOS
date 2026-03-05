#pragma once
// ============================================================================
// plugin/loader.hpp — Plugin registry and loader
// Replaces Go's core/plugin/loader.go
// ============================================================================

#include "executor.hpp"
#include "interface.hpp"

#include <memory>
#include <string>
#include <unordered_map>
#include <vector>

namespace elmos::context {
class Context;
}

namespace elmos::plugin {

/// Registry holds all loaded plugins and manages their lifecycle.
class Registry {
public:
    explicit Registry(HookExecutor* executor, bool verbose = false);

    /// Load all builtin plugins, passing context and config.
    auto load_builtins(context::Context* ctx, const AnyMap& config) -> VoidResult;

    /// Register and initialize a single plugin.
    auto register_plugin(context::Context* ctx, std::unique_ptr<Plugin> plugin,
                         const AnyMap& config) -> VoidResult;

    // Query
    auto get(const std::string& name) -> Plugin*;
    auto has_plugin(const std::string& name) const -> bool;
    auto list_plugins() const -> std::vector<Plugin*>;
    auto count() const -> size_t;

    /// Get the hook executor.
    auto get_executor() -> HookExecutor* { return executor_; }

    /// Cleanup all plugins.
    auto cleanup_all() -> VoidResult;

private:
    auto is_disabled(const AnyMap& config, const std::string& name) -> bool;
    auto extract_plugin_config(const AnyMap& config, const std::string& name) -> AnyMap;

    std::unordered_map<std::string, std::unique_ptr<Plugin>> plugins_;
    HookExecutor* executor_;
    [[maybe_unused]] bool verbose_;
};

/// Access the global builtin factory map (populated during static init).
auto builtin_factories() -> std::unordered_map<std::string, PluginFactory>&;

}  // namespace elmos::plugin
