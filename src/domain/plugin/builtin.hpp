#pragma once
// ============================================================================
// domain/plugin/builtin/kernel_builder_plugin.hpp
// ============================================================================

#include <plugin/interface.hpp>

namespace elmos::domain::plugin::builtin {

class KernelBuilderPlugin : public elmos::plugin::Plugin {
public:
    auto name() const -> std::string override { return "kernel-builder"; }
    auto version() const -> std::string override { return "1.0.0"; }
    auto description() const -> std::string override { return "Kernel build orchestration"; }

    auto init(context::Context* ctx, const AnyMap& config) -> VoidResult override;
    auto validate() -> VoidResult override { return {}; }
    auto cleanup() -> VoidResult override { return {}; }
    auto hooks() -> std::vector<elmos::plugin::HookRegistration> override;

private:
    context::Context* ctx_ = nullptr;
};

class BspManagerPlugin : public elmos::plugin::Plugin {
public:
    auto name() const -> std::string override { return "bsp-manager"; }
    auto version() const -> std::string override { return "1.0.0"; }
    auto description() const -> std::string override { return "Board support package management"; }

    auto init(context::Context* ctx, const AnyMap& config) -> VoidResult override;
    auto validate() -> VoidResult override { return {}; }
    auto cleanup() -> VoidResult override { return {}; }
    auto hooks() -> std::vector<elmos::plugin::HookRegistration> override { return {}; }

private:
    context::Context* ctx_ = nullptr;
};

class BootloaderBuilderPlugin : public elmos::plugin::Plugin {
public:
    auto name() const -> std::string override { return "bootloader-builder"; }
    auto version() const -> std::string override { return "1.0.0"; }
    auto description() const -> std::string override { return "Bootloader build orchestration"; }

    auto init(context::Context* ctx, const AnyMap& config) -> VoidResult override;
    auto validate() -> VoidResult override { return {}; }
    auto cleanup() -> VoidResult override { return {}; }
    auto hooks() -> std::vector<elmos::plugin::HookRegistration> override { return {}; }

private:
    context::Context* ctx_ = nullptr;
};

class CachingBackendPlugin : public elmos::plugin::Plugin {
public:
    auto name() const -> std::string override { return "caching-backend"; }
    auto version() const -> std::string override { return "1.0.0"; }
    auto description() const -> std::string override { return "Build artifact caching"; }

    auto init(context::Context* ctx, const AnyMap& config) -> VoidResult override;
    auto validate() -> VoidResult override { return {}; }
    auto cleanup() -> VoidResult override { return {}; }
    auto hooks() -> std::vector<elmos::plugin::HookRegistration> override { return {}; }

private:
    context::Context* ctx_ = nullptr;
};

class RootfsBuilderPlugin : public elmos::plugin::Plugin {
public:
    auto name() const -> std::string override { return "rootfs-builder"; }
    auto version() const -> std::string override { return "1.0.0"; }
    auto description() const -> std::string override { return "Root filesystem builder"; }

    auto init(context::Context* ctx, const AnyMap& config) -> VoidResult override;
    auto validate() -> VoidResult override { return {}; }
    auto cleanup() -> VoidResult override { return {}; }
    auto hooks() -> std::vector<elmos::plugin::HookRegistration> override { return {}; }

private:
    context::Context* ctx_ = nullptr;
};

class ImageAssemblerPlugin : public elmos::plugin::Plugin {
public:
    auto name() const -> std::string override { return "image-assembler"; }
    auto version() const -> std::string override { return "1.0.0"; }
    auto description() const -> std::string override { return "Final image assembly"; }

    auto init(context::Context* ctx, const AnyMap& config) -> VoidResult override;
    auto validate() -> VoidResult override { return {}; }
    auto cleanup() -> VoidResult override { return {}; }
    auto hooks() -> std::vector<elmos::plugin::HookRegistration> override { return {}; }

private:
    context::Context* ctx_ = nullptr;
};

/// Register all builtin plugin factories.
void register_all();

}  // namespace elmos::domain::plugin::builtin
