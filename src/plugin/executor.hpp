#pragma once
// ============================================================================
// plugin/executor.hpp — Hook execution engine
// Replaces Go's core/plugin/executor.go
// ============================================================================

#include "interface.hpp"

#include <algorithm>
#include <functional>
#include <mutex>
#include <stop_token>
#include <string>
#include <vector>

namespace elmos::plugin {

/// Manages hook registration and priority-ordered execution.
class HookExecutor {
public:
    HookExecutor() = default;
    explicit HookExecutor(bool verbose) : verbose_(verbose) {}

    /// Register a single hook for a plugin.
    void register_hook(const std::string& plugin_name, const std::string& event,
                       HookRegistration registration);

    /// Register all hooks for a plugin.
    void register_batch(const std::string& plugin_name,
                        const std::vector<HookRegistration>& registrations);

    /// Execute all hooks for an event, in priority order (higher first).
    auto execute(std::stop_token token, Event& event) -> VoidResult;

    /// Check if any hooks are registered for an event.
    auto has_hooks(const std::string& event) const -> bool;

    /// Get registered events.
    auto get_registered_events() const -> std::vector<std::string>;

    /// Clear all hooks (testing).
    void clear();

    void set_verbose(bool v) { verbose_ = v; }

private:
    struct RegisteredHook {
        std::string plugin_name;
        HookFunc handler;
        int priority = 5;
        bool required = true;
    };

    mutable std::mutex mu_;
    std::unordered_map<std::string, std::vector<RegisteredHook>> hooks_;
    bool verbose_ = false;
};

}  // namespace elmos::plugin
