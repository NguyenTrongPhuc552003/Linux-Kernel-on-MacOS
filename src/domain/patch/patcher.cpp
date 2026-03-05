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
    return exec_->run_in_dir(token, target_dir, "git", {"apply", "--stat", patch_file});
}

auto Patcher::list_patches(const std::string& patch_dir) -> Result<std::vector<PatchInfo>> {
    if (!fs_->exists(patch_dir))
        return std::vector<PatchInfo>{};

    auto entries = fs_->read_dir(patch_dir);
    if (!entries)
        return make_error(entries.error());

    std::vector<PatchInfo> patches;
    for (const auto& e : *entries) {
        if (e.is_directory)
            continue;
        if (e.name.ends_with(".patch") || e.name.ends_with(".diff")) {
            patches.push_back({
                .name = e.name,
                .path = (fs::path(patch_dir) / e.name).string(),
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
