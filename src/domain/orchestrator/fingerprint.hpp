#pragma once
// ============================================================================
// domain/orchestrator/fingerprint.hpp — Build artifact fingerprinting
// ============================================================================

#include <elmos/common.hpp>

#include <string>
#include <vector>

namespace elmos::domain::orchestrator {

/// Computes a content-based fingerprint for cache invalidation.
class Fingerprinter {
public:
    auto compute(const std::vector<std::string>& paths) -> Result<std::string>;
    auto compute_string(const std::string& content) -> std::string;
};

}  // namespace elmos::domain::orchestrator
