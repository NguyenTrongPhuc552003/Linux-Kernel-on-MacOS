// ============================================================================
// tests/unit/qemu_runner_tests.cpp — QEMU command assembly tests
// ============================================================================

#include <config/types.hpp>
#include <context/context.hpp>
#include <domain/emulator/qemu.hpp>
#include <infra/executor/mock.hpp>
#include <infra/filesystem/os_filesystem.hpp>
#include <infra/platform/interface.hpp>

#include <catch2/catch_test_macros.hpp>

#include <chrono>
#include <filesystem>
#include <fstream>

namespace fs = std::filesystem;

namespace {

class TestDiskImageManager final : public elmos::infra::platform::DiskImageManager {
public:
    auto create(std::stop_token, const std::string&, int) -> elmos::VoidResult override {
        return {};
    }

    auto mount(std::stop_token, const std::string&) -> elmos::Result<std::string> override {
        return std::string{};
    }

    auto unmount(std::stop_token, const std::string&) -> elmos::VoidResult override { return {}; }

    auto is_mounted(std::stop_token, const std::string&) -> elmos::Result<MountStatus> override {
        return MountStatus{false, ""};
    }
};

class TestPackages final : public elmos::infra::platform::PackageManager {
public:
    auto is_installed(const std::string&) -> bool override { return true; }
    auto install(std::stop_token, const std::string&) -> elmos::VoidResult override { return {}; }
    auto list_installed() -> elmos::Result<std::vector<std::string>> override {
        return std::vector<std::string>{};
    }
    auto get_bin_path(const std::string&) -> std::string override { return ""; }
    auto get_lib_path(const std::string&) -> std::string override { return ""; }
    auto get_include_path(const std::string&) -> std::string override { return ""; }
    auto build_gnu_environment() -> elmos::EnvList override { return {}; }
};

class TestPaths final : public elmos::infra::platform::PathProvider {
public:
    auto workspace_root(const std::string& name) -> std::string override { return name; }
    auto cache_dir() -> std::string override { return "/tmp"; }
    auto toolchain_dir() -> std::string override { return "/tmp"; }
};

class TestPlatform final : public elmos::infra::platform::Platform {
public:
    auto name() const -> std::string override { return "test"; }
    auto disk_image() -> elmos::infra::platform::DiskImageManager& override { return disk_image_; }
    auto packages() -> elmos::infra::platform::PackageManager& override { return packages_; }
    auto paths() -> elmos::infra::platform::PathProvider& override { return paths_; }
    void set_executor(elmos::infra::executor::Executor*) override {}

private:
    TestDiskImageManager disk_image_;
    TestPackages packages_;
    TestPaths paths_;
};

auto make_temp_dir() -> fs::path {
    auto stamp = std::chrono::steady_clock::now().time_since_epoch().count();
    auto path = fs::temp_directory_path() / ("elmos-qemu-test-" + std::to_string(stamp));
    fs::create_directories(path);
    return path;
}

void write_file(const fs::path& path, std::string_view content) {
    fs::create_directories(path.parent_path());
    std::ofstream out(path, std::ios::binary | std::ios::trunc);
    out << content;
}

auto contains_arg(const std::vector<std::string>& args, const std::string& needle) -> bool {
    return std::find(args.begin(), args.end(), needle) != args.end();
}

}  // namespace

