// ============================================================================
// app/commands/bootloader.cpp — Bootloader build commands
// ============================================================================

#include <app/app.hpp>
#include <config/arch.hpp>
#include <domain/toolchain/manager.hpp>
#include <infra/executor/env.hpp>

#include "commands.hpp"

#include <algorithm>
#include <filesystem>
#include <map>
#include <stop_token>

namespace elmos::app::commands {

static auto bootloader_dir(const config::Config& cfg) -> std::filesystem::path {
    return std::filesystem::path(cfg.image.mount_point) / "bootloader";
}

// Build an environment for bootloader (U-Boot) builds.
// U-Boot uses GCC, NOT LLVM — omit LLVM=1 and use the GCC cross-compiler.
static auto get_bootloader_env(App& app) -> std::vector<std::string> {
    auto& cfg = app.config();
    auto ac = config::get_arch_config(cfg.build.arch);

    // Start with current environment, filtering out LLVM vars.
    auto env_map = infra::executor::get_current_env();
    std::map<std::string, std::string> merged_env;
    for (const auto& [key, val] : env_map) {
        if (key == "LLVM")
            continue;
        merged_env[key] = val;
    }

    // Add toolchain bin dir to PATH so cross-compiler is found
    if (ac) {
        auto gcc = ac->gcc_binary;
        std::string target;
        if (gcc.ends_with("-gcc"))
            target = gcc.substr(0, gcc.size() - 4);
        if (!target.empty()) {
            auto toolchain_bin =
                (std::filesystem::path(cfg.paths.toolchains_dir) / "x-tools" / target / "bin")
                    .string();
            auto& path = merged_env["PATH"];
            if (path.empty()) {
                path = toolchain_bin;
            }
            else {
                path = toolchain_bin + ":" + path;
            }
        }
    }

    // Ensure OpenSSL headers/libs are visible for U-Boot host tools.
    // This fixes include/image.h -> openssl/evp.h failures on macOS.
    auto append_flag = [&](const std::string& key, const std::string& flag) {
        if (flag.empty()) {
            return;
        }
        auto& current = merged_env[key];
        if (current.find(flag) != std::string::npos) {
            return;
        }
        if (current.empty()) {
            current = flag;
        }
        else {
            current = flag + " " + current;
        }
    };

    auto append_pkgconfig_path = [&](const std::string& dir) {
        if (dir.empty()) {
            return;
        }
        auto& current = merged_env["PKG_CONFIG_PATH"];
        if (current.find(dir) != std::string::npos) {
            return;
        }
        if (current.empty()) {
            current = dir;
        }
        else {
            current = dir + ":" + current;
        }
    };

    for (const auto* pkg : {"openssl@3", "openssl"}) {
        auto include_dir = app.platform().packages().get_include_path(pkg);
        auto lib_dir = app.platform().packages().get_lib_path(pkg);

        if (!include_dir.empty()) {
            append_flag("CPPFLAGS", "-I" + include_dir);
        }
        if (!lib_dir.empty()) {
            append_flag("LDFLAGS", "-L" + lib_dir);
            append_pkgconfig_path((std::filesystem::path(lib_dir) / "pkgconfig").string());
        }
    }

    std::vector<std::string> env;
    env.reserve(merged_env.size());
    for (const auto& [key, val] : merged_env) {
        env.push_back(key + "=" + val);
    }

    return env;
}

// Derive CROSS_COMPILE prefix from the arch config's gcc_binary name.
static auto get_cross_compile_prefix(const config::ArchConfig* ac) -> std::string {
    if (!ac || ac->gcc_binary.empty())
        return {};
    auto gcc = ac->gcc_binary;
    if (gcc.ends_with("gcc"))
        return gcc.substr(0, gcc.size() - 3);  // "xxx-gcc" -> "xxx-"
    return {};
}

static auto collect_openssl_paths(App& app)
    -> std::pair<std::vector<std::string>, std::vector<std::string>> {
    namespace fs = std::filesystem;

    std::vector<std::string> includes;
    std::vector<std::string> libs;

    auto add_unique_dir = [](std::vector<std::string>& out, const std::string& dir) {
        if (dir.empty()) {
            return;
        }
        if (std::find(out.begin(), out.end(), dir) == out.end()) {
            out.push_back(dir);
        }
    };

    for (const auto* pkg : {"openssl@3", "openssl"}) {
        auto include_dir = app.platform().packages().get_include_path(pkg);
        auto lib_dir = app.platform().packages().get_lib_path(pkg);
        if (!include_dir.empty() && fs::is_directory(include_dir)) {
            add_unique_dir(includes, include_dir);
        }
        if (!lib_dir.empty() && fs::is_directory(lib_dir)) {
            add_unique_dir(libs, lib_dir);
        }
    }

#ifdef __APPLE__
    // Fallbacks for Homebrew layouts when resolver package names differ.
    for (const auto* prefix : {"/opt/homebrew/opt/openssl@3", "/usr/local/opt/openssl@3",
                               "/opt/homebrew/opt/openssl", "/usr/local/opt/openssl"}) {
        auto include_dir = (fs::path(prefix) / "include").string();
        auto lib_dir = (fs::path(prefix) / "lib").string();
        if (fs::is_directory(include_dir)) {
            add_unique_dir(includes, include_dir);
        }
        if (fs::is_directory(lib_dir)) {
            add_unique_dir(libs, lib_dir);
        }
    }
#endif

    return {includes, libs};
}

static auto append_bootloader_host_crypto_flags(App& app, std::vector<std::string>& args) -> void {
    auto [include_dirs, lib_dirs] = collect_openssl_paths(app);
    if (include_dirs.empty() && lib_dirs.empty()) {
        return;
    }

    std::string include_flags;
    for (const auto& dir : include_dirs) {
        if (!include_flags.empty()) {
            include_flags += " ";
        }
        include_flags += "-I" + dir;
    }

    std::string lib_flags;
    for (const auto& dir : lib_dirs) {
        if (!lib_flags.empty()) {
            lib_flags += " ";
        }
        lib_flags += "-L" + dir;
    }

    if (!include_flags.empty()) {
        args.push_back("HOSTCFLAGS=" + include_flags);
        args.push_back("HOSTCPPFLAGS=" + include_flags);
    }
    if (!lib_flags.empty()) {
        args.push_back("HOSTLDFLAGS=" + lib_flags);
    }
}

void register_bootloader(App& app, CLI::App& cli) {
    auto* bl = cli.add_subcommand("bootloader", "Build and configure bootloader (U-Boot)");

    // bootloader clone
    auto* clone_cmd = bl->add_subcommand("clone", "Clone U-Boot repository");
    clone_cmd->callback([&app] {
        auto dir = bootloader_dir(app.config()).string();
        if (std::filesystem::exists(std::filesystem::path(dir) / ".git")) {
            app.printer().info("U-Boot repository already cloned at {}", dir);
            return;
        }
        std::filesystem::create_directories(dir);
        app.printer().step("Cloning U-Boot to {}...", dir);
        std::stop_source ss;
        if (auto r = app.exec().run(ss.get_token(), "git",
                                    {"clone", "https://source.denx.de/u-boot/u-boot.git", dir});
            !r) {
            app.printer().error("Clone failed: {}", r.error().message());
            return;
        }
        app.printer().success("U-Boot cloned!");
    });

    // bootloader config [config_type]
    auto* config_cmd = bl->add_subcommand("config",
                                          "Configure bootloader for current architecture");
    auto* config_type = config_cmd
                            ->add_option("type", "Config type (default: qemu defconfig for arch)")
                            ->default_val("");
    config_cmd->callback([&app, config_type] {
        auto dir = bootloader_dir(app.config()).string();
        if (!std::filesystem::exists(std::filesystem::path(dir) / "Makefile")) {
            app.printer().error("U-Boot not cloned. Run 'elmos bootloader clone' first.");
            return;
        }
        auto type = config_type->as<std::string>();
        auto& arch = app.config().build.arch;
        if (type.empty()) {
            // Default defconfig based on architecture
            if (arch == "arm64")
                type = "qemu_arm64_defconfig";
            else if (arch == "arm")
                type = "qemu_arm_defconfig";
            else if (arch == "riscv")
                type = "qemu-riscv64_smode_defconfig";
            else
                type = "defconfig";
        }

        auto ac = config::get_arch_config(arch);
        app.printer().step("Configuring U-Boot ({})...", type);
        std::stop_source ss;

        // Use bootloader-specific env (GCC cross-compiler, no LLVM)
        auto env = get_bootloader_env(app);
        std::vector<std::string> args = {"-C", dir};
        if (ac) {
            args.push_back("ARCH=" + ac->kernel_arch);
        }
        auto cross_compile = get_cross_compile_prefix(ac);
        if (!cross_compile.empty()) {
            args.push_back("CROSS_COMPILE=" + cross_compile);
        }
        append_bootloader_host_crypto_flags(app, args);
        args.push_back(type);

        if (auto r = app.exec().run_with_env_in_dir(ss.get_token(), env, dir, "make", args); !r) {
            app.printer().error("Configuration failed: {}", r.error().message());
            return;
        }
        app.printer().success("Bootloader configured!");
    });

    // bootloader build
    auto* build_cmd = bl->add_subcommand("build", "Build bootloader");
    build_cmd->callback([&app] {
        auto dir = bootloader_dir(app.config()).string();
        if (!std::filesystem::exists(std::filesystem::path(dir) / "Makefile")) {
            app.printer().error("U-Boot not cloned. Run 'elmos bootloader clone' first.");
            return;
        }

        auto& arch = app.config().build.arch;
        auto ac = config::get_arch_config(arch);
        int jobs = app.config().build.jobs;

        app.printer().step("Building bootloader for {}...", arch);
        std::stop_source ss;

        // Use bootloader-specific env (GCC cross-compiler, no LLVM)
        auto env = get_bootloader_env(app);
        std::vector<std::string> args = {"-C", dir, "-j", std::to_string(jobs)};
        if (ac) {
            args.push_back("ARCH=" + ac->kernel_arch);
        }
        auto cross_compile = get_cross_compile_prefix(ac);
        if (!cross_compile.empty()) {
            args.push_back("CROSS_COMPILE=" + cross_compile);
        }
        append_bootloader_host_crypto_flags(app, args);

        if (auto r = app.exec().run_with_env_in_dir(ss.get_token(), env, dir, "make", args); !r) {
            app.printer().error("Build failed: {}", r.error().message());
            return;
        }
        app.printer().success("Bootloader built!");
    });

    // bootloader status
    auto* status_cmd = bl->add_subcommand("status", "Show bootloader status");
    status_cmd->callback([&app] {
        auto dir = bootloader_dir(app.config());
        const auto& arch = app.config().build.arch;
        app.printer().step("Bootloader status:");
        app.printer().print("  Architecture: {}", arch);
        app.printer().print("  Directory:    {}", dir.string());

        if (!std::filesystem::exists(dir / "Makefile")) {
            app.printer().print("  ○ Source not cloned");
            app.printer().info("  Next: 'elmos bootloader clone' then 'elmos bootloader config'");
            return;
        }

        app.printer().print("  ✓ Source cloned");
        bool configured = std::filesystem::exists(dir / ".config");
        app.printer().print("  {} Configured", configured ? "✓" : "○");

        bool built = std::filesystem::exists(dir / "u-boot") ||
                     std::filesystem::exists(dir / "u-boot.bin") ||
                     std::filesystem::exists(dir / "u-boot.itb");
        app.printer().print("  {} Built", built ? "✓" : "○");

        if (built) {
            std::string qemu_hint;
            if (arch == "arm64") {
                qemu_hint = "QEMU: -bios u-boot.bin  (arm64 firmware slot)";
            }
            else if (arch == "arm") {
                qemu_hint = "QEMU: -kernel u-boot    (arm32 kernel slot)";
            }
            else if (arch == "riscv") {
                qemu_hint = "QEMU: -bios default (OpenSBI) + -kernel u-boot.bin  (RISC-V S-mode)";
            }
            if (!qemu_hint.empty()) {
                app.printer().print("  ℹ {}", qemu_hint);
            }
            app.printer().print(
                "  ℹ Disk image needs FAT32 boot partition: run 'elmos rootfs build --uboot'");
        }
        else if (configured) {
            app.printer().info("  Next: 'elmos bootloader build'");
        }
        else {
            app.printer().info("  Next: 'elmos bootloader config' then 'elmos bootloader build'");
        }
    });

    // bootloader show
    auto* show_cmd = bl->add_subcommand("show", "Show bootloader build result");
    show_cmd->callback([&app] {
        namespace fs = std::filesystem;
        auto dir = bootloader_dir(app.config());
        const auto& arch = app.config().build.arch;
        app.printer().step("Bootloader build result:");
        app.printer().print("  Architecture: {}", arch);
        app.printer().print("  Directory:    {}", dir.string());

        if (!fs::exists(dir / "Makefile")) {
            app.printer().info("  U-Boot not cloned. Run 'elmos bootloader clone'");
            return;
        }
        app.printer().print("  ✓ U-Boot source present");

        bool any = false;
        for (const auto& artifact : {"u-boot", "u-boot.bin", "u-boot.itb", "spl/u-boot-spl"}) {
            auto path = dir / artifact;
            if (fs::exists(path)) {
                auto size = fs::file_size(path);
                app.printer().print("  ✓ {} ({} KB)", artifact, size / 1024);
                any = true;
            }
        }
        if (!any) {
            app.printer().print("  ○ No build artifacts found — run 'elmos bootloader build'");
            return;
        }

        app.printer().print("");
        app.printer().print("  QEMU integration:");
        if (arch == "arm64") {
            app.printer().print("    qemu-system-aarch64 ... -bios u-boot.bin");
            app.printer().print("    (U-Boot replaces firmware; no -kernel/-append)");
        }
        else if (arch == "arm") {
            app.printer().print("    qemu-system-arm ... -kernel u-boot");
            app.printer().print("    (U-Boot at kernel slot; no -append)");
        }
        else if (arch == "riscv") {
            app.printer().print("    qemu-system-riscv64 ... -bios default -kernel u-boot.bin");
            app.printer().print("    (OpenSBI M-mode + U-Boot S-mode; no -append)");
        }
        app.printer().print("");
        app.printer().print(
            "  Required disk image layout (create with 'elmos rootfs build --uboot'):");
        app.printer().print(
            "    Partition 1 [FAT32, 128 MiB]: /extlinux/extlinux.conf + kernel image");
        app.printer().print("    Partition 2 [ext4,  rest    ]: Debian rootfs");
    });

    // bootloader clean
    auto* clean_cmd = bl->add_subcommand("clean", "Clean bootloader build artifacts");
    clean_cmd->callback([&app] {
        auto dir = bootloader_dir(app.config()).string();
        if (!std::filesystem::exists(std::filesystem::path(dir) / "Makefile")) {
            app.printer().info("Nothing to clean");
            return;
        }
        app.printer().step("Cleaning bootloader build...");
        std::stop_source ss;
        if (auto r = app.exec().run(ss.get_token(), "make", {"-C", dir, "distclean"}); !r) {
            app.printer().error("Clean failed: {}", r.error().message());
            return;
        }
        app.printer().success("Bootloader cleaned!");
    });

    // Show help when 'elmos bootloader' is run with no subcommand
    bl->callback([bl] {
        if (bl->get_subcommands().empty()) {
            std::cout << bl->help();
        }
    });
}

}  // namespace elmos::app::commands
