#pragma once
// ============================================================================
// executor/mock.hpp — Test double for Executor
// Replaces Go's core/infra/executor/mock.go
// ============================================================================

#include "interface.hpp"

#include <functional>
#include <unordered_map>

namespace elmos::infra::executor {

/// Records a single command execution for test verification.
struct CommandCall {
    std::string cmd;
    std::vector<std::string> args;
    EnvList env;
    std::string dir;
};

/// MockExecutor records calls and returns configured responses.
class MockExecutor final : public Executor {
public:
    MockExecutor() = default;

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

    auto look_path(const std::string& cmd) -> std::optional<std::string> override;

    auto exec_replace(const std::string& cmd, const std::vector<std::string>& args,
                      const EnvList& env = {}) -> VoidResult override;

    /// Reset all recorded calls.
    void reset();

    // --- Test configuration ---
    std::vector<CommandCall> calls;
    std::optional<Error> run_error;
    std::unordered_map<std::string, std::string> output_responses;
    std::unordered_map<std::string, Error> output_errors;
    std::unordered_map<std::string, std::string> look_path_responses;
    std::unordered_map<std::string, Error> look_path_errors;
    bool exec_called = false;
    std::optional<Error> exec_error;
};

}  // namespace elmos::infra::executor
