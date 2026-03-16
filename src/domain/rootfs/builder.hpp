#pragma once
// ============================================================================
// domain/rootfs/builder.hpp — Root filesystem creation
// ============================================================================

#include <elmos/common.hpp>

#include <stop_token>
#include <string>
#include <vector>

namespace elmos::context {
class Context;
}

namespace elmos::domain::rootfs {

struct RootfsOptions {
    std::string distribution = "debian";
    std::string release = "bookworm";
    std::vector<std::string> packages;
    std::string post_build_script;
    bool assemble_disk_image = false;
    std::string disk_size;
    bool force_disk_image = false;
    /// When true, produce a two-partition disk image:
    ///   Partition 1 — FAT32 boot partition (128 MiB) containing the Linux
    ///                  kernel image + extlinux/extlinux.conf for U-Boot.
    ///   Partition 2 — ext4 rootfs partition (all remaining space).
    /// Required when booting via an external U-Boot bootloader.
    bool with_boot_partition = false;
};

class Builder {
public:
    explicit Builder(context::Context* ctx);

    auto create(std::stop_token token, RootfsOptions opts) -> VoidResult;
    auto ensure_disk_image(std::stop_token token, const std::string& size = "", bool force = false,
                           bool with_boot_partition = false) -> Result<bool>;
    auto install_modules(std::stop_token token) -> VoidResult;
    auto customize(std::stop_token token, const std::string& script) -> VoidResult;
    auto clean() -> VoidResult;

private:
    struct BootstrapSource {
        std::string arch;
        std::string release;
        std::string mirror;
    };

    context::Context* ctx_;

    auto run_debootstrap(std::stop_token token, const RootfsOptions& opts) -> VoidResult;
    auto install_packages(std::stop_token token, const std::vector<std::string>& packages)
        -> VoidResult;
    auto disk_image_needs_refresh(std::stop_token token) const -> Result<bool>;
    auto ensure_pid1() -> VoidResult;
    auto resolve_bootstrap_source(std::stop_token token, const RootfsOptions& opts)
        -> Result<BootstrapSource>;
    auto release_supports_arch(std::stop_token token, const std::string& mirror,
                               const std::string& release, const std::string& arch) -> Result<bool>;

    /// Creates a partitioned disk image with a FAT32 boot partition (containing
    /// the Linux kernel image and extlinux.conf) and an ext4 rootfs partition.
    /// Uses mtools (mformat/mcopy) + mke2fs with an offset.
    auto create_partitioned_disk_image(std::stop_token token, const std::string& total_size_str,
                                       bool force) -> Result<bool>;
};

}  // namespace elmos::domain::rootfs
