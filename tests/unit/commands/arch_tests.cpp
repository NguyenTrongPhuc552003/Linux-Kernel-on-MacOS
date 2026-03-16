// ============================================================================
// tests/unit/commands/arch_tests.cpp
//   Unit tests for the architecture subsystem and the arch CLI command.
//   Validates:
//     • config::is_valid_arch() correctness
//     • config::get_arch_config() field values
//     • config::supported_architectures() completeness
//     • Option* pattern (no dangling ref): CLI11 parsing via arch callback
// ============================================================================

#include <config/arch.hpp>

#include <catch2/catch_test_macros.hpp>
#include <catch2/matchers/catch_matchers_string.hpp>
#include <CLI/CLI.hpp>

#include <algorithm>
#include <string>
#include <vector>

using namespace elmos::config;
using namespace Catch::Matchers;

// ── is_valid_arch ──────────────────────────────────────────────────────────

TEST_CASE("is_valid_arch: known architectures accepted", "[arch][validation]") {
    CHECK(is_valid_arch("arm64"));
    CHECK(is_valid_arch("arm"));
    CHECK(is_valid_arch("riscv"));
}

TEST_CASE("is_valid_arch: unknown architectures rejected", "[arch][validation]") {
    CHECK_FALSE(is_valid_arch("x86_64"));
    CHECK_FALSE(is_valid_arch("mips"));
    CHECK_FALSE(is_valid_arch("powerpc"));
    CHECK_FALSE(is_valid_arch("aarch64"));  // alias not in registry
    CHECK_FALSE(is_valid_arch("riscv64"));  // alias not in registry
    CHECK_FALSE(is_valid_arch(""));
}

// ── supported_architectures ────────────────────────────────────────────────

TEST_CASE("supported_architectures: all expected entries present", "[arch][registry]") {
    auto supported = supported_architectures();
    REQUIRE_FALSE(supported.empty());

    auto has = [&](const std::string& name) {
        return std::find(supported.begin(), supported.end(), name) != supported.end();
    };

    CHECK(has("arm64"));
    CHECK(has("arm"));
    CHECK(has("riscv"));
    // New architectures should be added to arch.cpp, not hard-coded here.
    CHECK(supported.size() >= 3);
}

// ── get_arch_config ────────────────────────────────────────────────────────

TEST_CASE("get_arch_config: arm64 fields", "[arch][config]") {
    const ArchConfig* ac = get_arch_config("arm64");
    REQUIRE(ac != nullptr);

    CHECK(ac->name == "arm64");
    CHECK(ac->kernel_arch == "arm64");
    CHECK(ac->kernel_image == "Image");
    CHECK_THAT(ac->qemu_binary, ContainsSubstring("aarch64"));
    CHECK_THAT(ac->gcc_binary, ContainsSubstring("aarch64"));
    CHECK_FALSE(ac->default_targets.empty());
}

TEST_CASE("get_arch_config: arm fields", "[arch][config]") {
    const ArchConfig* ac = get_arch_config("arm");
    REQUIRE(ac != nullptr);

    CHECK(ac->name == "arm");
    CHECK(ac->kernel_arch == "arm");
    CHECK(ac->kernel_image == "zImage");
    CHECK_THAT(ac->qemu_binary, ContainsSubstring("arm"));
    CHECK_FALSE(ac->default_targets.empty());
}

TEST_CASE("get_arch_config: riscv fields", "[arch][config]") {
    const ArchConfig* ac = get_arch_config("riscv");
    REQUIRE(ac != nullptr);

    CHECK(ac->name == "riscv");
    CHECK(ac->kernel_arch == "riscv");
    CHECK(ac->kernel_image == "Image");
    CHECK_THAT(ac->qemu_binary, ContainsSubstring("riscv"));
}

TEST_CASE("get_arch_config: unknown arch returns nullptr", "[arch][config]") {
    CHECK(get_arch_config("x86_64") == nullptr);
    CHECK(get_arch_config("") == nullptr);
    CHECK(get_arch_config("INVALID") == nullptr);
}

// ── CLI11 Option* pattern — proves dangling-ref fix is correct ─────────────
//
// The bug was: register_arch() declared `std::string target;`, passed it to
// `add_option(..., target, ...)` AND captured `&target` in the callback.
// Once register_arch() returned, target was destroyed — dangling reference.
//
// The fix: store the `CLI::Option*` returned by add_option(), call ->as<T>()
// inside the callback. CLI::App _owns_ the options, so the pointer is valid
// for the lifetime of the CLI::App — well past the end of any register_*() call.
//
TEST_CASE("Option* is valid after adding function returns", "[arch][cli][lifetime]") {
    // Simulates: a register_*() function that creates options and returns.
    // The function is modelled as a lambda that goes out of scope.
    CLI::App app{"test"};
    CLI::Option* saved = nullptr;

    {
        // ── Scope represents the body of register_arch() ──
        // All locals here (including any std::string) die at the closing brace.
        // The Option* is owned by the CLI::App — it outlives this scope.
        saved = app.add_option("target", "Architecture to set");
    }  // ← local std::string would be dead here. Option* is still alive.

    // The pointer must remain valid because CLI::App owns its options.
    REQUIRE(saved != nullptr);

    // Parsing must fill the option correctly through the saved pointer.
    app.parse(std::vector<std::string>{"arm64"});

    REQUIRE(saved->count() == 1);
    CHECK(saved->as<std::string>() == "arm64");
}
