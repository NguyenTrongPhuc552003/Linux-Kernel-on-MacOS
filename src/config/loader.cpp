// ============================================================================
// config/loader.cpp — Configuration loading implementation
// ============================================================================

#include "loader.hpp"

#include "arch.hpp"
#include "defaults.hpp"
#include "machine.hpp"

#include <yaml-cpp/yaml.h>

#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <thread>

namespace elmos::config {

namespace fs = std::filesystem;

static auto image_extension() -> std::string {
#ifdef ELMOS_PLATFORM_DARWIN
    return ".sparseimage";
#else
    return ".img";
#endif
}

static void set_if_empty(std::string& target, const std::string& value) {
    if (target.empty())
        target = value;
}

static void apply_project_root(Config& cfg) {
    if (cfg.paths.project_root.empty()) {
        std::error_code ec;
        cfg.paths.project_root = fs::current_path(ec).string();
    }
}

static void apply_image_defaults(Config& cfg) {
    const auto& root = cfg.paths.project_root;
    const auto& name = cfg.image.volume_name;
    if (cfg.image.path.empty()) {
        cfg.image.path = (fs::path(root) / name / (name + image_extension())).string();
    }
    // MountPoint depends on platform — use a reasonable default
    if (cfg.image.mount_point.empty()) {
#ifdef ELMOS_PLATFORM_DARWIN
        cfg.image.mount_point = "/Volumes/" + name;
#elif defined(ELMOS_PLATFORM_WINDOWS)
        cfg.image.mount_point = "\\\\wsl$\\Ubuntu\\mnt\\elmos\\" + name;
#else
        cfg.image.mount_point = "/mnt/" + name;
#endif
    }
}

static void apply_path_defaults(Config& cfg) {
    const auto& root = cfg.paths.project_root;
    const auto& mount = cfg.image.mount_point;

    set_if_empty(cfg.paths.kernel_dir, (fs::path(mount) / "linux").string());
    set_if_empty(cfg.paths.modules_dir, (fs::path(root) / "examples" / "modules").string());
    set_if_empty(cfg.paths.apps_dir, (fs::path(root) / "examples" / "apps").string());
    set_if_empty(cfg.paths.libraries_dir, (fs::path(root) / "assets" / "libraries").string());
    set_if_empty(cfg.paths.patches_dir, (fs::path(root) / "patches").string());
    set_if_empty(cfg.paths.rootfs_dir, (fs::path(mount) / "rootfs").string());
    set_if_empty(cfg.paths.disk_image, (fs::path(mount) / "disk.img").string());
    set_if_empty(cfg.paths.toolchains_dir, (fs::path(mount) / "toolchains").string());
}

static void apply_computed_defaults(Config& cfg) {
    apply_project_root(cfg);
    apply_image_defaults(cfg);
    apply_path_defaults(cfg);
}

static void set_defaults(Config& cfg) {
    if (cfg.image.volume_name.empty())
        cfg.image.volume_name = std::string(kDefaultVolumeName);
    if (cfg.image.size.empty())
        cfg.image.size = std::string(kDefaultImageSize);
    if (cfg.build.arch.empty())
        cfg.build.arch = std::string(kDefaultArch);
    if (cfg.build.jobs == 0)
        cfg.build.jobs = static_cast<int>(std::thread::hardware_concurrency());
    if (cfg.build.cross_compile.empty())
        cfg.build.cross_compile = std::string(kDefaultCrossPrefix);
    if (cfg.qemu.memory.empty())
        cfg.qemu.memory = std::string(kDefaultMemory);
    if (cfg.qemu.gdb_port == 0)
        cfg.qemu.gdb_port = kDefaultGDBPort;
    if (cfg.qemu.ssh_port == 0)
        cfg.qemu.ssh_port = kDefaultSSHPort;
    if (cfg.qemu.smp == 0)
        cfg.qemu.smp = static_cast<int>(std::thread::hardware_concurrency());
    if (cfg.paths.debian_mirror.empty())
        cfg.paths.debian_mirror = std::string(kDefaultDebianMirror);
}

static auto parse_yaml_node(const YAML::Node& node, Config& cfg) -> VoidResult {
    if (!node.IsDefined() || node.IsNull())
        return {};

    // Image
    if (auto img = node["image"]) {
        if (img["path"])
            cfg.image.path = img["path"].as<std::string>("");
        if (img["volume_name"])
            cfg.image.volume_name = img["volume_name"].as<std::string>("");
        if (img["size"])
            cfg.image.size = img["size"].as<std::string>("");
        if (img["mount_point"])
            cfg.image.mount_point = img["mount_point"].as<std::string>("");
    }

    // Build
    if (auto b = node["build"]) {
        if (b["arch"])
            cfg.build.arch = b["arch"].as<std::string>("");
        if (b["jobs"])
            cfg.build.jobs = b["jobs"].as<int>(0);
        if (b["llvm"])
            cfg.build.llvm = b["llvm"].as<bool>(true);
        if (b["cross_compile"])
            cfg.build.cross_compile = b["cross_compile"].as<std::string>("");
        if (b["verbose"])
            cfg.build.verbose = b["verbose"].as<bool>(false);
    }

    // QEMU
    if (auto q = node["qemu"]) {
        if (q["memory"])
            cfg.qemu.memory = q["memory"].as<std::string>("");
        if (q["gdb_port"])
            cfg.qemu.gdb_port = q["gdb_port"].as<int>(0);
        if (q["ssh_port"])
            cfg.qemu.ssh_port = q["ssh_port"].as<int>(0);
        if (q["smp"])
            cfg.qemu.smp = q["smp"].as<int>(0);
    }

    // Paths
    if (auto p = node["paths"]) {
        if (p["project_root"])
            cfg.paths.project_root = p["project_root"].as<std::string>("");
        if (p["kernel_dir"])
            cfg.paths.kernel_dir = p["kernel_dir"].as<std::string>("");
        if (p["modules_dir"])
            cfg.paths.modules_dir = p["modules_dir"].as<std::string>("");
        if (p["apps_dir"])
            cfg.paths.apps_dir = p["apps_dir"].as<std::string>("");
        if (p["libraries_dir"])
            cfg.paths.libraries_dir = p["libraries_dir"].as<std::string>("");
        if (p["patches_dir"])
            cfg.paths.patches_dir = p["patches_dir"].as<std::string>("");
        if (p["rootfs_dir"])
            cfg.paths.rootfs_dir = p["rootfs_dir"].as<std::string>("");
        if (p["disk_image"])
            cfg.paths.disk_image = p["disk_image"].as<std::string>("");
        if (p["debian_mirror"])
            cfg.paths.debian_mirror = p["debian_mirror"].as<std::string>("");
        if (p["toolchains_dir"])
            cfg.paths.toolchains_dir = p["toolchains_dir"].as<std::string>("");
    }

    // Machine
    if (node["machine"])
        cfg.machine = node["machine"].as<std::string>("");

    return {};
}

auto load(const std::string& config_path) -> Result<Config> {
    Config cfg;

    // Determine which config file to load
    std::string file_to_load = config_path;

    if (file_to_load.empty()) {
        // Auto-detect workspace config
        std::error_code ec;
        auto cwd = fs::current_path(ec);
        if (!ec) {
            auto dir_name = cwd.filename().string();
            auto candidate = cwd / (dir_name + ".yaml");
            if (fs::exists(candidate)) {
                file_to_load = candidate.string();
            }
        }

        if (file_to_load.empty()) {
            // Search for elmos.yaml in standard locations
            auto cwd_path = fs::current_path(ec);
            for (const auto& name : {"elmos.yaml", "elmos.yml"}) {
                auto candidate = cwd_path / name;
                if (fs::exists(candidate)) {
                    file_to_load = candidate.string();
                    break;
                }
            }
        }

        if (file_to_load.empty()) {
            const char* home = std::getenv("HOME");
            if (home) {
                auto candidate = fs::path(home) / ".config" / "elmos" / "elmos.yaml";
                if (fs::exists(candidate)) {
                    file_to_load = candidate.string();
                }
            }
        }
    }

    // Set defaults first
    set_defaults(cfg);

    // Load and parse YAML if file found
    if (!file_to_load.empty()) {
        if (!config_path.empty() && !fs::exists(file_to_load)) {
            return make_error(Error::config("config file not found: " + file_to_load));
        }

        if (fs::exists(file_to_load)) {
            try {
                auto node = YAML::LoadFile(file_to_load);
                auto r = parse_yaml_node(node, cfg);
                if (!r)
                    return make_error(r.error());
                cfg.config_file = file_to_load;
            }
            catch (const YAML::Exception& e) {
                return make_error(
                    Error::config("failed to parse config: " + std::string(e.what())));
            }
        }
    }

    // Apply environment variable overrides
    if (const char* arch = std::getenv("ELMOS_BUILD_ARCH")) {
        cfg.build.arch = arch;
    }
    if (const char* verbose = std::getenv("ELMOS_BUILD_VERBOSE")) {
        cfg.build.verbose = (std::string(verbose) == "true" || std::string(verbose) == "1");
    }

    // Apply computed defaults (paths based on project root)
    apply_computed_defaults(cfg);

    return cfg;
}

}  // namespace elmos::config
