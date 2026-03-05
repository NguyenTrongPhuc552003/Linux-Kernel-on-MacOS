// ============================================================================
// app/commands/doctor.cpp — Environment health check command
// ============================================================================

#include <app/app.hpp>

#include "commands.hpp"

#include <stop_token>

namespace elmos::app::commands {

void register_doctor(App& app, CLI::App& cli) {
    auto* doctor = cli.add_subcommand("doctor", "Check environment health");

    doctor->callback([&app] {
        app.printer().step("Running environment checks...");
        std::stop_source ss;
        auto [checks, issue_count] = app.health_checker().check_all(ss.get_token());
        bool all_pass = true;
        for (auto& r : checks) {
            if (r.passed) {
                app.printer().success("{}: {}", r.name, r.message);
            }
            else {
                app.printer().error("{}: {}", r.name, r.message);
                all_pass = false;
            }
        }
        if (all_pass) {
            app.printer().success("All checks passed!");
        }
        else {
            app.printer().warn("Some checks failed. Install missing dependencies.");
        }
    });
}

}  // namespace elmos::app::commands
