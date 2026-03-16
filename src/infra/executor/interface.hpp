#pragma once
// ============================================================================
// executor/interface.hpp — Abstract executor for shell command execution
// Replaces Go's core/infra/executor/interface.go
// ============================================================================

#include <elmos/common.hpp>

#include <stop_token>
#include <string>
#include <vector>

namespace elmos::infra::executor {

/// Abstract base class for executing shell commands.
/// Concrete implementations: ShellExecutor (real), MockExecutor (tests).
class Executor {
public:
    virtual ~Executor() = default;

    /// Execute a command, forwarding stdout/stderr to the parent process.
    virtual auto run(std::stop_token token, const std::string& cmd,
                     const std::vector<std::string>& args = {}) -> VoidResult = 0;

    /// Execute a command with custom environment variables ("KEY=VALUE").
    virtual auto run_with_env(std::stop_token token, const EnvList& env, const std::string& cmd,
                              const std::vector<std::string>& args = {}) -> VoidResult = 0;

    /// Execute a command in a specific working directory.
    virtual auto run_in_dir(std::stop_token token, const std::string& dir, const std::string& cmd,
                            const std::vector<std::string>& args = {}) -> VoidResult = 0;

    /// Execute a command with custom env in a specific directory.
    virtual auto run_with_env_in_dir(std::stop_token token, const EnvList& env,
                                     const std::string& dir, const std::string& cmd,
                                     const std::vector<std::string>& args = {}) -> VoidResult = 0;

    /// Execute a command and capture its stdout.
    virtual auto output(std::stop_token token, const std::string& cmd,
                        const std::vector<std::string>& args = {}) -> Result<std::string> = 0;

    /// Execute a command with custom env and capture stdout.
    virtual auto output_with_env(std::stop_token token, const EnvList& env, const std::string& cmd,
                                 const std::vector<std::string>& args = {})
        -> Result<std::string> = 0;

    /// Execute a command silently (suppress stderr).
    virtual auto run_silent(std::stop_token token, const EnvList& env, const std::string& cmd,
                            const std::vector<std::string>& args = {}) -> VoidResult = 0;

    /// Resolve executable location from PATH.
    virtual auto look_path(const std::string& cmd) -> std::optional<std::string> = 0;

    /// Replace current process image with command (no return on success).
    virtual auto exec_replace(const std::string& cmd, const std::vector<std::string>& args,
                              const EnvList& env = {}) -> VoidResult = 0;
};

}  // namespace elmos::infra::executor
