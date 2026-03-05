#pragma once
// ============================================================================
// config/loader.hpp — Configuration loading
// Replaces Go's core/config/loader.go
// ============================================================================

#include "types.hpp"

#include <string>

namespace elmos::config {

/// Load configuration from file and/or environment. Returns Config or error.
/// If config_path is empty, auto-detects workspace config.
auto load(const std::string& config_path = "") -> Result<Config>;

}  // namespace elmos::config
