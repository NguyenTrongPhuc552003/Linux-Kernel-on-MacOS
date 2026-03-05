#pragma once
// ============================================================================
// ui/printer.hpp — Console output with colored formatting
// ============================================================================

#include <cstdio>
#include <format>
#include <string>
#include <string_view>

namespace elmos::ui {

/// ANSI color codes for terminal output.
namespace color {
constexpr std::string_view kReset = "\033[0m";
constexpr std::string_view kRed = "\033[31m";
constexpr std::string_view kGreen = "\033[32m";
constexpr std::string_view kYellow = "\033[33m";
constexpr std::string_view kBlue = "\033[34m";
constexpr std::string_view kMagenta = "\033[35m";
constexpr std::string_view kCyan = "\033[36m";
constexpr std::string_view kWhite = "\033[37m";
constexpr std::string_view kBold = "\033[1m";
}  // namespace color

/// Printer provides formatted console output (success, error, warn, info, step).
class Printer {
public:
    template <typename... Args>
    void success(std::format_string<Args...> fmt, Args&&... args) const {
        auto msg = std::format(fmt, std::forward<Args>(args)...);
        std::printf("%s%s✓ %s%s\n", color::kBold.data(), color::kGreen.data(), msg.c_str(),
                    color::kReset.data());
    }

    template <typename... Args>
    void error(std::format_string<Args...> fmt, Args&&... args) const {
        auto msg = std::format(fmt, std::forward<Args>(args)...);
        std::fprintf(stderr, "%s%s✗ %s%s\n", color::kBold.data(), color::kRed.data(), msg.c_str(),
                     color::kReset.data());
    }

    template <typename... Args>
    void warn(std::format_string<Args...> fmt, Args&&... args) const {
        auto msg = std::format(fmt, std::forward<Args>(args)...);
        std::printf("%s%s⚠ %s%s\n", color::kBold.data(), color::kYellow.data(), msg.c_str(),
                    color::kReset.data());
    }

    template <typename... Args>
    void info(std::format_string<Args...> fmt, Args&&... args) const {
        auto msg = std::format(fmt, std::forward<Args>(args)...);
        std::printf("%s%sℹ %s%s\n", color::kBold.data(), color::kBlue.data(), msg.c_str(),
                    color::kReset.data());
    }

    template <typename... Args>
    void step(std::format_string<Args...> fmt, Args&&... args) const {
        auto msg = std::format(fmt, std::forward<Args>(args)...);
        std::printf("%s%s→ %s%s\n", color::kBold.data(), color::kMagenta.data(), msg.c_str(),
                    color::kReset.data());
    }

    template <typename... Args>
    void print(std::format_string<Args...> fmt, Args&&... args) const {
        auto msg = std::format(fmt, std::forward<Args>(args)...);
        std::printf("%s\n", msg.c_str());
    }
};

/// Global banner string.
inline auto banner() -> std::string {
    return std::format("{}{}{}{}", color::kBold, color::kMagenta,
                       R"(
 ███████╗██╗     ███╗   ███╗ ██████╗ ███████╗
 ██╔════╝██║     ████╗ ████║██╔═══██╗██╔════╝
 █████╗  ██║     ██╔████╔██║██║   ██║███████╗
 ██╔══╝  ██║     ██║╚██╔╝██║██║   ██║╚════██║
 ███████╗███████╗██║ ╚═╝ ██║╚██████╔╝███████║
 ╚══════╝╚══════╝╚═╝     ╚═╝ ╚═════╝ ╚══════╝)",
                       color::kReset);
}

}  // namespace elmos::ui
