#pragma once
// ============================================================================
// filesystem/os_filesystem.hpp — Real filesystem using std::filesystem
// Replaces Go's core/infra/filesystem/os.go
// ============================================================================

#include "interface.hpp"

namespace elmos::infra::filesystem {

/// OSFileSystem implements FileSystem using std::filesystem and standard I/O.
class OSFileSystem final : public FileSystem {
public:
    OSFileSystem() = default;

    auto exists(const fs::path& path) const -> bool override;
    auto is_dir(const fs::path& path) const -> bool override;
    auto read_file(const fs::path& path) const -> Result<std::string> override;
    auto write_file(const fs::path& path, std::string_view data, fs::perms perm)
        -> VoidResult override;
    auto mkdir_all(const fs::path& path, fs::perms perm) -> VoidResult override;
    auto read_dir(const fs::path& path) const -> Result<std::vector<DirEntry>> override;
    auto remove(const fs::path& path) -> VoidResult override;
    auto remove_all(const fs::path& path) -> VoidResult override;
    auto getwd() const -> Result<fs::path> override;
    auto file_size(const fs::path& path) const -> Result<std::uintmax_t> override;
};

}  // namespace elmos::infra::filesystem
