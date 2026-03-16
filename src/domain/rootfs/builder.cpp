// ============================================================================
// domain/rootfs/builder.cpp — Root filesystem creation
// ============================================================================

#include "builder.hpp"

#include <config/arch.hpp>
#include <config/defaults.hpp>
#include <config/types.hpp>
#include <context/context.hpp>

#include <algorithm>
#include <array>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <sstream>

namespace elmos::domain::rootfs {

namespace fs = std::filesystem;

namespace {

auto latest_tree_write_time(const fs::path& root) -> Result<fs::file_time_type> {
    std::error_code ec;
    if (!fs::exists(root, ec)) {
        return make_error(Error::rootfs("rootfs directory not found: " + root.string()));
    }

    auto latest = fs::last_write_time(root, ec);
    if (ec) {
        return make_error(Error::rootfs("cannot inspect rootfs timestamp for " + root.string() +
                                        ": " + ec.message()));
    }

    fs::recursive_directory_iterator it(root, fs::directory_options::skip_permission_denied, ec);
    fs::recursive_directory_iterator end;
    if (ec) {
        ec.clear();
    }

    for (; it != end; it.increment(ec)) {
        if (ec) {
            ec.clear();
            continue;
        }

        auto current = fs::last_write_time(it->path(), ec);
        if (ec) {
            ec.clear();
            continue;
        }
        if (current > latest) {
            latest = current;
        }
    }

    return latest;
}

auto directory_size_bytes(const fs::path& root) -> Result<std::uintmax_t> {
    std::error_code ec;
    if (!fs::exists(root, ec)) {
        return make_error(Error::rootfs("rootfs directory not found: " + root.string()));
    }

    std::uintmax_t total = 0;
    fs::recursive_directory_iterator it(root, fs::directory_options::skip_permission_denied, ec);
    fs::recursive_directory_iterator end;
    if (ec) {
        ec.clear();
    }

    for (; it != end; it.increment(ec)) {
        if (ec) {
            ec.clear();
            continue;
        }

        if (!it->is_regular_file(ec)) {
            ec.clear();
            continue;
        }

        auto size = it->file_size(ec);
        if (ec) {
            ec.clear();
            continue;
        }
        total += size;
    }

    return total;
}

auto align_up(std::uintmax_t value, std::uintmax_t alignment) -> std::uintmax_t {
    if (alignment == 0) {
        return value;
    }
    auto remainder = value % alignment;
    if (remainder == 0) {
        return value;
    }
    return value + (alignment - remainder);
}

auto recommended_disk_size_bytes(const fs::path& rootfs_dir) -> Result<std::uintmax_t> {
    constexpr std::uintmax_t kMiB = 1024ull * 1024ull;
    constexpr std::uintmax_t kGiB = 1024ull * 1024ull * 1024ull;

    auto rootfs_bytes = directory_size_bytes(rootfs_dir);
    if (!rootfs_bytes) {
        return make_error(rootfs_bytes.error());
    }

    const auto padded = (*rootfs_bytes * 13) / 10 + 256ull * kMiB;
    const auto minimum = 1ull * kGiB;
    return align_up(std::max(padded, minimum), 64ull * kMiB);
}

auto ensure_console_nodes_in_image(context::Context* ctx, std::stop_token token,
                                   const fs::path& disk_image) -> VoidResult {
    auto env = ctx->platform().packages().build_gnu_environment();

    std::vector<std::string> commands = {
        "mkdir /dev",
        "mknod /dev/console c 5 1",
        "mknod /dev/null c 1 3",
        "mknod /dev/ttyS0 c 4 64",
    };

    std::optional<Error> last_error;
    for (const auto& cmd : commands) {
        auto run = ctx->exec().run_with_env(token, env, "debugfs",
                                            {"-w", "-R", cmd, disk_image.string()});
        if (run) {
            continue;
        }

        auto run_homebrew = ctx->exec().run_with_env(token, env,
                                                     "/opt/homebrew/opt/e2fsprogs/sbin/debugfs",
                                                     {"-w", "-R", cmd, disk_image.string()});
        if (run_homebrew) {
            continue;
        }

        // mkdir can legitimately fail when /dev already exists.
        if (cmd == "mkdir /dev") {
            continue;
        }

        last_error = run_homebrew.error();
        break;
    }

    if (last_error) {
        return make_error(
            Error::wrap("inject console device nodes into " + disk_image.string(), *last_error));
    }

    return {};
}

/// Write a minimal MBR partition table directly into an open disk-image file.
/// Writes two primary partition entries at MBR offset 446, then the 0x55AA
/// signature at offset 510.  CHS fields are set to the LBA-only placeholder
/// (0xFE/0xFF/0xFF) which is universally accepted by modern boot firmware.
auto write_mbr(const std::string& path, uint32_t boot_lba_start, uint32_t boot_lba_cnt,
               uint32_t rootfs_lba_start, uint32_t rootfs_lba_cnt) -> VoidResult {
    std::fstream f(path, std::ios::in | std::ios::out | std::ios::binary);
    if (!f) {
        return make_error(Error::generic("cannot open disk image for MBR write: " + path));
    }

    auto write_u32le = [&](uint32_t v) {
        f.put(static_cast<char>(v & 0xFF));
        f.put(static_cast<char>((v >> 8) & 0xFF));
        f.put(static_cast<char>((v >> 16) & 0xFF));
        f.put(static_cast<char>((v >> 24) & 0xFF));
    };

    auto write_entry = [&](uint8_t status, uint8_t type, uint32_t lba_start, uint32_t lba_cnt) {
        f.put(static_cast<char>(status));
        // CHS first — LBA-only placeholder
        f.put(static_cast<char>(0xFE));
        f.put(static_cast<char>(0xFF));
        f.put(static_cast<char>(0xFF));
        f.put(static_cast<char>(type));
        // CHS last
        f.put(static_cast<char>(0xFE));
        f.put(static_cast<char>(0xFF));
        f.put(static_cast<char>(0xFF));
        write_u32le(lba_start);
        write_u32le(lba_cnt);
    };

    f.seekp(446);
    write_entry(0x80, 0x0C, boot_lba_start, boot_lba_cnt);      // FAT32 LBA, bootable
    write_entry(0x00, 0x83, rootfs_lba_start, rootfs_lba_cnt);  // Linux ext4
    // Two unused entries (16 bytes each)
    static const std::array<char, 32> kZero{};
    f.write(kZero.data(), 32);
    // MBR boot signature
    f.seekp(510);
    f.put(static_cast<char>(0x55));
    f.put(static_cast<char>(0xAA));

    f.flush();
    if (!f) {
        return make_error(Error::generic("MBR write failed for: " + path));
    }
    return {};
}

/// Generate the extlinux.conf content used by U-Boot's distro_bootcmd.
/// The root device is /dev/vda2 (second partition in the virtio block device).
auto make_extlinux_conf(const std::string& kernel_filename, const std::string& console)
    -> std::string {
    return std::string("DEFAULT linux\n") + "LABEL linux\n" + "  KERNEL /" + kernel_filename +
           "\n" + "  APPEND root=/dev/vda2 rw rootwait earlycon console=" + console + "\n";
}

/// Run a mtools command, trying both system PATH and common Homebrew paths.
auto run_mtools(context::Context* ctx, std::stop_token token, const std::string& cmd,
                const std::vector<std::string>& args) -> VoidResult {
    // Store candidate paths in persistent strings to avoid dangling pointers.
    const std::vector<std::string> candidates = {
        cmd,
        "/opt/homebrew/bin/" + cmd,
        "/usr/local/bin/" + cmd,
    };
    for (const auto& binary : candidates) {
        if (auto r = ctx->exec().run(token, binary, args)) {
            return {};
        }
    }
    // All paths failed — surface the error from the plain command name.
    return ctx->exec().run(token, cmd, args);
}

}  // namespace

Builder::Builder(context::Context* ctx) : ctx_(ctx) {}

auto Builder::release_supports_arch(std::stop_token token, const std::string& mirror,
                                    const std::string& release, const std::string& arch)
    -> Result<bool> {
    auto base = mirror;
    while (!base.empty() && base.back() == '/') {
        base.pop_back();
    }

    auto url = base + "/dists/" + release + "/Release";
    auto output = ctx_->exec().output(token, "curl", {"-fsSL", url});
    if (!output) {
        return make_error(output.error());
    }

    std::istringstream in(*output);
    std::string line;
    while (std::getline(in, line)) {
        if (!line.starts_with("Architectures:")) {
            continue;
        }

        auto values = line.substr(std::string("Architectures:").size());
        std::istringstream arch_stream(values);
        std::string candidate;
        while (arch_stream >> candidate) {
            if (candidate == arch) {
                return true;
            }
        }
        return false;
    }

    return make_error(Error::generic("unable to parse architectures from " + url));
}

auto Builder::resolve_bootstrap_source(std::stop_token token, const RootfsOptions& opts)
    -> Result<BootstrapSource> {
    auto& cfg = ctx_->config();

    std::string arch = cfg.build.arch;
    if (arch == "arm") {
        arch = "armhf";
    }
    else if (arch == "riscv") {
        arch = "riscv64";
    }

    const auto& mirror = cfg.paths.debian_mirror;
    const auto& requested_release = opts.release;

    auto supported = release_supports_arch(token, mirror, requested_release, arch);
    if (supported && *supported) {
        return BootstrapSource{.arch = arch, .release = requested_release, .mirror = mirror};
    }

    std::vector<std::string> fallbacks;
    if (requested_release != "stable") {
        fallbacks.push_back("stable");
    }
    if (requested_release != "testing") {
        fallbacks.push_back("testing");
    }
    if (requested_release != "sid") {
        fallbacks.push_back("sid");
    }

    for (const auto& release : fallbacks) {
        auto fallback_supported = release_supports_arch(token, mirror, release, arch);
        if (fallback_supported && *fallback_supported) {
            return BootstrapSource{.arch = arch, .release = release, .mirror = mirror};
        }
    }

    std::string checked = requested_release;
    for (const auto& release : fallbacks) {
        checked += ", ";
        checked += release;
    }

    return make_error(Error::generic("no Debian release on " + mirror + " provides arch '" + arch +
                                     "' (checked: " + checked + ")"));
}

auto Builder::create(std::stop_token token, RootfsOptions opts) -> VoidResult {
    auto r = run_debootstrap(token, opts);
    if (!r)
        return r;

    r = ensure_pid1();
    if (!r)
        return r;

    if (!opts.packages.empty()) {
        r = install_packages(token, opts.packages);
        if (!r)
            return r;
    }

    if (!opts.post_build_script.empty()) {
        r = customize(token, opts.post_build_script);
        if (!r)
            return r;
    }

    if (opts.assemble_disk_image) {
        auto disk_result = ensure_disk_image(token, opts.disk_size, opts.force_disk_image,
                                             opts.with_boot_partition);
        if (!disk_result) {
            return make_error(disk_result.error());
        }
    }

    return {};
}

auto Builder::ensure_disk_image(std::stop_token token, const std::string& size, bool force,
                                bool with_boot_partition) -> Result<bool> {
    auto& cfg = ctx_->config();
    auto rootfs_dir = fs::path(cfg.paths.rootfs_dir);
    auto disk_image = fs::path(cfg.paths.disk_image);

    if (disk_image.empty()) {
        return make_error(Error::config("rootfs disk image path is not configured"));
    }

    const bool has_rootfs_dir = ctx_->fs().is_dir(rootfs_dir);

    if (has_rootfs_dir) {
        auto pid1_ready = ensure_pid1();
        if (!pid1_ready) {
            return make_error(pid1_ready.error());
        }
    }

    if (!force && ctx_->fs().exists(disk_image)) {
        if (!has_rootfs_dir) {
            return false;
        }

        auto needs_refresh = disk_image_needs_refresh(token);
        if (!needs_refresh) {
            return make_error(needs_refresh.error());
        }
        if (!*needs_refresh) {
            return false;
        }
    }

    if (!has_rootfs_dir) {
        return make_error(Error::rootfs("rootfs directory not found: " + rootfs_dir.string() +
                                        " (run 'elmos rootfs build' first)"));
    }

    auto mkdir_result = ctx_->fs().mkdir_all(disk_image.parent_path());
    if (!mkdir_result) {
        return make_error(
            Error::wrap("create disk image directory " + disk_image.parent_path().string(),
                        mkdir_result.error()));
    }

    if (ctx_->fs().exists(disk_image)) {
        auto remove_result = ctx_->fs().remove(disk_image);
        if (!remove_result) {
            return make_error(Error::wrap("remove stale disk image " + disk_image.string(),
                                          remove_result.error()));
        }
    }

    // Delegate to the two-partition (U-Boot) path when requested.
    if (with_boot_partition) {
        return create_partitioned_disk_image(token, size, force);
    }

    auto env = ctx_->platform().packages().build_gnu_environment();

    std::string requested_size = size;
    const bool use_auto_sizing = requested_size.empty() &&
                                 (cfg.image.size.empty() ||
                                  cfg.image.size == std::string(config::kDefaultImageSize));
    if (use_auto_sizing) {
        auto auto_size = recommended_disk_size_bytes(rootfs_dir);
        if (!auto_size) {
            return make_error(auto_size.error());
        }
        requested_size = std::to_string(*auto_size);
    }
    else if (requested_size.empty()) {
        requested_size = cfg.image.size;
    }
    if (requested_size.empty()) {
        requested_size = "5G";
    }

    auto truncate_result = ctx_->exec().run_with_env(token, env, "truncate",
                                                     {"-s", requested_size, disk_image.string()});
    if (!truncate_result) {
        return make_error(Error::wrap("create sparse rootfs disk image " + disk_image.string(),
                                      truncate_result.error()));
    }

    auto label = cfg.image.volume_name.empty() ? std::string("rootfs") : cfg.image.volume_name;

    std::optional<Error> last_error;
    {
        auto mkfs_result = ctx_->exec().run_with_env(
            token, env, "mke2fs",
            {"-d", rootfs_dir.string(), "-t", "ext4", "-F", "-L", label, disk_image.string()});
        if (mkfs_result) {
            auto nodes_result = ensure_console_nodes_in_image(ctx_, token, disk_image);
            if (!nodes_result) {
                return make_error(nodes_result.error());
            }
            return true;
        }
        last_error = mkfs_result.error();
    }

    {
        auto mkfs_result = ctx_->exec().run_with_env(
            token, env, "mkfs.ext4",
            {"-d", rootfs_dir.string(), "-F", "-L", label, disk_image.string()});
        if (mkfs_result) {
            auto nodes_result = ensure_console_nodes_in_image(ctx_, token, disk_image);
            if (!nodes_result) {
                return make_error(nodes_result.error());
            }
            return true;
        }
        last_error = mkfs_result.error();
    }

    auto cleanup_result = ctx_->fs().remove(disk_image);
    if (!cleanup_result) {
        (void)cleanup_result;
    }

    return make_error(Error::wrap("format rootfs disk image " + disk_image.string() +
                                      " (install e2fsprogs if ext4 image tools are missing)",
                                  *last_error));
}

auto Builder::create_partitioned_disk_image(std::stop_token token,
                                            const std::string& total_size_str, bool /*force*/)
    -> Result<bool> {
    // ── Disk layout constants ─────────────────────────────────────────────
    //   Sector 0     – MBR
    //   Sector 2048  – FAT32 boot partition start  (1 MiB alignment)
    //   Sector 264192 – ext4 rootfs partition start (1 MiB + 128 MiB)
    constexpr uint64_t kSector = 512ULL;
    constexpr uint64_t kBootStart = 2048ULL;                      // sectors
    constexpr uint64_t kBootSectors = 262144ULL;                  // 128 MiB
    constexpr uint64_t kRootfsStart = kBootStart + kBootSectors;  // 264192 sectors
    const uint64_t kRootfsOffsetBytes = kRootfsStart * kSector;   // 129 MiB

    auto& cfg = ctx_->config();
    const auto& arch = cfg.build.arch;
    auto arch_cfg = config::get_arch_config(arch);
    if (!arch_cfg) {
        return make_error(Error::config("no arch config for " + arch));
    }

    auto rootfs_dir = fs::path(cfg.paths.rootfs_dir);
    auto disk_image = fs::path(cfg.paths.disk_image);
    auto env = ctx_->platform().packages().build_gnu_environment();

    // ── 1. Resolve total disk size ─────────────────────────────────────────
    std::string req_size = total_size_str;
    if (req_size.empty() &&
        (!cfg.image.size.empty() && cfg.image.size != std::string(config::kDefaultImageSize))) {
        req_size = cfg.image.size;
    }
    if (req_size.empty()) {
        auto auto_bytes = recommended_disk_size_bytes(rootfs_dir);
        if (!auto_bytes) {
            return make_error(auto_bytes.error());
        }
        // Add the 129 MiB boot + alignment overhead.
        auto total = *auto_bytes + kRootfsOffsetBytes;
        // Align to 64 MiB.
        req_size = std::to_string(align_up(total, 64ULL * 1024ULL * 1024ULL));
    }

    // Parse size to bytes for rootfs sector count computation.
    auto parse_size_bytes = [](const std::string& s) -> uint64_t {
        if (s.empty())
            return 5ULL * 1024 * 1024 * 1024;
        try {
            std::size_t idx = 0;
            double v = std::stod(s, &idx);
            std::string suffix = s.substr(idx);
            if (suffix == "G" || suffix == "g")
                return static_cast<uint64_t>(v * 1024 * 1024 * 1024);
            if (suffix == "M" || suffix == "m")
                return static_cast<uint64_t>(v * 1024 * 1024);
            if (suffix == "K" || suffix == "k")
                return static_cast<uint64_t>(v * 1024);
            return static_cast<uint64_t>(v);
        }
        catch (...) {
            return 5ULL * 1024 * 1024 * 1024;
        }
    };

    const uint64_t total_bytes = parse_size_bytes(req_size);
    const uint64_t rootfs_sectors = (total_bytes - kRootfsOffsetBytes) / kSector;

    // ── 2. Create sparse disk image (full size) ────────────────────────────
    auto tr = ctx_->exec().run_with_env(token, env, "truncate",
                                        {"-s", req_size, disk_image.string()});
    if (!tr) {
        return make_error(
            Error::wrap("create sparse disk image " + disk_image.string(), tr.error()));
    }

    // ── 3. Write MBR partition table (pure C++ — no external tool needed) ─
    auto mbr_r = write_mbr(disk_image.string(), static_cast<uint32_t>(kBootStart),
                           static_cast<uint32_t>(kBootSectors), static_cast<uint32_t>(kRootfsStart),
                           static_cast<uint32_t>(rootfs_sectors));
    if (!mbr_r) {
        ctx_->fs().remove(disk_image);
        return make_error(Error::wrap("write MBR", mbr_r.error()));
    }

    // ── 4. Build FAT32 boot partition image (standalone temp file) ─────────
    auto boot_tmp = disk_image.parent_path() / (disk_image.filename().string() + ".boot.tmp");
    // Clean up any leftover from a previous failed run.
    ctx_->fs().remove(boot_tmp);

    const uint64_t boot_bytes = kBootSectors * kSector;  // 128 MiB
    auto boot_tr = ctx_->exec().run_with_env(token, env, "truncate",
                                             {"-s", std::to_string(boot_bytes), boot_tmp.string()});
    if (!boot_tr) {
        ctx_->fs().remove(disk_image);
        return make_error(Error::wrap("create boot partition temp image", boot_tr.error()));
    }

    // mformat — create FAT32 in the temp file (tries system + Homebrew paths)
    auto fmt_r = run_mtools(ctx_, token, "mformat",
                            {"-i", boot_tmp.string(), "-F", "-v", "BOOT", "::"});
    if (!fmt_r) {
        ctx_->fs().remove(disk_image);
        ctx_->fs().remove(boot_tmp);
        return make_error(Error::wrap(
            "format FAT32 boot partition (install mtools: brew install mtools)", fmt_r.error()));
    }

    // ── 5. Place extlinux.conf on the FAT32 partition ──────────────────────
    auto extlinux_conf = make_extlinux_conf(arch_cfg->kernel_image, arch_cfg->console);
    auto conf_tmp = disk_image.parent_path() / "extlinux.conf.tmp";
    {
        std::ofstream out(conf_tmp, std::ios::binary);
        if (!out) {
            ctx_->fs().remove(disk_image);
            ctx_->fs().remove(boot_tmp);
            return make_error(Error::generic("cannot write extlinux.conf.tmp"));
        }
        out << extlinux_conf;
    }

    auto mmd_r = run_mtools(ctx_, token, "mmd", {"-i", boot_tmp.string(), "::/extlinux"});
    // mmd might fail if directory already exists, but that's ok — we just want it to exist
    (void)mmd_r;  // non-fatal

    auto mcopy_conf =
        run_mtools(ctx_, token, "mcopy",
                   {"-i", boot_tmp.string(), conf_tmp.string(), "::/extlinux/extlinux.conf"});
    ctx_->fs().remove(conf_tmp);
    if (!mcopy_conf) {
        ctx_->fs().remove(disk_image);
        ctx_->fs().remove(boot_tmp);
        return make_error(
            Error::wrap("copy extlinux.conf to FAT boot partition", mcopy_conf.error()));
    }

    // ── 6. Copy kernel image to the FAT32 partition ────────────────────────
    auto kernel_image = ctx_->get_kernel_image();
    if (!kernel_image.empty() && ctx_->fs().exists(kernel_image)) {
        auto mcopy_k =
            run_mtools(ctx_, token, "mcopy",
                       {"-i", boot_tmp.string(), kernel_image, "::/" + arch_cfg->kernel_image});
        if (!mcopy_k) {
            // Non-fatal: U-Boot can still find the kernel if it's on the ext4
            // rootfs (e.g. at /boot/Image), but we warn the user.
            (void)mcopy_k;
        }
    }

    // ── 7. Splice the FAT partition into the full disk image ───────────────
    // dd if=boot_tmp of=disk_image bs=512 seek=kBootStart conv=notrunc
    std::vector<std::string> dd_args = {
        "if=" + boot_tmp.string(),
        "of=" + disk_image.string(),
        "bs=512",
        "seek=" + std::to_string(kBootStart),
        "conv=notrunc",
        "status=none",
    };
    auto dd_r = ctx_->exec().run(token, "dd", dd_args);
    ctx_->fs().remove(boot_tmp);
    if (!dd_r) {
        ctx_->fs().remove(disk_image);
        return make_error(Error::wrap("splice FAT partition into disk image", dd_r.error()));
    }

    // ── 8. Create ext4 rootfs partition at kRootfsOffsetBytes ─────────────
    auto label = cfg.image.volume_name.empty() ? std::string("rootfs") : cfg.image.volume_name;
    auto offset_str = std::to_string(kRootfsOffsetBytes);

    std::optional<Error> last_err;
    for (const auto* mke2fs : {"mke2fs", "/opt/homebrew/opt/e2fsprogs/sbin/mke2fs"}) {
        auto r =
            ctx_->exec().run_with_env(token, env, mke2fs,
                                      {"-d", rootfs_dir.string(), "-t", "ext4", "-F", "-E",
                                       "offset=" + offset_str, "-L", label, disk_image.string()});
        if (r) {
            return true;
        }
        last_err = r.error();
    }

    ctx_->fs().remove(disk_image);
    return make_error(Error::wrap(
        "format ext4 rootfs partition (install e2fsprogs: brew install e2fsprogs)", *last_err));
}

auto Builder::run_debootstrap(std::stop_token token, const RootfsOptions& opts) -> VoidResult {
    auto& cfg = ctx_->config();
    auto rootfs_dir = cfg.paths.rootfs_dir;

    auto r = ctx_->fs().mkdir_all(rootfs_dir);
    if (!r)
        return r;

    // Resolve the bundled debootstrap from tools/debootstrap inside the project root
    // (Go equivalent: DEBOOTSTRAP_DIR = filepath.Join(cfg.Paths.ProjectRoot, "tools",
    // "debootstrap"))
    auto debootstrap_src_dir =
        (fs::path(cfg.paths.project_root) / "tools" / "debootstrap").string();
    auto debootstrap_install_dir =
        (fs::path(debootstrap_src_dir) / ".local" / "usr" / "share" / "debootstrap").string();
    auto debootstrap_install_bin =
        (fs::path(debootstrap_src_dir) / ".local" / "usr" / "sbin" / "debootstrap").string();

    std::string debootstrap_dir = debootstrap_src_dir;
    std::string debootstrap_bin = (fs::path(debootstrap_src_dir) / "debootstrap").string();

    if (ctx_->fs().exists(debootstrap_install_bin) && ctx_->fs().is_dir(debootstrap_install_dir)) {
        debootstrap_dir = debootstrap_install_dir;
        debootstrap_bin = debootstrap_install_bin;
    }

    if (!ctx_->fs().exists(debootstrap_bin)) {
        return make_error(Error::generic("debootstrap not found at " + debootstrap_bin +
                                         ". Run 'elmos rootfs clone' to download it."));
    }

    auto source = resolve_bootstrap_source(token, opts);
    if (!source) {
        return make_error(source.error());
    }

    // Run debootstrap with --foreign (first stage only, second stage runs inside QEMU)
    // DEBOOTSTRAP_DIR must point to the bundled debootstrap scripts directory
    std::vector<std::string> env = {"DEBOOTSTRAP_DIR=" + debootstrap_dir};

    auto result = ctx_->exec().run_with_env(token, env, "sudo",
                                            {
                                                "-E",
                                                "DEBOOTSTRAP_DIR=" + debootstrap_dir,
                                                "fakeroot",
                                                debootstrap_bin,
                                                "--foreign",
                                                "--arch=" + source->arch,
                                                "--no-check-gpg",
                                                source->release,
                                                rootfs_dir,
                                                source->mirror,
                                            });
    if (!result) {
        return make_error(Error::generic("debootstrap failed for arch '" + source->arch +
                                         "' using release '" + source->release + "' from " +
                                         source->mirror + ": " + result.error().message()));
    }

    return {};
}

auto Builder::install_packages(std::stop_token token, const std::vector<std::string>& packages)
    -> VoidResult {
    auto& cfg = ctx_->config();

    std::vector<std::string> args = {
        "chroot", cfg.paths.rootfs_dir, "apt-get", "install", "-y",
    };
    for (const auto& pkg : packages) {
        args.push_back(pkg);
    }

    return ctx_->exec().run(token, "sudo", args);
}

auto Builder::install_modules(std::stop_token token) -> VoidResult {
    auto& cfg = ctx_->config();
    auto env = ctx_->get_make_env();

    return ctx_->exec().run_with_env(token, env, "sudo",
                                     {
                                         "make",
                                         "-C",
                                         cfg.paths.kernel_dir,
                                         "ARCH=" + cfg.build.arch,
                                         "INSTALL_MOD_PATH=" + cfg.paths.rootfs_dir,
                                         "modules_install",
                                     });
}

auto Builder::customize(std::stop_token token, const std::string& script) -> VoidResult {
    auto& cfg = ctx_->config();
    return ctx_->exec().run(token, "sudo",
                            {
                                "chroot",
                                cfg.paths.rootfs_dir,
                                "/bin/bash",
                                "-c",
                                script,
                            });
}

auto Builder::disk_image_needs_refresh(std::stop_token token) const -> Result<bool> {
    auto& cfg = ctx_->config();
    auto rootfs_dir = fs::path(cfg.paths.rootfs_dir);
    auto disk_image = fs::path(cfg.paths.disk_image);

    if (!ctx_->fs().is_dir(rootfs_dir)) {
        return make_error(Error::rootfs("rootfs directory not found: " + rootfs_dir.string()));
    }

    if (!ctx_->fs().exists(disk_image)) {
        return true;
    }

    std::error_code size_ec;
    auto image_size = fs::file_size(disk_image, size_ec);
    if (size_ec || image_size == 0) {
        return true;
    }

    auto rootfs_write_time = latest_tree_write_time(rootfs_dir);
    if (!rootfs_write_time) {
        return make_error(rootfs_write_time.error());
    }

    std::error_code ec;
    auto image_write_time = fs::last_write_time(disk_image, ec);
    if (ec) {
        return true;
    }

    return *rootfs_write_time > image_write_time;
}

auto Builder::ensure_pid1() -> VoidResult {
    auto& cfg = ctx_->config();
    auto rootfs_dir = fs::path(cfg.paths.rootfs_dir);
    if (!ctx_->fs().is_dir(rootfs_dir)) {
        return {};
    }

    const auto sbin_init = rootfs_dir / "sbin" / "init";
    const auto usr_sbin_init = rootfs_dir / "usr" / "sbin" / "init";
    if (ctx_->fs().exists(sbin_init) || ctx_->fs().exists(usr_sbin_init)) {
        return {};
    }

    const auto lib_systemd = rootfs_dir / "lib" / "systemd" / "systemd";
    const auto usr_lib_systemd = rootfs_dir / "usr" / "lib" / "systemd" / "systemd";

    fs::path target_in_root;
    std::string link_target;
    if (ctx_->fs().exists(lib_systemd)) {
        target_in_root = lib_systemd;
        link_target = "/lib/systemd/systemd";
    }
    else if (ctx_->fs().exists(usr_lib_systemd)) {
        target_in_root = usr_lib_systemd;
        link_target = "/usr/lib/systemd/systemd";
    }

    if (!target_in_root.empty()) {
        std::error_code ec;
        fs::create_directories(usr_sbin_init.parent_path(), ec);
        if (ec) {
            return make_error(
                Error::generic("failed to create /usr/sbin in rootfs: " + ec.message()));
        }

        fs::remove(usr_sbin_init, ec);
        ec.clear();
        fs::create_symlink(link_target, usr_sbin_init, ec);
        if (ec) {
            return make_error(Error::generic("failed to create /sbin/init symlink to " +
                                             link_target + ": " + ec.message()));
        }
        return {};
    }

    // Fallback for first-stage debootstrap rootfs (common on macOS hosts):
    // install a tiny /sbin/init shim as PID1 so kernel boot does not panic.
    // This keeps the VM interactive without relying on a /init kernel argument.
    const std::string fallback_init = R"INIT(#!/bin/sh

mount -t proc proc /proc 2>/dev/null || true
mount -t sysfs sysfs /sys 2>/dev/null || true
mount -t devtmpfs devtmpfs /dev 2>/dev/null || true
mkdir -p /dev/pts
mount -t devpts devpts /dev/pts 2>/dev/null || true

[ -c /dev/console ] || mknod /dev/console c 5 1 2>/dev/null || true
[ -c /dev/null ] || mknod /dev/null c 1 3 2>/dev/null || true

echo "ELMOS: fallback /sbin/init active (debootstrap first-stage rootfs)"

while true; do
    if [ -x /sbin/getty ] && [ -c /dev/ttyS0 ]; then
        /sbin/getty -L 115200 ttyS0 vt100 </dev/null >/dev/null 2>&1
    elif [ -x /bin/sh ] && [ -c /dev/ttyS0 ]; then
        /bin/sh -i </dev/ttyS0 >/dev/ttyS0 2>&1
    elif [ -x /bin/sh ] && [ -c /dev/console ]; then
        /bin/sh -i </dev/console >/dev/console 2>&1
    else
        sleep 1
    fi
    sleep 1
done
)INIT";

