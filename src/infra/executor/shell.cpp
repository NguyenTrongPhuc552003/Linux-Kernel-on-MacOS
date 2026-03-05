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
    if (!result)
        return std::unexpected(result.error());
    return {};
}

auto ShellExecutor::look_path(const std::string& cmd) -> Result<std::string> {
    // If cmd contains '/', it's an explicit path
    if (cmd.find('/') != std::string::npos) {
        if (access(cmd.c_str(), X_OK) == 0)
            return cmd;
        return make_error(Error(ErrorCode::Dependency, "executable not found: " + cmd));
    }

    const char* path_env = std::getenv("PATH");
    if (path_env == nullptr) {
        return make_error(Error(ErrorCode::Dependency, "PATH not set"));
    }

    std::istringstream paths(path_env);
    std::string dir;
    while (std::getline(paths, dir, ':')) {
        auto full = std::filesystem::path(dir) / cmd;
        if (access(full.c_str(), X_OK) == 0) {
            return full.string();
        }
    }

    return make_error(Error(ErrorCode::Dependency, "executable not found in PATH: " + cmd));
}

auto ShellExecutor::exec_replace(const std::string& cmd, const std::vector<std::string>& args,
                                 const EnvList& env) -> VoidResult {
    // Build argv array for execvp
    std::vector<const char*> argv;
    argv.push_back(cmd.c_str());
    for (const auto& arg : args) {
        argv.push_back(arg.c_str());
    }
    argv.push_back(nullptr);

    // Build envp array
    auto merged = merge_env(get_current_env_list(), env);
    std::vector<const char*> envp;
    for (const auto& e : merged) {
        envp.push_back(e.c_str());
    }
    envp.push_back(nullptr);

    execvpe(cmd.c_str(), const_cast<char* const*>(argv.data()),
            const_cast<char* const*>(envp.data()));

    // execvpe only returns on failure
    return make_error(Error(ErrorCode::Build, "exec failed: " + cmd + ": " + std::strerror(errno)));
}

auto ShellExecutor::execute(std::stop_token token, const std::string& cmd,
                            const std::vector<std::string>& args, const ExecOptions& opts)
    -> Result<std::string> {
    // Set up pipe for capturing stdout if needed
    std::array<int, 2> stdout_pipe = {-1, -1};
    if (opts.capture_stdout) {
        if (pipe(stdout_pipe.data()) == -1) {
            return make_error(
                Error(ErrorCode::Build, "pipe failed: " + std::string(std::strerror(errno))));
        }
    }

    // Set up pipe for suppressing stderr if needed
    std::array<int, 2> stderr_pipe = {-1, -1};
    if (opts.suppress_stderr) {
        if (pipe(stderr_pipe.data()) == -1) {
            if (opts.capture_stdout) {
                close(stdout_pipe[0]);
                close(stdout_pipe[1]);
            }
            return make_error(
                Error(ErrorCode::Build, "pipe failed: " + std::string(std::strerror(errno))));
        }
    }

    pid_t pid = fork();
    if (pid == -1) {
        if (opts.capture_stdout) {
            close(stdout_pipe[0]);
            close(stdout_pipe[1]);
        }
        if (opts.suppress_stderr) {
            close(stderr_pipe[0]);
            close(stderr_pipe[1]);
        }
        return make_error(
            Error(ErrorCode::Build, "fork failed: " + std::string(std::strerror(errno))));
    }

    if (pid == 0) {
        // Child process
        if (opts.capture_stdout) {
            close(stdout_pipe[0]);
            dup2(stdout_pipe[1], STDOUT_FILENO);
            close(stdout_pipe[1]);
        }

        if (opts.suppress_stderr) {
            close(stderr_pipe[0]);
            dup2(stderr_pipe[1], STDERR_FILENO);
            close(stderr_pipe[1]);
        }

        if (!opts.dir.empty()) {
            if (chdir(opts.dir.c_str()) == -1) {
                _exit(127);
            }
        }

        // Build argv
        std::vector<const char*> argv;
        argv.push_back(cmd.c_str());
        for (const auto& arg : args) {
            argv.push_back(arg.c_str());
        }
        argv.push_back(nullptr);

        // Set environment if specified
        if (!opts.env.empty()) {
            auto merged = merge_env(get_current_env_list(), opts.env);
            std::vector<const char*> envp;
            for (const auto& e : merged) {
                envp.push_back(e.c_str());
            }
            envp.push_back(nullptr);
            execvpe(cmd.c_str(), const_cast<char* const*>(argv.data()),
                    const_cast<char* const*>(envp.data()));
        }
        else {
            execvp(cmd.c_str(), const_cast<char* const*>(argv.data()));
        }

        _exit(127);  // exec failed
    }

    // Parent process
    std::string captured;
    if (opts.capture_stdout) {
        close(stdout_pipe[1]);
        std::array<char, 4096> buf{};
        ssize_t n = 0;
        while ((n = read(stdout_pipe[0], buf.data(), buf.size())) > 0) {
            captured.append(buf.data(), static_cast<size_t>(n));
        }
        close(stdout_pipe[0]);
    }

    if (opts.suppress_stderr) {
        close(stderr_pipe[1]);
        // Drain stderr pipe to avoid blocking the child
        std::array<char, 4096> buf{};
        while (read(stderr_pipe[0], buf.data(), buf.size()) > 0) {}
        close(stderr_pipe[0]);
    }

    int status = 0;
    while (true) {
        pid_t result = waitpid(pid, &status, 0);
        if (result == -1) {
            if (errno == EINTR)
                continue;
            return make_error(
                Error(ErrorCode::Build, "waitpid failed: " + std::string(std::strerror(errno))));
        }
        break;
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
