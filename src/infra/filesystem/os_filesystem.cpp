// ============================================================================
// filesystem/os_filesystem.cpp — OSFileSystem implementation
// ============================================================================

#include "os_filesystem.hpp"

#include <fstream>
#include <sstream>

namespace elmos::infra::filesystem {

auto OSFileSystem::exists(const fs::path& path) const -> bool {
    std::error_code ec;
    return fs::exists(path, ec);
}

auto OSFileSystem::is_dir(const fs::path& path) const -> bool {
    std::error_code ec;
    return fs::is_directory(path, ec);
}

auto OSFileSystem::read_file(const fs::path& path) const -> Result<std::string> {
    std::ifstream file(path, std::ios::in | std::ios::binary);
    if (!file) {
        return make_error(Error(ErrorCode::Config, "cannot read file: " + path.string()));
    }
    std::ostringstream ss;
    ss << file.rdbuf();
    return ss.str();
}

auto OSFileSystem::write_file(const fs::path& path, std::string_view data, fs::perms perm)
    -> VoidResult {
    // Ensure parent directory exists
    if (path.has_parent_path()) {
        std::error_code ec;
        fs::create_directories(path.parent_path(), ec);
        if (ec) {
            return make_error(
                Error(ErrorCode::Config, "cannot create parent dir: " + ec.message()));
        }
    }

    std::ofstream file(path, std::ios::out | std::ios::binary | std::ios::trunc);
    if (!file) {
        return make_error(Error(ErrorCode::Config, "cannot write file: " + path.string()));
    }
    file.write(data.data(), static_cast<std::streamsize>(data.size()));
    if (!file) {
        return make_error(Error(ErrorCode::Config, "write failed: " + path.string()));
    }
    file.close();

    std::error_code ec;
    fs::permissions(path, perm, ec);
    if (ec) {
        return make_error(Error(ErrorCode::Config, "cannot set permissions: " + ec.message()));
    }
    return {};
}

auto OSFileSystem::mkdir_all(const fs::path& path, fs::perms perm) -> VoidResult {
    std::error_code ec;
    fs::create_directories(path, ec);
    if (ec) {
        return make_error(Error(ErrorCode::Config,
                                "cannot create directory: " + path.string() + ": " + ec.message()));
    }
    fs::permissions(path, perm, ec);
    return {};
}

auto OSFileSystem::read_dir(const fs::path& path) const -> Result<std::vector<DirEntry>> {
    std::error_code ec;
    std::vector<DirEntry> entries;

    for (const auto& entry : fs::directory_iterator(path, ec)) {
        if (ec) {
            return make_error(Error(ErrorCode::Config, "cannot read directory: " + path.string() +
                                                           ": " + ec.message()));
        }
        std::uintmax_t size = 0;
        if (entry.is_regular_file()) {
            size = entry.file_size(ec);
            if (ec)
                size = 0;
        }
        entries.push_back({
            .name = entry.path().filename().string(),
            .is_directory = entry.is_directory(),
            .size = size,
        });
    }

    if (ec) {
        return make_error(Error(ErrorCode::Config,
                                "cannot read directory: " + path.string() + ": " + ec.message()));
    }

    return entries;
}

auto OSFileSystem::remove(const fs::path& path) -> VoidResult {
    std::error_code ec;
    fs::remove(path, ec);
    if (ec) {
        return make_error(
            Error(ErrorCode::Config, "cannot remove: " + path.string() + ": " + ec.message()));
    }
    return {};
}

auto OSFileSystem::remove_all(const fs::path& path) -> VoidResult {
    std::error_code ec;
    fs::remove_all(path, ec);
    if (ec) {
        return make_error(
            Error(ErrorCode::Config, "cannot remove: " + path.string() + ": " + ec.message()));
    }
    return {};
}

auto OSFileSystem::getwd() const -> Result<fs::path> {
    std::error_code ec;
    auto cwd = fs::current_path(ec);
    if (ec) {
        return make_error(
            Error(ErrorCode::Config, "cannot get working directory: " + ec.message()));
    }
    return cwd;
}

auto OSFileSystem::file_size(const fs::path& path) const -> Result<std::uintmax_t> {
    std::error_code ec;
    auto size = fs::file_size(path, ec);
    if (ec) {
        return make_error(Error(ErrorCode::Config,
                                "cannot get file size: " + path.string() + ": " + ec.message()));
    }
    return size;
}

}  // namespace elmos::infra::filesystem
