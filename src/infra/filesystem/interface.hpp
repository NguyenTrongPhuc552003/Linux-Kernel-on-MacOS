#pragma once
// ============================================================================
// filesystem/interface.hpp — Abstract file system interface
// Replaces Go's core/infra/filesystem/interface.go
// ============================================================================

#include <elmos/common.hpp>

#include <cstdint>
#include <filesystem>
#include <fstream>
#include <string>
#include <vector>

namespace elmos::infra::filesystem {

namespace fs = std::filesystem;

/// Directory entry info.
struct DirEntry {
    std::string name;
    bool is_directory;
    std::uintmax_t size;
};

/// Abstract base class for file system operations.
class FileSystem {
public:
    virtual ~FileSystem() = default;

    /// Check if a path exists.
    virtual auto exists(const fs::path& path) const -> bool = 0;

    /// Check if a path is a directory.
    virtual auto is_dir(const fs::path& path) const -> bool = 0;

    /// Read the entire contents of a file.
    virtual auto read_file(const fs::path& path) const -> Result<std::string> = 0;

    /// Write data to a file, creating it if necessary.
    virtual auto write_file(const fs::path& path, std::string_view data,
                            fs::perms perm = fs::perms::owner_read | fs::perms::owner_write |
                                             fs::perms::group_read | fs::perms::others_read)
        -> VoidResult = 0;

    /// Create a directory and all necessary parents.
    virtual auto mkdir_all(const fs::path& path,
                           fs::perms perm = fs::perms::owner_all | fs::perms::group_read |
                                            fs::perms::group_exec | fs::perms::others_read |
                                            fs::perms::others_exec) -> VoidResult = 0;

    /// Read directory entries.
    virtual auto read_dir(const fs::path& path) const -> Result<std::vector<DirEntry>> = 0;

    /// Remove a file or empty directory.
    virtual auto remove(const fs::path& path) -> VoidResult = 0;

    /// Remove a path and all children.
    virtual auto remove_all(const fs::path& path) -> VoidResult = 0;

    /// Get the current working directory.
    virtual auto getwd() const -> Result<fs::path> = 0;

    /// Get file size in bytes.
    virtual auto file_size(const fs::path& path) const -> Result<std::uintmax_t> = 0;
};

}  // namespace elmos::infra::filesystem
