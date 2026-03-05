#pragma once
// ============================================================================
// ui/tui/types.hpp — TUI data types, menu items, and model state
// ============================================================================

#include <functional>
#include <string>
#include <vector>

namespace elmos::ui::tui {

/// MenuItem represents a menu entry in the TUI.
struct MenuItem {
    std::string label;
    std::string desc;
    std::string action;
    std::string command;
    bool interactive = false;
    bool needs_input = false;
    std::string input_prompt;
    std::string input_placeholder;
    std::vector<std::string> args;
    std::vector<MenuItem> children;
};

/// CommandResult from running an action.
struct CommandResult {
    std::string action;
    std::string output;
    int exit_code = 0;
};

/// Build the complete menu tree.
auto build_menu_structure() -> std::vector<MenuItem>;

}  // namespace elmos::ui::tui
