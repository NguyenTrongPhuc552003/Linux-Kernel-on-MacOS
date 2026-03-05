// ============================================================================
// domain/bsp/firmware.cpp — Firmware blob management
// ============================================================================

#include "firmware.hpp"

#include <openssl/evp.h>

#include <cstdio>
#include <filesystem>
#include <fstream>
#include <iomanip>
#include <memory>
#include <sstream>

namespace elmos::domain::bsp {

namespace fs = std::filesystem;

// Known firmware blob registry
static const std::unordered_map<std::string, std::vector<BlobSpec>>& known_blobs() {
    static const std::unordered_map<std::string, std::vector<BlobSpec>> blobs = {
        {"rk3588",
         {
             {.soc = "rk3588",
              .name = "rk3588_ddr_lp4_2112MHz.bin",
              .url = "https://github.com/rockchip-linux/rkbin/raw/master/bin/rk35/"
                     "rk3588_ddr_lp4_2112MHz_v1.16.bin"},
             {.soc = "rk3588",
              .name = "rk3588_spl_loader.bin",
              .url = "https://github.com/rockchip-linux/rkbin/raw/master/bin/rk35/"
                     "rk3588_spl_loader_v1.15.113.bin"},
         }},
        {"rk3566",
         {
             {.soc = "rk3566",
              .name = "rk3566_ddr_1056MHz.bin",
              .url = "https://github.com/rockchip-linux/rkbin/raw/master/bin/rk35/"
                     "rk3566_ddr_1056MHz_v1.21.bin"},
             {.soc = "rk3566",
              .name = "rk356x_spl_loader.bin",
              .url = "https://github.com/rockchip-linux/rkbin/raw/master/bin/rk35/"
                     "rk356x_spl_loader_v1.18.112.bin"},
         }},
        {"rk3399",
         {
             {.soc = "rk3399",
              .name = "rk3399_ddr_933MHz.bin",
              .url = "https://github.com/rockchip-linux/rkbin/raw/master/bin/rk33/"
                     "rk3399_ddr_933MHz_v1.30.bin"},
             {.soc = "rk3399",
              .name = "rk3399_miniloader.bin",
              .url = "https://github.com/rockchip-linux/rkbin/raw/master/bin/rk33/"
                     "rk3399_miniloader_v1.30.bin"},
         }},
    };
    return blobs;
}

auto get_blobs_for_soc(const std::string& soc) -> std::vector<BlobSpec> {
    auto& blobs = known_blobs();
    auto it = blobs.find(soc);
    if (it == blobs.end())
        return {};
    return it->second;
}

auto get_blob_by_name(const std::string& soc, const std::string& name) -> const BlobSpec* {
    auto& blobs = known_blobs();
    auto it = blobs.find(soc);
    if (it == blobs.end())
        return nullptr;
    for (const auto& b : it->second) {
        if (b.name == name)
            return &b;
    }
    return nullptr;
}

FirmwareManager::FirmwareManager(const std::string& cache_dir)
    : cache_dir_((fs::path(cache_dir) / "firmware").string()) {}

auto FirmwareManager::get_blob_path(const std::string& soc, const std::string& blob_name)
    -> std::string {
    return (fs::path(cache_dir_) / soc / blob_name).string();
}

auto FirmwareManager::is_cached(const std::string& soc, const std::string& blob_name) -> bool {
    return fs::exists(get_blob_path(soc, blob_name));
}

auto FirmwareManager::ensure_blob(const std::string& soc, const std::string& blob_name,
                                  const std::string& url, const std::string& checksum)
    -> Result<std::string> {
    auto dest = get_blob_path(soc, blob_name);

    if (is_cached(soc, blob_name)) {
        if (checksum.empty())
            return dest;
        auto r = validate_checksum(dest, checksum);
        if (r)
            return dest;
    }

    std::error_code ec;
    fs::create_directories(fs::path(dest).parent_path(), ec);
    if (ec)
        return make_error(Error::generic("cannot create firmware cache: " + ec.message()));

    auto r = download_blob(url, dest);
    if (!r)
        return make_error(r.error());

    if (!checksum.empty()) {
        auto v = validate_checksum(dest, checksum);
        if (!v) {
            fs::remove(dest);
            return make_error(v.error());
        }
    }
    return dest;
}

auto FirmwareManager::validate_checksum(const std::string& path, const std::string& expected)
    -> VoidResult {
    auto actual = compute_checksum(path);
    if (!actual)
        return make_error(actual.error());
    if (*actual != expected) {
        return make_error(
            Error::generic("checksum mismatch: expected " + expected + ", got " + *actual));
    }
    return {};
}

auto FirmwareManager::compute_checksum(const std::string& path) -> Result<std::string> {
    std::ifstream file(path, std::ios::binary);
    if (!file)
        return make_error(Error::generic("cannot open file: " + path));

    auto ctx = std::unique_ptr<EVP_MD_CTX, decltype(&EVP_MD_CTX_free)>(EVP_MD_CTX_new(),
                                                                       EVP_MD_CTX_free);
    if (!ctx)
        return make_error(Error::generic("failed to create EVP_MD_CTX"));
    if (EVP_DigestInit_ex(ctx.get(), EVP_sha256(), nullptr) != 1)
        return make_error(Error::generic("failed to init SHA256"));

    char buf[8192];
    while (file.read(buf, sizeof(buf)) || file.gcount() > 0) {
        if (EVP_DigestUpdate(ctx.get(), buf, static_cast<size_t>(file.gcount())) != 1)
            return make_error(Error::generic("failed to update SHA256"));
    }

    unsigned char hash[EVP_MAX_MD_SIZE];
    unsigned int hash_len = 0;
    if (EVP_DigestFinal_ex(ctx.get(), hash, &hash_len) != 1)
        return make_error(Error::generic("failed to finalize SHA256"));

    std::ostringstream ss;
    for (unsigned int i = 0; i < hash_len; ++i) {
        ss << std::hex << std::setfill('0') << std::setw(2) << static_cast<int>(hash[i]);
    }
    return ss.str();
}

auto FirmwareManager::clean_soc_cache(const std::string& soc) -> VoidResult {
    std::error_code ec;
    fs::remove_all(fs::path(cache_dir_) / soc, ec);
    return {};
}

auto FirmwareManager::list_cached_blobs(const std::string& soc)
    -> Result<std::vector<std::string>> {
    auto dir = (fs::path(cache_dir_) / soc).string();
    std::error_code ec;
    if (!fs::exists(dir, ec))
        return std::vector<std::string>{};

    std::vector<std::string> blobs;
    for (const auto& entry : fs::directory_iterator(dir, ec)) {
        if (!entry.is_directory()) {
            blobs.push_back(entry.path().filename().string());
        }
    }
    return blobs;
}

auto FirmwareManager::download_blob(const std::string& /*url*/, const std::string& /*dest*/)
    -> VoidResult {
    // HTTP download using cpp-httplib would go here.
    // For now, return an error indicating manual download is needed.
    return make_error(
        Error::generic("network downloads not yet implemented — place blobs manually"));
}

}  // namespace elmos::domain::bsp
