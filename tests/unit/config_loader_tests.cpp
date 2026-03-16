// ============================================================================
// tests/unit/config_loader_tests.cpp — Configuration loading unit tests
// ============================================================================

#include <config/arch.hpp>
#include <config/defaults.hpp>
#include <config/loader.hpp>
#include <config/types.hpp>

#include <catch2/catch_test_macros.hpp>
#include <catch2/matchers/catch_matchers_string.hpp>

#include <filesystem>
#include <fstream>
#include <sstream>

namespace fs = std::filesystem;
using namespace Catch::Matchers;
using namespace elmos::config;

// ── Default values applied on empty config ─────────────────────────────────

TEST_CASE("Config defaults applied when no file found", "[config][defaults]") {
    // Isolate from any active workspace by pointing HOME at a temp directory
    auto tmp_home = fs::temp_directory_path() / "elmos_test_home";
    fs::create_directories(tmp_home);
    std::string old_home = std::getenv("HOME") ? std::getenv("HOME") : "";
    setenv("HOME", tmp_home.c_str(), 1);

    auto result = load("");  // empty path → auto-detect returns default config
    REQUIRE(result.has_value());
    const auto& cfg = *result;

    CHECK(cfg.build.arch == std::string(kDefaultArch));
    CHECK(cfg.build.jobs > 0);
    CHECK(cfg.qemu.memory == std::string(kDefaultMemory));
    CHECK(cfg.qemu.gdb_port == kDefaultGDBPort);
    CHECK(cfg.qemu.ssh_port == kDefaultSSHPort);
    CHECK(cfg.image.volume_name == std::string(kDefaultVolumeName));
    CHECK(cfg.image.size == std::string(kDefaultImageSize));

    // Restore HOME
    setenv("HOME", old_home.c_str(), 1);
    fs::remove_all(tmp_home);
}

// ── YAML round-trip ─────────────────────────────────────────────────────────

TEST_CASE("Config YAML file: round-trip write→read", "[config][yaml]") {
    auto tmp = fs::temp_directory_path() / "elmos_test_config.yaml";

    // Write a minimal YAML to a temp file
    {
        std::ofstream f(tmp);
        REQUIRE(f.is_open());
        f << "build:\n"
             "  arch: riscv\n"
             "  jobs: 4\n"
             "  llvm: false\n"
             "  verbose: true\n"
             "qemu:\n"
             "  memory: 4G\n"
             "  gdb_port: 5678\n"
             "  ssh_port: 3333\n"
             "  smp: 2\n"
             "image:\n"
             "  volume_name: mytest\n"
             "  size: 20G\n";
    }

    auto result = load(tmp.string());
    fs::remove(tmp);

    REQUIRE(result.has_value());
    const auto& cfg = *result;

    CHECK(cfg.build.arch == "riscv");
    CHECK(cfg.build.jobs == 4);
    CHECK(cfg.build.llvm == false);
    CHECK(cfg.build.verbose == true);
    CHECK(cfg.qemu.memory == "4G");
    CHECK(cfg.qemu.gdb_port == 5678);
    CHECK(cfg.qemu.ssh_port == 3333);
    CHECK(cfg.qemu.smp == 2);
    CHECK(cfg.image.volume_name == "mytest");
    CHECK(cfg.image.size == "20G");
}

// ── Missing file returns error only when path is explicit ──────────────────

TEST_CASE("Config: explicit missing file returns error", "[config][errors]") {
    auto result = load("/nonexistent/elmos_totally_missing.yaml");
    REQUIRE_FALSE(result.has_value());
    CHECK_THAT(result.error().message(), ContainsSubstring("not found"));
}

// ── Malformed YAML returns error ────────────────────────────────────────────

TEST_CASE("Config: malformed YAML returns parse error", "[config][errors]") {
    auto tmp = fs::temp_directory_path() / "elmos_bad_yaml.yaml";
    {
        std::ofstream f(tmp);
        REQUIRE(f.is_open());
        f << "build:\n"
             "  arch: [unclosed bracket\n";
    }

    auto result = load(tmp.string());
    fs::remove(tmp);

    REQUIRE_FALSE(result.has_value());
    CHECK_THAT(result.error().message(), ContainsSubstring("parse"));
}

// ── Default kDefaultArch is a valid registered architecture ────────────────

TEST_CASE("Default arch is registered in arch registry", "[config][arch]") {
    CHECK(is_valid_arch(std::string(kDefaultArch)));
}

// ── Computed defaults: paths derive from project_root ──────────────────────

TEST_CASE("Config: paths derived from project_root", "[config][paths]") {
    auto tmp = fs::temp_directory_path() / "elmos_paths_test.yaml";
    {
        std::ofstream f(tmp);
        REQUIRE(f.is_open());
        f << "paths:\n"
             "  project_root: /tmp/my_workspace\n"
             "image:\n"
             "  volume_name: myvol\n"
             "  mount_point: /Volumes/myvol\n";
    }

    auto result = load(tmp.string());
    fs::remove(tmp);

    REQUIRE(result.has_value());
    const auto& cfg = *result;

    // patches_dir should default to project_root/patches if not set
    CHECK_THAT(cfg.paths.patches_dir, ContainsSubstring("patches"));
    CHECK_THAT(cfg.paths.patches_dir, ContainsSubstring("my_workspace"));

    // modules_dir should default to project_root/examples/modules if not set
    CHECK_THAT(cfg.paths.modules_dir, ContainsSubstring("modules"));
}
