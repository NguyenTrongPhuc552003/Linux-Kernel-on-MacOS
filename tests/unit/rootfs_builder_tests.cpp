// ============================================================================
// tests/unit/rootfs_builder_tests.cpp — Rootfs disk image assembly tests
// ============================================================================

#include <config/defaults.hpp>
#include <config/types.hpp>
#include <context/context.hpp>
#include <domain/rootfs/builder.hpp>
#include <infra/executor/mock.hpp>
#include <infra/filesystem/os_filesystem.hpp>
#include <infra/platform/interface.hpp>

#include <catch2/catch_test_macros.hpp>

#include <algorithm>
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
    auto path = fs::temp_directory_path() / ("elmos-rootfs-test-" + std::to_string(stamp));
    fs::create_directories(path);
    return path;
}

void write_file(const fs::path& path, std::string_view content) {
    fs::create_directories(path.parent_path());
    std::ofstream out(path, std::ios::binary | std::ios::trunc);
    out << content;
}

auto has_command_call(const elmos::infra::executor::MockExecutor& exec, const std::string& cmd)
    -> bool {
    return std::any_of(exec.calls.begin(), exec.calls.end(),
                       [&](const auto& call) { return call.cmd == cmd; });
}

}  // namespace

TEST_CASE("Rootfs builder assembles a missing disk image", "[rootfs][disk-image]") {
    auto temp_dir = make_temp_dir();
    auto rootfs_dir = temp_dir / "rootfs";
    auto disk_image = temp_dir / "disk.img";

    write_file(rootfs_dir / "etc" / "os-release", "NAME=ELMOS\n");

    elmos::config::Config cfg;
    cfg.image.volume_name = "elmos-test";
    cfg.image.size = "64M";
    cfg.paths.rootfs_dir = rootfs_dir.string();
    cfg.paths.disk_image = disk_image.string();

    auto exec = std::make_unique<elmos::infra::executor::MockExecutor>();
    auto filesystem = std::make_unique<elmos::infra::filesystem::OSFileSystem>();
    elmos::context::Context ctx(&cfg, exec.get(), filesystem.get());
    ctx.set_platform(std::make_unique<TestPlatform>());
    elmos::domain::rootfs::Builder builder(&ctx);

    auto result = builder.ensure_disk_image({}, "64M");

    REQUIRE(result);
    CHECK(*result);
    CHECK(has_command_call(*exec, "truncate"));
    const bool has_ext4_mkfs = has_command_call(*exec, "mke2fs") ||
                               has_command_call(*exec, "mkfs.ext4");
    CHECK(has_ext4_mkfs);

    fs::remove_all(temp_dir);
}

TEST_CASE("Rootfs builder skips rebuild when disk image is current", "[rootfs][disk-image]") {
    auto temp_dir = make_temp_dir();
    auto rootfs_dir = temp_dir / "rootfs";
    auto disk_image = temp_dir / "disk.img";

    fs::create_directories(rootfs_dir / "etc");
    write_file(rootfs_dir / "etc" / "os-release", "NAME=ELMOS\n");

    elmos::config::Config cfg;
    cfg.image.volume_name = "elmos-test";
    cfg.image.size = "64M";
    cfg.paths.rootfs_dir = rootfs_dir.string();
    cfg.paths.disk_image = disk_image.string();

    auto exec = std::make_unique<elmos::infra::executor::MockExecutor>();
    auto filesystem = std::make_unique<elmos::infra::filesystem::OSFileSystem>();
    elmos::context::Context ctx(&cfg, exec.get(), filesystem.get());
    ctx.set_platform(std::make_unique<TestPlatform>());
    elmos::domain::rootfs::Builder builder(&ctx);

    // First call creates the image and writes guest init.
    auto first = builder.ensure_disk_image({}, "64M");
    REQUIRE(first);
    REQUIRE(*first);

    // MockExecutor doesn't create files; emulate produced image and make it newer
    // than rootfs so incremental check can skip rebuilding.
    write_file(disk_image, std::string(4096, '\0'));
    const auto now = fs::file_time_type::clock::now();

    // Builder scans latest timestamp across all files in rootfs tree.
    for (auto it = fs::recursive_directory_iterator(rootfs_dir);
         it != fs::recursive_directory_iterator(); ++it) {
        fs::last_write_time(it->path(), now - std::chrono::minutes(2));
    }
    fs::last_write_time(rootfs_dir, now - std::chrono::minutes(2));
    fs::last_write_time(disk_image, now - std::chrono::minutes(1));

    exec->calls.clear();

    // Second call should detect image is current and skip rebuild.
    auto second = builder.ensure_disk_image({});
    REQUIRE(second);
    CHECK_FALSE(*second);
    CHECK_FALSE(has_command_call(*exec, "truncate"));
    CHECK_FALSE(has_command_call(*exec, "mke2fs"));
    CHECK_FALSE(has_command_call(*exec, "mkfs.ext4"));

    fs::remove_all(temp_dir);
}

TEST_CASE("Rootfs builder auto-sizes disk image from rootfs content", "[rootfs][disk-image]") {
    auto temp_dir = make_temp_dir();
    auto rootfs_dir = temp_dir / "rootfs";
    auto disk_image = temp_dir / "disk.img";

    write_file(rootfs_dir / "usr" / "share" / "payload.bin", std::string(8192, 'x'));

    elmos::config::Config cfg;
    cfg.image.volume_name = "elmos-test";
    cfg.image.size = std::string(elmos::config::kDefaultImageSize);
    cfg.paths.rootfs_dir = rootfs_dir.string();
    cfg.paths.disk_image = disk_image.string();

    auto exec = std::make_unique<elmos::infra::executor::MockExecutor>();
    auto filesystem = std::make_unique<elmos::infra::filesystem::OSFileSystem>();
    elmos::context::Context ctx(&cfg, exec.get(), filesystem.get());
    ctx.set_platform(std::make_unique<TestPlatform>());
    elmos::domain::rootfs::Builder builder(&ctx);

    auto result = builder.ensure_disk_image({});

    REQUIRE(result);
    CHECK(*result);

    const auto truncate_call =
        std::find_if(exec->calls.begin(), exec->calls.end(),
                     [](const auto& call) { return call.cmd == "truncate"; });
    REQUIRE(truncate_call != exec->calls.end());
    REQUIRE(truncate_call->args.size() >= 2);
    CHECK(truncate_call->args[1] != std::string(elmos::config::kDefaultImageSize));
    CHECK(std::stoull(truncate_call->args[1]) >= 1024ull * 1024ull * 1024ull);

    fs::remove_all(temp_dir);
}