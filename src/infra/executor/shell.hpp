#pragma once
// ============================================================================
// executor/shell.hpp — Real shell executor using fork/exec
// Replaces Go's core/infra/executor/shell.go
// ============================================================================

#include "interface.hpp"

namespace elmos::infra::executor {

/// ShellExecutor implements Executor using POSIX fork/execvp.
class ShellExecutor final : public Executor {
public:
    ShellExecutor() = default;

    auto run(std::stop_token token, const std::string& cmd,
             const std::vector<std::string>& args = {}) -> VoidResult override;

    auto run_with_env(std::stop_token token, const EnvList& env, const std::string& cmd,
                      const std::vector<std::string>& args = {}) -> VoidResult override;

    auto run_in_dir(std::stop_token token, const std::string& dir, const std::string& cmd,
                    const std::vector<std::string>& args = {}) -> VoidResult override;

    auto run_with_env_in_dir(std::stop_token token, const EnvList& env, const std::string& dir,
                             const std::string& cmd, const std::vector<std::string>& args = {})
        -> VoidResult override;

    auto output(std::stop_token token, const std::string& cmd,
                const std::vector<std::string>& args = {}) -> Result<std::string> override;

    auto output_with_env(std::stop_token token, const EnvList& env, const std::string& cmd,
                         const std::vector<std::string>& args = {}) -> Result<std::string> override;

    auto run_silent(std::stop_token token, const EnvList& env, const std::string& cmd,
                    const std::vector<std::string>& args = {}) -> VoidResult override;

    auto look_path(const std::string& cmd) -> Result<std::string> override;

    auto exec_replace(const std::string& cmd, const std::vector<std::string>& args,
                      const EnvList& env) -> VoidResult override;

private:
    struct ExecOptions {
        EnvList env;
        std::string dir;
        bool capture_stdout = false;
        bool suppress_stderr = false;
    };

    auto execute(std::stop_token token, const std::string& cmd,
                 const std::vector<std::string>& args, const ExecOptions& opts)
        -> Result<std::string>;
};

}  // namespace elmos::infra::executor
