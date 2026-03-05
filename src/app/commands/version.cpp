// ============================================================================
// app/commands/version.cpp — Version command
// ============================================================================

#include <app/app.hpp>

#include "commands.hpp"

#ifndef ELMOS_VERSION
#define ELMOS_VERSION "dev"
#endif

#ifndef ELMOS_COMMIT
#define ELMOS_COMMIT "unknown"
#endif

namespace elmos::app::commands {

void register_version(App& app, CLI::App& cli) {
    auto* ver = cli.add_subcommand("version", "Show version information");
    ver->callback(
        [&app] { app.printer().print("elmos version {} ({})", ELMOS_VERSION, ELMOS_COMMIT); });
}

}  // namespace elmos::app::commands
