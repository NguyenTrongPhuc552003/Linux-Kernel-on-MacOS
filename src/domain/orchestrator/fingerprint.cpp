// ============================================================================
// domain/orchestrator/fingerprint.cpp
// ============================================================================

#include "fingerprint.hpp"

#include <openssl/evp.h>

#include <fstream>
#include <iomanip>
#include <memory>
#include <sstream>

namespace elmos::domain::orchestrator {

auto Fingerprinter::compute_string(const std::string& content) -> std::string {
    auto ctx = std::unique_ptr<EVP_MD_CTX, decltype(&EVP_MD_CTX_free)>(EVP_MD_CTX_new(),
                                                                       EVP_MD_CTX_free);
    if (!ctx)
        return {};
    EVP_DigestInit_ex(ctx.get(), EVP_sha256(), nullptr);
    EVP_DigestUpdate(ctx.get(), content.data(), content.size());

    unsigned char hash[EVP_MAX_MD_SIZE];
    unsigned int hash_len = 0;
    EVP_DigestFinal_ex(ctx.get(), hash, &hash_len);

    std::ostringstream ss;
    for (unsigned int i = 0; i < hash_len; ++i) {
        ss << std::hex << std::setfill('0') << std::setw(2) << static_cast<int>(hash[i]);
    }
    return ss.str();
}

auto Fingerprinter::compute(const std::vector<std::string>& paths) -> Result<std::string> {
    auto ctx = std::unique_ptr<EVP_MD_CTX, decltype(&EVP_MD_CTX_free)>(EVP_MD_CTX_new(),
                                                                       EVP_MD_CTX_free);
    if (!ctx)
        return make_error(Error::generic("failed to create EVP_MD_CTX"));
    if (EVP_DigestInit_ex(ctx.get(), EVP_sha256(), nullptr) != 1)
        return make_error(Error::generic("failed to init SHA256"));

    for (const auto& path : paths) {
        std::ifstream file(path, std::ios::binary);
        if (!file) {
            return make_error(Error::generic("cannot open file for fingerprinting: " + path));
        }
        char buf[8192];
        while (file.read(buf, sizeof(buf)) || file.gcount() > 0) {
            if (EVP_DigestUpdate(ctx.get(), buf, static_cast<size_t>(file.gcount())) != 1)
                return make_error(Error::generic("failed to update SHA256"));
        }
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

}  // namespace elmos::domain::orchestrator
