// ============================================================================
// plugin/executor.cpp — Hook execution engine
// ============================================================================

#include "executor.hpp"

#include <algorithm>

namespace elmos::plugin {

void HookExecutor::register_hook(const std::string& plugin_name, const std::string& event,
                                 HookRegistration registration) {
    if (!registration.handler)
        return;

    std::lock_guard lock(mu_);
    auto& vec = hooks_[event];
    vec.push_back({
        .plugin_name = plugin_name,
        .handler = std::move(registration.handler),
        .priority = registration.priority,
        .required = registration.required,
    });

    // Sort by priority descending (higher runs first)
    std::sort(vec.begin(), vec.end(),
              [](const auto& a, const auto& b) { return a.priority > b.priority; });
}

void HookExecutor::register_batch(const std::string& plugin_name,
                                  const std::vector<HookRegistration>& registrations) {
    for (const auto& reg : registrations) {
        register_hook(plugin_name, reg.event, reg);
    }
}

auto HookExecutor::execute(std::stop_token token, Event& event) -> VoidResult {
    std::vector<RegisteredHook> hooks_copy;
    {
        std::lock_guard lock(mu_);
        auto it = hooks_.find(event.name);
        if (it == hooks_.end() || it->second.empty())
            return {};
        hooks_copy = it->second;
    }

    for (const auto& hook : hooks_copy) {
        if (token.stop_requested()) {
            return make_error(Error::generic("execution cancelled"));
        }

        auto result = hook.handler(token, event);
        if (!result) {
            if (hook.required) {
                return make_error(
                    Error::wrap("[" + hook.plugin_name + ":" + event.name + "]", result.error()));
            }
            // Non-required: continue
        }
    }

    return {};
}

auto HookExecutor::has_hooks(const std::string& event) const -> bool {
    std::lock_guard lock(mu_);
    auto it = hooks_.find(event);
    return it != hooks_.end() && !it->second.empty();
}

auto HookExecutor::get_registered_events() const -> std::vector<std::string> {
    std::lock_guard lock(mu_);
    std::vector<std::string> events;
    events.reserve(hooks_.size());
    for (const auto& [event, _] : hooks_) {
        events.push_back(event);
    }
    std::sort(events.begin(), events.end());
    return events;
}

void HookExecutor::clear() {
    std::lock_guard lock(mu_);
    hooks_.clear();
}

}  // namespace elmos::plugin
