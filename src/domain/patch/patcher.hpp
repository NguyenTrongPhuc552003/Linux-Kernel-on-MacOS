#pragma once
// ============================================================================
// domain/patch/patcher.hpp — Patch application
// ============================================================================

#include <elmos/common.hpp>

#include <infra/executor/interface.hpp>
#include <infra/filesystem/interface.hpp>

#include <stop_token>
#include <string>
#include <vector>

namespace elmos::domain::patch {

struct PatchInfo {
    std::string name;
    std::string path;
    bool applied = false;
};

class Patcher {
public:
    Patcher(infra::executor::Executor* exec, infra::filesystem::FileSystem* fs);

    auto apply(std::stop_token token, const std::string& target_dir, const std::string& patch_dir)
        -> VoidResult;
    auto apply_single(std::stop_token token, const std::string& target_dir,
                      const std::string& patch_file) -> VoidResult;
    auto list_patches(const std::string& patch_dir) -> Result<std::vector<PatchInfo>>;
    auto check_applied(std::stop_token token, const std::string& target_dir,
                       const std::string& patch_file) -> bool;

private:
    infra::executor::Executor* exec_;
    infra::filesystem::FileSystem* fs_;
};

}  // namespace elmos::domain::patch
