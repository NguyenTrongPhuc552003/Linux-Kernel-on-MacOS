#pragma once
// ============================================================================
// ui/tui/app.hpp — FTXUI-based interactive TUI application
// ============================================================================

#include "types.hpp"

#include <ftxui/component/component.hpp>
#include <ftxui/component/screen_interactive.hpp>
#include <ftxui/dom/elements.hpp>

#include <string>
#include <vector>

namespace elmos::ui::tui {

/// The main TUI application using FTXUI.
class App {
public:
    App();

    /// Run the TUI event loop (blocks until quit).
    void run();

private:
    // State
    std::vector<MenuItem> root_menu_;
    std::vector<MenuItem>* current_menu_;
    std::vector<std::vector<MenuItem>*> menu_stack_;
    int cursor_ = 0;
    std::string parent_title_;
    std::vector<std::string> log_lines_;
    bool is_running_ = false;
    std::string current_task_;
    bool input_mode_ = false;
    std::string input_value_;
    std::string input_action_;
    std::string input_prompt_;

    // FTXUI
    ftxui::ScreenInteractive screen_;

    // Rendering
    auto render_left_panel() -> ftxui::Element;
    auto render_right_panel() -> ftxui::Element;
    auto render_footer() -> ftxui::Element;

    // Actions
    void enter_submenu(MenuItem& item);
    void pop_menu();
    void execute_action(const MenuItem& item);
    void run_command(const std::string& action, const std::vector<std::string>& args);
    void submit_input();
    auto action_to_args(const std::string& action, const std::string& value)
        -> std::vector<std::string>;
};

/// Entry point — creates App and runs it.
void run_tui();

}  // namespace elmos::ui::tui
