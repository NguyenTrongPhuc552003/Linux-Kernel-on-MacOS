// ============================================================================
// executor/shell.cpp — ShellExecutor implementation using POSIX APIs
// ============================================================================

#include "shell.hpp"

#include "env.hpp"

#include <array>
#include <cerrno>
#include <cstdlib>
#include <cstring>
#include <filesystem>
#include <sstream>

#include <sys/wait.h>
#include <unistd.h>

// macOS does not provide execvpe (GNU extension).
// Emulate it by replacing environ and calling execvp.
// Safe: only called post-fork (child) or in exec_replace (replaces process).
#ifdef __APPLE__
extern "C" {
extern char** environ;
}
static int execvpe(const char* file, char* const argv[], char* const envp[]) {
    environ = const_cast<char**>(envp);
    return execvp(file, argv);
}
#endif

namespace elmos::infra::executor {

auto ShellExecutor::run(std::stop_token token, const std::string& cmd,
                        const std::vector<std::string>& args) -> VoidResult {
    return run_with_env_in_dir(token, {}, "", cmd, args);
}

auto ShellExecutor::run_with_env(std::stop_token token, const EnvList& env, const std::string& cmd,
                                 const std::vector<std::string>& args) -> VoidResult {
    return run_with_env_in_dir(token, env, "", cmd, args);
}

auto ShellExecutor::run_in_dir(std::stop_token token, const std::string& dir,
                               const std::string& cmd, const std::vector<std::string>& args)
    -> VoidResult {
    return run_with_env_in_dir(token, {}, dir, cmd, args);
}

auto ShellExecutor::run_with_env_in_dir(std::stop_token token, const EnvList& env,
                                        const std::string& dir, const std::string& cmd,
                                        const std::vector<std::string>& args) -> VoidResult {
    auto result = execute(token, cmd, args, {.env = env, .dir = dir});
    if (!result)
        return std::unexpected(result.error());
    return {};
}

auto ShellExecutor::output(std::stop_token token, const std::string& cmd,
                           const std::vector<std::string>& args) -> Result<std::string> {
    return output_with_env(token, {}, cmd, args);
}

auto ShellExecutor::output_with_env(std::stop_token token, const EnvList& env,
                                    const std::string& cmd, const std::vector<std::string>& args)
    -> Result<std::string> {
    return execute(token, cmd, args, {.env = env, .capture_stdout = true});
}

auto ShellExecutor::run_silent(std::stop_token token, const EnvList& env, const std::string& cmd,
                               const std::vector<std::string>& args) -> VoidResult {
    auto result = execute(token, cmd, args, {.env = env, .suppress_stderr = true});
    if (!result) {
        return make_error(result.error());
    }
    return {};
}

auto ShellExecutor::look_path(const std::string& cmd) -> std::optional<std::string> {
    if (cmd.empty()) {
        return std::nullopt;
    }

    if (cmd.find('/') != std::string::npos) {
        std::error_code ec;
        if (std::filesystem::exists(cmd, ec)) {
            return cmd;
        }
        return std::nullopt;
    }

    const char* path_env = std::getenv("PATH");
    if (!path_env) {
        return std::nullopt;
    }

    std::stringstream ss(path_env);
    std::string dir;
    while (std::getline(ss, dir, ':')) {
        if (dir.empty()) {
            continue;
        }
        auto candidate = std::filesystem::path(dir) / cmd;
        if (::access(candidate.c_str(), X_OK) == 0) {
            return candidate.string();
        }
    }

    return std::nullopt;
}

auto ShellExecutor::exec_replace(const std::string& cmd, const std::vector<std::string>& args,
                                 const EnvList& env) -> VoidResult {
    std::vector<std::string> argv_storage;
    argv_storage.reserve(args.size() + 1);
    argv_storage.push_back(cmd);
    argv_storage.insert(argv_storage.end(), args.begin(), args.end());

    std::vector<char*> argv;
    argv.reserve(argv_storage.size() + 1);
    for (auto& token : argv_storage) {
        argv.push_back(token.data());
    }
    argv.push_back(nullptr);

    if (!env.empty()) {
        auto merged = merge_env(get_current_env_list(), env);
        std::vector<char*> envp;
        envp.reserve(merged.size() + 1);
        for (auto& entry : merged) {
            envp.push_back(entry.data());
        }
        envp.push_back(nullptr);

        ::execvpe(cmd.c_str(), argv.data(), envp.data());
    }
    else {
        ::execvp(cmd.c_str(), argv.data());
    }

    return make_error(Error(ErrorCode::Build,
                            "exec failed for '" + cmd + "': " + std::string(std::strerror(errno))));
}

auto ShellExecutor::execute(std::stop_token /*token*/, const std::string& cmd,
                            const std::vector<std::string>& args, const ExecOptions& opts)
    -> Result<std::string> {
    std::array<int, 2> stdout_pipe = {-1, -1};
    if (opts.capture_stdout && ::pipe(stdout_pipe.data()) == -1) {
        return make_error(
            Error(ErrorCode::Build, "pipe failed: " + std::string(std::strerror(errno))));
    }

    std::array<int, 2> stderr_pipe = {-1, -1};
    if (opts.suppress_stderr && ::pipe(stderr_pipe.data()) == -1) {
        if (opts.capture_stdout) {
            ::close(stdout_pipe[0]);
            ::close(stdout_pipe[1]);
        }
        return make_error(
            Error(ErrorCode::Build, "pipe failed: " + std::string(std::strerror(errno))));
    }

    pid_t pid = ::fork();
    if (pid == -1) {
        if (opts.capture_stdout) {
            ::close(stdout_pipe[0]);
            ::close(stdout_pipe[1]);
        }
        if (opts.suppress_stderr) {
            ::close(stderr_pipe[0]);
            ::close(stderr_pipe[1]);
        }
        return make_error(
            Error(ErrorCode::Build, "fork failed: " + std::string(std::strerror(errno))));
    }

    if (pid == 0) {
        if (opts.capture_stdout) {
            ::close(stdout_pipe[0]);
            ::dup2(stdout_pipe[1], STDOUT_FILENO);
            ::close(stdout_pipe[1]);
        }

        if (opts.suppress_stderr) {
            ::close(stderr_pipe[0]);
            ::dup2(stderr_pipe[1], STDERR_FILENO);
            ::close(stderr_pipe[1]);
        }

        if (!opts.dir.empty() && ::chdir(opts.dir.c_str()) == -1) {
            _exit(127);
        }

        std::vector<std::string> argv_storage;
        argv_storage.reserve(args.size() + 1);
        argv_storage.push_back(cmd);
        argv_storage.insert(argv_storage.end(), args.begin(), args.end());

        std::vector<char*> argv;
        argv.reserve(argv_storage.size() + 1);
        for (auto& token : argv_storage) {
            argv.push_back(token.data());
        }
        argv.push_back(nullptr);

        if (!opts.env.empty()) {
            auto merged = merge_env(get_current_env_list(), opts.env);
            std::vector<char*> envp;
            envp.reserve(merged.size() + 1);
            for (auto& entry : merged) {
                envp.push_back(entry.data());
            }
            envp.push_back(nullptr);
            ::execvpe(cmd.c_str(), argv.data(), envp.data());
        }
        else {
            ::execvp(cmd.c_str(), argv.data());
        }

        _exit(127);
    }

    std::string captured;
    if (opts.capture_stdout) {
        ::close(stdout_pipe[1]);
        std::array<char, 4096> buf{};
        ssize_t n = 0;
        while ((n = ::read(stdout_pipe[0], buf.data(), buf.size())) > 0) {
            captured.append(buf.data(), static_cast<size_t>(n));
        }
        ::close(stdout_pipe[0]);
    }

    if (opts.suppress_stderr) {
        ::close(stderr_pipe[1]);
        std::array<char, 4096> buf{};
        while (::read(stderr_pipe[0], buf.data(), buf.size()) > 0) {}
        ::close(stderr_pipe[0]);
    }

    int status = 0;
    while (::waitpid(pid, &status, 0) == -1) {
        if (errno != EINTR) {
            return make_error(
                Error(ErrorCode::Build, "waitpid failed: " + std::string(std::strerror(errno))));
        }
    }

    if (WIFEXITED(status) && WEXITSTATUS(status) != 0) {
        return make_error(Error(ErrorCode::Build,
                                cmd + " exited with code " + std::to_string(WEXITSTATUS(status))));
    }

    if (WIFSIGNALED(status)) {
        return make_error(
            Error(ErrorCode::Build, cmd + " killed by signal " + std::to_string(WTERMSIG(status))));
    }

    return captured;
}

}  // namespace elmos::infra::executor
