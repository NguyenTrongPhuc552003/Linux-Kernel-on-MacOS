#pragma once
// ============================================================================
// executor/env.hpp — Environment variable merging utilities
// Replaces Go's core/infra/executor/env.go
// ============================================================================

#include <elmos/common.hpp>

#include <string>
#include <unordered_map>
#include <vector>

namespace elmos::infra::executor {

/// Merge two environment variable lists. Override entries take precedence.
/// Format: "KEY=VALUE"
auto merge_env(const EnvList& base, const EnvList& overrides) -> EnvList;

/// Get the current process environment as a map.
auto get_current_env() -> std::unordered_map<std::string, std::string>;

/// Get the current process environment as a list of "KEY=VALUE" strings.
auto get_current_env_list() -> EnvList;

/// Split "KEY=VALUE" into (key, value). Returns ("","") for invalid entries.
auto split_env(const std::string& entry) -> std::pair<std::string, std::string>;

}  // namespace elmos::infra::executor
