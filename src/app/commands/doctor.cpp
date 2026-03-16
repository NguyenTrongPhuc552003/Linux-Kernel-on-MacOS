// ============================================================================
// app/commands/doctor.cpp — Environment health check command
// ============================================================================

#include <app/app.hpp>

#include "commands.hpp"

#include <map>
#include <stop_token>
#include <vector>

namespace elmos::app::commands {

namespace {

// Ordered section display for doctor output.
const std::vector<std::string>& section_order() {
    static const std::vector<std::string> sections = {
        "System",          "Build Tools", "Headers", "Virtualization", "Toolchain Dependencies",
        "Cross-compilers", "Toolchains",
    };
    return sections;
}

}  // namespace

void register_doctor(App& app, CLI::App& cli) {
    auto* doctor = cli.add_subcommand("doctor", "Check environment health");
    auto* fix_flag = doctor->add_flag("-f,--fix", "Auto-install missing dependencies");

    doctor->callback([&app, fix_flag] {
        app.printer().step("Running environment checks...");
        std::stop_source ss;
        auto token = ss.get_token();
        auto [checks, issue_count] = app.health_checker().check_all(token);

        // Group results by category.
        std::map<std::string, std::vector<domain::doctor::CheckResult*>> by_section;
        for (auto& r : checks)
            by_section[r.category].push_back(&r);

        bool all_pass = true;
        for (const auto& section : section_order()) {
            auto it = by_section.find(section);
            if (it == by_section.end() || it->second.empty())
                continue;

            app.printer().print("");
            app.printer().step("── {} ──", section);
            for (auto* r : it->second) {
                std::string tag = r->required ? "[required]" : "[optional]";
                if (r->passed) {
                    app.printer().success("{} {}: {}", tag, r->name, r->message);
                }
                else {
                    all_pass = false;
                    app.printer().error("{} {}: {}", tag, r->name, r->message);
                    if (!r->install_hint.empty()) {
                        app.printer().info("  Fix: {}", r->install_hint);
                    }
                }
            }
        }

        // Print any categories not in the predefined order.
        for (auto& [cat, items] : by_section) {
            bool found = false;
            for (const auto& s : section_order()) {
                if (s == cat) {
                    found = true;
                    break;
                }
            }
            if (found)
                continue;
            app.printer().print("");
            app.printer().step("── {} ──", cat);
            for (auto* r : items) {
                std::string tag = r->required ? "[required]" : "[optional]";
                if (r->passed) {
                    app.printer().success("{} {}: {}", tag, r->name, r->message);
                }
                else {
                    all_pass = false;
                    app.printer().error("{} {}: {}", tag, r->name, r->message);
                    if (!r->install_hint.empty()) {
                        app.printer().info("  Fix: {}", r->install_hint);
                    }
                }
            }
        }

        app.printer().print("");

        // Auto-fix if --fix flag is set.
        if (fix_flag->count() > 0 && !all_pass) {
            app.printer().step("Attempting to fix missing dependencies...");

            // Fix packages.
            auto fixed_pkgs = app.health_checker().fix_packages(token, checks);
            for (auto& pkg : fixed_pkgs) {
                app.printer().success("Installed: {}", pkg);
            }

            // Fix headers.
            auto fixed_hdrs = app.health_checker().fix_headers(token);
            for (auto& hdr : fixed_hdrs) {
                app.printer().success("Installed header: {}", hdr);
            }

            if (fixed_pkgs.empty() && fixed_hdrs.empty()) {
                app.printer().info("No automatic fixes available for remaining issues.");
            }
            else {
                app.printer().success("Fixed {} package(s) and {} header(s).", fixed_pkgs.size(),
                                      fixed_hdrs.size());
            }
        }
        else if (all_pass) {
            app.printer().success("All checks passed!");
        }
        else {
            app.printer().warn("Some checks failed. Run 'elmos doctor --fix' to auto-install.");
        }
    });
}

}  // namespace elmos::app::commands
