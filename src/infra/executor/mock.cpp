// ============================================================================
// executor/mock.cpp — MockExecutor implementation for testing
// ============================================================================

#include "mock.hpp"

namespace elmos::infra::executor {

auto MockExecutor::run(std::stop_token /*token*/, const std::string& cmd,
                       const std::vector<std::string>& args) -> VoidResult {
    calls.push_back({.cmd = cmd, .args = args});
    if (run_error)
        return make_error(*run_error);
    return {};
}

auto MockExecutor::run_with_env(std::stop_token /*token*/, const EnvList& env,
                                const std::string& cmd, const std::vector<std::string>& args)
    -> VoidResult {
    calls.push_back({.cmd = cmd, .args = args, .env = env});
    if (run_error)
        return make_error(*run_error);
    return {};
}

auto MockExecutor::run_in_dir(std::stop_token /*token*/, const std::string& dir,
                              const std::string& cmd, const std::vector<std::string>& args)
    -> VoidResult {
    calls.push_back({.cmd = cmd, .args = args, .dir = dir});
    if (run_error)
        return make_error(*run_error);
    return {};
}

auto MockExecutor::run_with_env_in_dir(std::stop_token /*token*/, const EnvList& env,
                                       const std::string& dir, const std::string& cmd,
                                       const std::vector<std::string>& args) -> VoidResult {
    calls.push_back({.cmd = cmd, .args = args, .env = env, .dir = dir});
    if (run_error)
        return make_error(*run_error);
    return {};
}

auto MockExecutor::output(std::stop_token token, const std::string& cmd,
                          const std::vector<std::string>& args) -> Result<std::string> {
    return output_with_env(token, {}, cmd, args);
}

auto MockExecutor::output_with_env(std::stop_token /*token*/, const EnvList& env,
                                   const std::string& cmd, const std::vector<std::string>& args)
    -> Result<std::string> {
    calls.push_back({.cmd = cmd, .args = args, .env = env});

    if (auto it = output_errors.find(cmd); it != output_errors.end()) {
        return make_error(it->second);
    }
    if (auto it = output_responses.find(cmd); it != output_responses.end()) {
        return it->second;
    }
    return std::string{};
}

auto MockExecutor::run_silent(std::stop_token /*token*/, const EnvList& env, const std::string& cmd,
                              const std::vector<std::string>& args) -> VoidResult {
    calls.push_back({.cmd = cmd, .args = args, .env = env});
    if (run_error)
        return make_error(*run_error);
    return {};
}

auto MockExecutor::look_path(const std::string& cmd) -> Result<std::string> {
    if (auto it = look_path_errors.find(cmd); it != look_path_errors.end()) {
        return make_error(it->second);
    }
    if (auto it = look_path_responses.find(cmd); it != look_path_responses.end()) {
        return it->second;
    }
    return make_error(Error(ErrorCode::Dependency, "executable not found in PATH: " + cmd));
}

auto MockExecutor::exec_replace(const std::string& cmd, const std::vector<std::string>& args,
                                const EnvList& env) -> VoidResult {
    exec_called = true;
    calls.push_back({.cmd = cmd, .args = args, .env = env});
    if (exec_error)
        return make_error(*exec_error);
    return {};
}

void MockExecutor::reset() {
    calls.clear();
    exec_called = false;
}

}  // namespace elmos::infra::executor
