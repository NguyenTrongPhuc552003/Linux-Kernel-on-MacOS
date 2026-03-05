#pragma once
// ============================================================================
// platform/detect.hpp — Platform factory function
// ============================================================================

#include "interface.hpp"

#include <memory>

namespace elmos::infra::platform {

/// Creates the appropriate Platform implementation for the current OS
auto create_platform(executor::Executor* exec) -> std::unique_ptr<Platform>;

}  // namespace elmos::infra::platform
