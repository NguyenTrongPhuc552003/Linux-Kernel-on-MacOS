#pragma once
// ============================================================================
// domain/bsp/firmware.hpp — Firmware blob management
// ============================================================================

#include <elmos/common.hpp>

#include <string>
#include <unordered_map>
#include <vector>

namespace elmos::domain::bsp {

struct BlobSpec {
    std::string soc;
    std::string name;
    std::string url;
    std::string sha256;
};

/// Manages firmware blob downloads and cache under <workspace>/bsp-cache/firmware/
class FirmwareManager {
public:
    explicit FirmwareManager(const std::string& cache_dir);

    auto get_blob_path(const std::string& soc, const std::string& blob_name) -> std::string;
    auto is_cached(const std::string& soc, const std::string& blob_name) -> bool;
    auto ensure_blob(const std::string& soc, const std::string& blob_name, const std::string& url,
                     const std::string& checksum) -> Result<std::string>;
    auto validate_checksum(const std::string& path, const std::string& expected) -> VoidResult;
    auto compute_checksum(const std::string& path) -> Result<std::string>;
    auto clean_soc_cache(const std::string& soc) -> VoidResult;
    auto list_cached_blobs(const std::string& soc) -> Result<std::vector<std::string>>;

private:
    std::string cache_dir_;
    auto download_blob(const std::string& url, const std::string& dest) -> VoidResult;
};

auto get_blobs_for_soc(const std::string& soc) -> std::vector<BlobSpec>;
auto get_blob_by_name(const std::string& soc, const std::string& name) -> const BlobSpec*;

}  // namespace elmos::domain::bsp
