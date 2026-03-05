// ============================================================================
// platform/detect.cpp — Platform factory function
// Replaces Go's core/infra/platform/detect.go
// ============================================================================

#include "detect.hpp"

#include "interface.hpp"

#include <filesystem>

#ifdef ELMOS_PLATFORM_DARWIN
#include "darwin.hpp"
#endif

#ifdef ELMOS_PLATFORM_LINUX
#include "linux.hpp"
#endif

#ifdef ELMOS_PLATFORM_WINDOWS
#include "windows.hpp"
#endif

namespace elmos::infra::platform {

auto create_platform(executor::Executor* exec) -> std::unique_ptr<Platform> {
#ifdef ELMOS_PLATFORM_DARWIN
    return std::make_unique<DarwinPlatform>(exec);
#elif defined(ELMOS_PLATFORM_LINUX)
    auto p = std::make_unique<LinuxPlatform>(exec);
    // Detect OrbStack VM
    if (std::filesystem::exists("/opt/orbstack-guest")) {
        p->set_orbstack(true);
    }
    return p;
#elif defined(ELMOS_PLATFORM_WINDOWS)
    return std::make_unique<WindowsPlatform>(exec);
#else
    // Fallback to Linux
    return std::make_unique<LinuxPlatform>(exec);
#endif
}

}  // namespace elmos::infra::platform
