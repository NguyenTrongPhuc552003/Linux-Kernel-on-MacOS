#pragma once
// ============================================================================
// elmos/forward.hpp — Forward declarations for all major types
// Avoids circular include dependencies between layers.
// ============================================================================

namespace elmos {

class Error;

namespace config {
struct Config;
struct ImageConfig;
struct BuildConfig;
struct QEMUConfig;
struct PathsConfig;
struct ArchConfig;
struct MachineDefinition;
struct BootloaderConfig;
struct PluginsConfig;
class Loader;
class WorkspaceManager;
}  // namespace config

namespace context {
class Context;
}

namespace executor {
class Executor;
class ShellExecutor;
class MockExecutor;
}  // namespace executor

namespace filesystem {
class FileSystem;
class OSFileSystem;
}  // namespace filesystem

namespace platform {
class Platform;
class DiskImageManager;
class PackageManager;
class PathProvider;
}  // namespace platform

namespace homebrew {
class PathResolver;
class Resolver;
}  // namespace homebrew

namespace plugin {
class Plugin;
class HookExecutor;
class Registry;
struct HookRegistration;
struct Event;
}  // namespace plugin

namespace builder {
class KernelBuilder;
class ModuleBuilder;
class AppBuilder;
}  // namespace builder

namespace bsp {
class RegistryClient;
class RegistryManager;
class FirmwareMgr;
}  // namespace bsp

namespace doctor {
class HealthChecker;
class AutoFixer;
}  // namespace doctor

namespace emulator {
class QEMURunner;
}

namespace orchestrator {
class Pipeline;
class PipelineExecutor;
class BuildFingerprinter;
}  // namespace orchestrator

namespace patch {
class Manager;
}

namespace rootfs {
class Creator;
}

namespace toolchain {
class Manager;
class Builder;
}  // namespace toolchain

namespace ui {
class Printer;
}

}  // namespace elmos
