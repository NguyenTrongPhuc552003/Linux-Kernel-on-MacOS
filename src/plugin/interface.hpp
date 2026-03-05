#pragma once
// ============================================================================
// plugin/interface.hpp — Plugin interface and hook system
// Replaces Go's core/plugin/interface.go
// ============================================================================

#include <elmos/common.hpp>

#include <functional>
#include <stop_token>
#include <string>
#include <vector>

// Forward declare context
namespace elmos::context {
class Context;
}

namespace elmos::plugin {

// Hook event constants — standard build lifecycle events
namespace events {
inline constexpr auto kBeforeInit = "before_init";
inline constexpr auto kAfterInit = "after_init";
inline constexpr auto kAfterConfigLoad = "after_config_load";

inline constexpr auto kPreKernelClone = "pre_kernel_clone";
inline constexpr auto kPostKernelClone = "post_kernel_clone";
inline constexpr auto kPreKernelConfig = "pre_kernel_config";
inline constexpr auto kPostKernelConfig = "post_kernel_config";
inline constexpr auto kPreKernelBuild = "pre_kernel_build";
inline constexpr auto kPostKernelBuild = "post_kernel_build";

inline constexpr auto kPreBootloaderClone = "pre_bootloader_clone";
inline constexpr auto kPostBootloaderClone = "post_bootloader_clone";
inline constexpr auto kPreBootloaderConfig = "pre_bootloader_config";
inline constexpr auto kPostBootloaderConfig = "post_bootloader_config";
inline constexpr auto kPreBootloaderBuild = "pre_bootloader_build";
inline constexpr auto kPostBootloaderBuild = "post_bootloader_build";

inline constexpr auto kPreRootfsCreate = "pre_rootfs_create";
inline constexpr auto kPostRootfsCreate = "post_rootfs_create";

inline constexpr auto kPreImageAssemble = "pre_image_assemble";
inline constexpr auto kPostImageAssemble = "post_image_assemble";

inline constexpr auto kPreArtifactCache = "pre_artifact_cache";
inline constexpr auto kPostArtifactCache = "post_artifact_cache";

inline constexpr auto kPreQEMUBoot = "pre_qemu_boot";
inline constexpr auto kPostQEMUBoot = "post_qemu_boot";

inline constexpr auto kOnBuildError = "on_build_error";
inline constexpr auto kOnCleanup = "on_cleanup";
}  // namespace events

// Common metadata keys used in Event.metadata
namespace metadata {
inline constexpr auto kArtifactPath = "artifact_path";
inline constexpr auto kArtifactPaths = "artifact_paths";
inline constexpr auto kArtifactSize = "artifact_size";
inline constexpr auto kArtifactChecksum = "artifact_checksum";
inline constexpr auto kMachine = "machine";
inline constexpr auto kArchitecture = "architecture";
inline constexpr auto kKernelVersion = "kernel_version";
inline constexpr auto kBootloaderType = "bootloader_type";
inline constexpr auto kBootloaderVer = "bootloader_version";
inline constexpr auto kDuration = "duration_seconds";
inline constexpr auto kCacheHit = "cache_hit";
inline constexpr auto kFingerprint = "fingerprint";
inline constexpr auto kError = "error";
inline constexpr auto kErrorReason = "error_reason";
}  // namespace metadata

/// Event passed to hook handlers during build lifecycle.
struct Event {
    std::string name;
    AnyMap metadata;
};

/// Hook callback signature.
using HookFunc = std::function<VoidResult(std::stop_token, Event&)>;

/// Registration for a single hook.
struct HookRegistration {
    std::string event;
    HookFunc handler;
    int priority = 5;      // 0-10, higher runs first
    bool required = true;  // If false, failure doesn't stop build
};

/// Base interface for all plugins.
class Plugin {
public:
    virtual ~Plugin() = default;

    // Metadata
    virtual auto name() const -> std::string = 0;
    virtual auto version() const -> std::string = 0;
    virtual auto description() const -> std::string = 0;

    // Lifecycle
    virtual auto init(context::Context* ctx, const AnyMap& config) -> VoidResult = 0;
    virtual auto validate() -> VoidResult = 0;
    virtual auto cleanup() -> VoidResult = 0;

    // Hook registrations
    virtual auto hooks() -> std::vector<HookRegistration> = 0;
};

/// TaskRunner interface — plugins that perform actual build work.
class TaskRunner {
public:
    virtual ~TaskRunner() = default;
    virtual auto run_task(std::stop_token token, const std::string& task_id, const AnyMap& config)
        -> VoidResult = 0;
};

/// Factory function type for builtin plugins.
using PluginFactory = std::function<std::unique_ptr<Plugin>()>;

/// Register a builtin plugin factory (called from init modules).
void register_builtin_factory(const std::string& name, PluginFactory factory);

}  // namespace elmos::plugin
