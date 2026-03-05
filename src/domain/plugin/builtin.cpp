// ============================================================================
// domain/plugin/builtin.cpp — Builtin plugin implementations + factory registration
// ============================================================================

#include "builtin.hpp"

#include <plugin/loader.hpp>

namespace elmos::domain::plugin::builtin {

// --- Plugin init implementations ---

auto KernelBuilderPlugin::init(context::Context* ctx, const AnyMap&) -> VoidResult {
    ctx_ = ctx;
    return {};
}

auto KernelBuilderPlugin::hooks() -> std::vector<elmos::plugin::HookRegistration> {
    return {};  // Hooks added in Phase 3
}

auto BspManagerPlugin::init(context::Context* ctx, const AnyMap&) -> VoidResult {
    ctx_ = ctx;
    return {};
}

auto BootloaderBuilderPlugin::init(context::Context* ctx, const AnyMap&) -> VoidResult {
    ctx_ = ctx;
    return {};
}

auto CachingBackendPlugin::init(context::Context* ctx, const AnyMap&) -> VoidResult {
    ctx_ = ctx;
    return {};
}

auto RootfsBuilderPlugin::init(context::Context* ctx, const AnyMap&) -> VoidResult {
    ctx_ = ctx;
    return {};
}

auto ImageAssemblerPlugin::init(context::Context* ctx, const AnyMap&) -> VoidResult {
    ctx_ = ctx;
    return {};
}

// --- Factory registration ---

void register_all() {
    elmos::plugin::register_builtin_factory("kernel-builder",
                                            [] { return std::make_unique<KernelBuilderPlugin>(); });
    elmos::plugin::register_builtin_factory("bsp-manager",
                                            [] { return std::make_unique<BspManagerPlugin>(); });
    elmos::plugin::register_builtin_factory(
        "bootloader-builder", [] { return std::make_unique<BootloaderBuilderPlugin>(); });
    elmos::plugin::register_builtin_factory(
        "caching-backend", [] { return std::make_unique<CachingBackendPlugin>(); });
    elmos::plugin::register_builtin_factory("rootfs-builder",
                                            [] { return std::make_unique<RootfsBuilderPlugin>(); });
    elmos::plugin::register_builtin_factory(
        "image-assembler", [] { return std::make_unique<ImageAssemblerPlugin>(); });
}

}  // namespace elmos::domain::plugin::builtin
