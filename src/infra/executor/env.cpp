// ============================================================================
// executor/env.cpp — Environment variable merging implementation
// ============================================================================

#include "env.hpp"

#include <cstdlib>

extern "C" {
extern char** environ;
}

namespace elmos::infra::executor {

auto split_env(const std::string& entry) -> std::pair<std::string, std::string> {
    auto pos = entry.find('=');
    if (pos == std::string::npos || pos == 0) {
        return {"", ""};
    }
    return {entry.substr(0, pos), entry.substr(pos + 1)};
}

auto merge_env(const EnvList& base, const EnvList& overrides) -> EnvList {
    std::unordered_map<std::string, std::string> env_map;

    for (const auto& entry : base) {
        auto [key, value] = split_env(entry);
        if (!key.empty()) {
            env_map[key] = value;
        }
    }

    for (const auto& entry : overrides) {
        auto [key, value] = split_env(entry);
        if (!key.empty()) {
            env_map[key] = value;
        }
    }

    EnvList result;
    result.reserve(env_map.size());
    for (const auto& [key, value] : env_map) {
        result.push_back(key + "=" + value);
    }
    return result;
}

auto get_current_env() -> std::unordered_map<std::string, std::string> {
    std::unordered_map<std::string, std::string> env;
    for (char** e = environ; *e != nullptr; ++e) {
        auto [key, value] = split_env(*e);
        if (!key.empty()) {
            env[key] = value;
        }
    }
    return env;
}

auto get_current_env_list() -> EnvList {
    EnvList result;
    for (char** e = environ; *e != nullptr; ++e) {
        result.emplace_back(*e);
    }
    return result;
}

}  // namespace elmos::infra::executor
