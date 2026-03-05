// ============================================================================
// main.cpp — ELMOS entry point
// ============================================================================

#include <app/app.hpp>
#include <config/loader.hpp>
#include <infra/executor/shell.hpp>
#include <infra/filesystem/os_filesystem.hpp>

#include <cstdlib>
#include <iostream>

int main(int argc, char** argv) {
    // Load configuration (auto-detect workspace config)
    auto cfg_result = elmos::config::load("");
    elmos::config::Config cfg;
    if (cfg_result) {
        cfg = std::move(*cfg_result);
    }
    // If load fails, proceed with defaults — init command doesn't need config

    // Wire infrastructure
    auto exec = std::make_unique<elmos::infra::executor::ShellExecutor>();
    auto fs = std::make_unique<elmos::infra::filesystem::OSFileSystem>();

    // Build and run application
    elmos::app::App app(std::move(exec), std::move(fs), std::move(cfg));
    return app.run(argc, argv);
}
