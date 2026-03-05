#pragma once
// ============================================================================
// ui/help.hpp — Custom help formatter for CLI11 commands
// ============================================================================

#include "printer.hpp"

#include <CLI/CLI.hpp>

#include <algorithm>
#include <format>
#include <map>
#include <sstream>
#include <string>
#include <vector>

namespace elmos::ui {

/// Command category for grouped help display.
inline auto command_category(std::string_view name) -> std::string {
    static const std::map<std::string_view, std::string_view> categories = {
        {"init", "Core"},         {"exit", "Core"},     {"doctor", "Core"},
        {"version", "Core"},      {"tui", "Core"},      {"status", "Core"},
        {"arch", "Core"},         {"kernel", "Build"},  {"module", "Build"},
        {"app", "Build"},         {"rootfs", "Build"},  {"patch", "Build"},
        {"bootloader", "Build"},  {"qemu", "Runtime"},  {"gdb", "Runtime"},
        {"toolchains", "Config"}, {"config", "Config"}, {"bsp", "Config"},
        {"plugins", "Config"},
    };
    if (auto it = categories.find(name); it != categories.end()) {
        return std::string(it->second);
    }
    return "Other";
}

/// Custom help formatter that renders colored, grouped help output.
class HelpFormatter : public CLI::Formatter {
public:
    auto make_help(const CLI::App* app, std::string name, CLI::AppFormatMode mode) const
        -> std::string override {

        std::ostringstream out;

        // Banner for root command
        if (app->get_parent() == nullptr) {
            out << banner() << "\n\n";
        }

        // Description
        if (!app->get_description().empty()) {
            out << std::format("{}{}{}{}\n", color::kBold, color::kMagenta, app->get_description(),
                               color::kReset);
        }

        // Usage
        out << std::format("\n{}{}USAGE{}\n", color::kBold, color::kWhite, color::kReset);
        out << std::format("  {}{}{}{}\n", color::kGreen, app->get_name(), color::kReset,
                           app->get_subcommands().empty() ? "" : " [command]");

        // Subcommands (grouped for root, flat otherwise)
        std::vector<const CLI::App*> subs = app->get_subcommands(
            [](const CLI::App* a) { return !a->get_name().empty() && !a->get_group().empty(); });
        if (!subs.empty()) {
            out << std::format("\n{}{}COMMANDS{}\n", color::kBold, color::kWhite, color::kReset);

            if (app->get_parent() == nullptr) {
                write_grouped_commands(out, subs);
            }
            else {
                for (const auto* sub : subs) {
                    write_command(out, sub);
                }
            }
        }

        // Options
        auto options = app->get_options();
        if (!options.empty()) {
            out << std::format("\n{}{}FLAGS{}\n", color::kBold, color::kWhite, color::kReset);
            for (const auto* opt : options) {
                if (opt->get_name() == "help" || opt->get_name() == "--help")
                    continue;
                out << std::format("  {}{}{}  {}\n", color::kYellow, opt->get_name(), color::kReset,
                                   opt->get_description());
            }
        }

        // Footer
        if (!subs.empty()) {
            out << std::format("\n{}Use \"{} [command] --help\" for more information.{}\n",
                               color::kCyan, app->get_name(), color::kReset);
        }

        return out.str();
    }

private:
    static void write_grouped_commands(std::ostringstream& out,
                                       const std::vector<const CLI::App*>& subs) {
        // Group by category
        std::map<std::string, std::vector<const CLI::App*>> groups;
        for (const auto* sub : subs) {
            groups[command_category(sub->get_name())].push_back(sub);
        }

        static constexpr std::array order = {"Core", "Build", "Runtime", "Config", "Other"};
        for (auto cat : order) {
            auto it = groups.find(std::string(cat));
            if (it == groups.end() || it->second.empty())
                continue;

            out << std::format("  {}─── {} ───{}\n", color::kCyan, cat, color::kReset);
            for (const auto* sub : it->second) {
                write_command(out, sub);
            }
        }
    }

    static void write_command(std::ostringstream& out, const CLI::App* sub) {
        out << std::format("  {}{:<14}{}  {}\n", color::kGreen, sub->get_name(), color::kReset,
                           sub->get_description());
    }
};

}  // namespace elmos::ui
