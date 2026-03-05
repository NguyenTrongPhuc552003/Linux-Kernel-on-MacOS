#pragma once
// ============================================================================
// elmos/common.hpp — Project-wide type aliases and utilities
// ============================================================================

#include <cstdint>
#include <expected>
#include <functional>
#include <memory>
#include <optional>
#include <string>
#include <string_view>
#include <unordered_map>
#include <variant>
#include <vector>

namespace elmos {

// ── Error infrastructure ────────────────────────────────────────────────────

/// Error codes matching Go's context/errors.go ErrCode enum.
enum class ErrorCode : int {
    Generic = 0,
    Config,
    Image,
    Repo,
    Build,
    QEMU,
    Module,
    Dependency,
    Permission,
    App,
    Rootfs,
    Plugin,
    Platform,
    Network,
    PathTraversal,
};

/// Core error type — replaces Go's error interface + fmt.Errorf("%w", err).
/// Supports error chaining via an optional inner cause.
class Error {
public:
    Error(ErrorCode code, std::string message) : code_(code), message_(std::move(message)) {}

    Error(ErrorCode code, std::string message, Error cause)
        : code_(code), message_(std::move(message)),
          cause_(std::make_shared<Error>(std::move(cause))) {}

    [[nodiscard]] auto code() const noexcept -> ErrorCode { return code_; }
    [[nodiscard]] auto message() const noexcept -> const std::string& { return message_; }
    [[nodiscard]] auto cause() const noexcept -> const Error* {
        return cause_ ? cause_.get() : nullptr;
    }

    /// Full error chain as string: "outer: inner: root"
    [[nodiscard]] auto what() const -> std::string {
        if (cause_) {
            return message_ + ": " + cause_->what();
        }
        return message_;
    }

    /// Check if this error or any in its chain has the given code.
    /// Equivalent to Go's errors.Is().
    [[nodiscard]] auto is(ErrorCode c) const noexcept -> bool {
        if (code_ == c)
            return true;
        if (cause_)
            return cause_->is(c);
        return false;
    }

    // Named constructors for common error categories
    static auto generic(std::string msg) -> Error { return {ErrorCode::Generic, std::move(msg)}; }
    static auto config(std::string msg) -> Error { return {ErrorCode::Config, std::move(msg)}; }
    static auto image(std::string msg) -> Error { return {ErrorCode::Image, std::move(msg)}; }
    static auto repo(std::string msg) -> Error { return {ErrorCode::Repo, std::move(msg)}; }
    static auto build(std::string msg) -> Error { return {ErrorCode::Build, std::move(msg)}; }
    static auto qemu(std::string msg) -> Error { return {ErrorCode::QEMU, std::move(msg)}; }
    static auto module(std::string msg) -> Error { return {ErrorCode::Module, std::move(msg)}; }
    static auto dependency(std::string msg) -> Error {
        return {ErrorCode::Dependency, std::move(msg)};
    }
    static auto permission(std::string msg) -> Error {
        return {ErrorCode::Permission, std::move(msg)};
    }
    static auto app(std::string msg) -> Error { return {ErrorCode::App, std::move(msg)}; }
    static auto rootfs(std::string msg) -> Error { return {ErrorCode::Rootfs, std::move(msg)}; }
    static auto plugin(std::string msg) -> Error { return {ErrorCode::Plugin, std::move(msg)}; }
    static auto platform(std::string msg) -> Error { return {ErrorCode::Platform, std::move(msg)}; }
    static auto network(std::string msg) -> Error { return {ErrorCode::Network, std::move(msg)}; }
    static auto path_traversal(std::string msg) -> Error {
        return {ErrorCode::PathTraversal, std::move(msg)};
    }

    /// Wrap an existing error with additional context.
    /// Equivalent to Go's fmt.Errorf("context: %w", err).
    static auto wrap(std::string context, Error inner) -> Error {
        return {inner.code(), std::move(context), std::move(inner)};
    }

private:
    ErrorCode code_;
    std::string message_;
    std::shared_ptr<Error> cause_;  // shared_ptr enables copyability
};

// ── Result type ─────────────────────────────────────────────────────────────

/// Result<T> is the primary error-handling mechanism, replacing Go's (T, error).
/// Uses C++23 std::expected — the value path is the common case.
template <typename T>
using Result = std::expected<T, Error>;

/// Shorthand for Result<void> (operations that can only fail).
using VoidResult = Result<void>;

/// Helper to create an unexpected error (shorthand for std::unexpected).
inline auto make_error(Error err) -> std::unexpected<Error> {
    return std::unexpected(std::move(err));
}

// ── Common type aliases ─────────────────────────────────────────────────────

/// String-keyed map with std::any values (replaces Go map[string]interface{}).
using AnyMap = std::unordered_map<std::string, std::string>;

/// String-keyed string map (common config pattern).
using StringMap = std::unordered_map<std::string, std::string>;

/// Environment variable list (replaces Go []string of "KEY=VALUE").
using EnvList = std::vector<std::string>;

}  // namespace elmos