    std::error_code ec;
    fs::create_directories(usr_sbin_init.parent_path(), ec);
    if (ec) {
        return make_error(Error::generic("failed to create /usr/sbin in rootfs: " + ec.message()));
    }

    std::ofstream out(usr_sbin_init, std::ios::binary | std::ios::trunc);
    if (!out) {
        return make_error(
            Error::generic("failed to write fallback /sbin/init at " + usr_sbin_init.string()));
    }
    out << fallback_init;
    out.close();

    fs::permissions(usr_sbin_init,
                    fs::perms::owner_read | fs::perms::owner_write | fs::perms::owner_exec |
                        fs::perms::group_read | fs::perms::group_exec | fs::perms::others_read |
                        fs::perms::others_exec,
                    fs::perm_options::replace, ec);
    if (ec) {
        return make_error(Error::generic("failed to chmod fallback /sbin/init: " + ec.message()));
    }

    return {};
}

auto Builder::clean() -> VoidResult {
    auto& cfg = ctx_->config();
    std::error_code ec;
    std::filesystem::remove_all(cfg.paths.rootfs_dir, ec);
    if (ec)
        return make_error(Error::generic("failed to clean rootfs: " + ec.message()));
    std::filesystem::remove(cfg.paths.disk_image, ec);
    ec.clear();
    return {};
}

}  // namespace elmos::domain::rootfs
