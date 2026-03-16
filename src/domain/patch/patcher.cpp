// ============================================================================
// domain/patch/patcher.cpp — Patch application
// ============================================================================

#include "patcher.hpp"

#include <algorithm>
#include <filesystem>

namespace elmos::domain::patch {

namespace fs = std::filesystem;

Patcher::Patcher(infra::executor::Executor* exec, infra::filesystem::FileSystem* fs_arg)
    : exec_(exec), fs_(fs_arg) {}

auto Patcher::apply(std::stop_token token, const std::string& target_dir,
                    const std::string& patch_dir) -> VoidResult {
    auto patches = list_patches(patch_dir);
    if (!patches)
        return make_error(patches.error());

    for (const auto& p : *patches) {
        if (check_applied(token, target_dir, p.path))
            continue;
        auto r = apply_single(token, target_dir, p.path);
        if (!r)
            return r;
    }
    return {};
}

auto Patcher::apply_single(std::stop_token token, const std::string& target_dir,
                           const std::string& patch_file) -> VoidResult {
    return exec_->run_in_dir(token, target_dir, "git", {"apply", patch_file});
}

auto Patcher::list_patches(const std::string& patch_dir) -> Result<std::vector<PatchInfo>> {
    if (!fs_->exists(patch_dir))
        return std::vector<PatchInfo>{};

    // Recursively scan patch_dir for .patch and .diff files
    // Supports nested structure: version/arch/patch_file
    std::vector<PatchInfo> patches;
    for (const auto& entry : fs::recursive_directory_iterator(patch_dir)) {
        if (entry.is_directory())
            continue;
        auto name = entry.path().filename().string();
        if (name.ends_with(".patch") || name.ends_with(".diff")) {
            auto rel = fs::relative(entry.path(), patch_dir).string();
            patches.push_back({
                .name = rel,
                .path = entry.path().string(),
            });
        }
    }

    std::sort(patches.begin(), patches.end(),
              [](const auto& a, const auto& b) { return a.name < b.name; });
    return patches;
}

auto Patcher::check_applied(std::stop_token token, const std::string& target_dir,
                            const std::string& patch_file) -> bool {
    auto r = exec_->run_silent(token, {}, "git",
                               {"-C", target_dir, "apply", "--check", "--reverse", patch_file});
    return r.has_value();
}

}  // namespace elmos::domain::patch