TEST_CASE("QEMU runner builds full command with all configured arguments", "[qemu][argv]") {
    const auto temp_dir = make_temp_dir();
    const auto kernel_dir = temp_dir / "kernel";
    const auto disk_image = temp_dir / "disk.img";
    const auto modules_dir = temp_dir / "modules";
    const auto apps_dir = temp_dir / "apps";

    write_file(kernel_dir / "arch" / "riscv" / "boot" / "Image", "kernel");
    write_file(disk_image, "disk");
    fs::create_directories(modules_dir);
    fs::create_directories(apps_dir);

    elmos::config::Config cfg;
    cfg.build.arch = "riscv";
    cfg.qemu.memory = "2G";
    cfg.qemu.smp = 4;
    cfg.qemu.gdb_port = 1234;
    cfg.qemu.ssh_port = 2222;
    cfg.paths.kernel_dir = kernel_dir.string();
    cfg.paths.disk_image = disk_image.string();
    cfg.paths.modules_dir = modules_dir.string();
    cfg.paths.apps_dir = apps_dir.string();

    auto exec = std::make_unique<elmos::infra::executor::MockExecutor>();
    exec->look_path_responses["qemu-system-riscv64"] = "/usr/bin/qemu-system-riscv64";

    auto filesystem = std::make_unique<elmos::infra::filesystem::OSFileSystem>();
    elmos::context::Context ctx(&cfg, exec.get(), filesystem.get());
    ctx.set_platform(std::make_unique<TestPlatform>());

    elmos::domain::emulator::QEMURunner runner(&ctx);
    elmos::domain::emulator::RunOptions opts;
    opts.gdb = true;
    opts.graphic = false;
    opts.append = "panic=1 loglevel=8";
    opts.extra_args = {"-d", "guest_errors"};

    const auto result = runner.build_command(opts);

    REQUIRE(result);
    const auto& [binary, args] = *result;
    CHECK(binary == "/usr/bin/qemu-system-riscv64");
    CHECK(contains_arg(args, "-machine"));
    CHECK(contains_arg(args, "virt"));
    CHECK(contains_arg(args, "-nographic"));
    CHECK(contains_arg(args, "-serial"));
    CHECK(contains_arg(args, "mon:stdio"));
    CHECK(contains_arg(args, "-gdb"));
    CHECK(contains_arg(args, "tcp::1234"));
    CHECK(contains_arg(args, "-S"));
    CHECK(contains_arg(args, "-d"));
    CHECK(contains_arg(args, "guest_errors"));

    const auto append_it = std::find(args.begin(), args.end(), "-append");
    REQUIRE(append_it != args.end());
    REQUIRE((append_it + 1) != args.end());
    CHECK((append_it + 1)->find("root=/dev/vda") != std::string::npos);
    CHECK((append_it + 1)->find("init=/init") == std::string::npos);
    CHECK((append_it + 1)->find("console=ttyS0") != std::string::npos);
    CHECK((append_it + 1)->find("panic=1 loglevel=8") != std::string::npos);

    fs::remove_all(temp_dir);
}

TEST_CASE("QEMU runner supports initrd mode without disk image", "[qemu][argv]") {
    const auto temp_dir = make_temp_dir();
    const auto kernel_dir = temp_dir / "kernel";
    const auto initrd = temp_dir / "initrd.img";

    write_file(kernel_dir / "arch" / "riscv" / "boot" / "Image", "kernel");
    write_file(initrd, "initrd");

    elmos::config::Config cfg;
    cfg.build.arch = "riscv";
    cfg.qemu.memory = "1G";
    cfg.qemu.smp = 1;
    cfg.qemu.gdb_port = 1234;
    cfg.qemu.ssh_port = 2222;
    cfg.paths.kernel_dir = kernel_dir.string();
    cfg.paths.disk_image = (temp_dir / "missing-disk.img").string();

    auto exec = std::make_unique<elmos::infra::executor::MockExecutor>();
    exec->look_path_responses["qemu-system-riscv64"] = "/usr/bin/qemu-system-riscv64";

    auto filesystem = std::make_unique<elmos::infra::filesystem::OSFileSystem>();
    elmos::context::Context ctx(&cfg, exec.get(), filesystem.get());
    ctx.set_platform(std::make_unique<TestPlatform>());

    elmos::domain::emulator::QEMURunner runner(&ctx);
    elmos::domain::emulator::RunOptions opts;
    opts.initrd = initrd.string();

    const auto result = runner.build_command(opts);

    REQUIRE(result);
    const auto& [binary, args] = *result;
    CHECK(binary == "/usr/bin/qemu-system-riscv64");
    CHECK(contains_arg(args, "-initrd"));
    CHECK(contains_arg(args, initrd.string()));

    const bool has_drive = contains_arg(args, "-drive");
    CHECK_FALSE(has_drive);

    fs::remove_all(temp_dir);
}
