// ============================================================================
// ui/tui/app.cpp — FTXUI-based TUI implementation
// ============================================================================

#include "app.hpp"

#include <ftxui/component/component.hpp>
#include <ftxui/component/event.hpp>
#include <ftxui/component/screen_interactive.hpp>
#include <ftxui/dom/elements.hpp>

#include <array>
#include <cstdio>
#include <format>

namespace elmos::ui::tui {

using namespace ftxui;

// ---- Construction ----

App::App()
    : root_menu_(build_menu_structure()), current_menu_(&root_menu_),
      screen_(ScreenInteractive::Fullscreen()) {}

// ---- Rendering ----

auto App::render_left_panel() -> Element {
    Elements items;
    std::string title = parent_title_.empty() ? "ELMOS" : parent_title_;
    items.push_back(text("─ " + title + " ─") | bold | color(Color::Magenta));
    items.push_back(separator());

    if (!menu_stack_.empty()) {
        items.push_back(text("  ← Back (Esc)") | dim);
        items.push_back(text(""));
    }

    for (int i = 0; i < static_cast<int>(current_menu_->size()); ++i) {
        auto& item = (*current_menu_)[i];
        std::string prefix = !item.children.empty() ? "▸ "
                             : item.interactive     ? "⚡"
                             : item.needs_input     ? "✎ "
                                                    : "• ";
        auto label = text(prefix + item.label);
        if (i == cursor_) {
            items.push_back(label | bold | inverted);
        }
        else {
            items.push_back(label | dim);
        }
    }

    return vbox(std::move(items)) | border | size(WIDTH, EQUAL, 32);
}

auto App::render_right_panel() -> Element {
    Elements lines;

    // Title
    std::string title = is_running_   ? "⏳ " + current_task_
                        : input_mode_ ? ">> Input Required"
                                      : "Output";
    lines.push_back(text("─ " + title + " ─") | bold | color(Color::Magenta));
    lines.push_back(separator());

    // Input mode
    if (input_mode_) {
        lines.push_back(text(input_prompt_) | bold | color(Color::Yellow));
        lines.push_back(text("> " + input_value_ + "█") | color(Color::White));
        lines.push_back(text("  Press Enter to confirm, Esc to cancel") | dim);
        lines.push_back(separator());
    }

    // Hint for current menu item
    if (!input_mode_ && !is_running_ && cursor_ >= 0 &&
        cursor_ < static_cast<int>(current_menu_->size())) {
        auto& item = (*current_menu_)[cursor_];
        if (!item.command.empty()) {
            lines.push_back(text(" $ " + item.command) | color(Color::Cyan) | border);
            if (!item.desc.empty()) {
                lines.push_back(text("  " + item.desc) | dim);
            }
        }
        else if (!item.children.empty()) {
            lines.push_back(text("  Press Enter to expand → " + item.desc) | dim);
        }
        lines.push_back(text(""));
    }

    // Log output
    for (auto& line : log_lines_) {
        lines.push_back(text(line));
    }

    return vbox(std::move(lines)) | border | flex;
}

auto App::render_footer() -> Element {
    return hbox({
               text("↑↓") | color(Color::Cyan),
               text(" Navigate  "),
               text("⏎") | color(Color::Cyan),
               text(" Select  "),
               text("Esc") | color(Color::Cyan),
               text(" Back  "),
               text("c") | color(Color::Cyan),
               text(" Clear  "),
               text("q") | color(Color::Cyan),
               text(" Quit"),
           }) |
           dim;
}

// ---- Navigation / Actions ----

void App::enter_submenu(MenuItem& item) {
    menu_stack_.push_back(current_menu_);
    parent_title_ = item.label;
    current_menu_ = &item.children;
    cursor_ = 0;
}

void App::pop_menu() {
    if (menu_stack_.empty())
        return;
    current_menu_ = menu_stack_.back();
    menu_stack_.pop_back();
    parent_title_ = "";
    cursor_ = 0;
}

void App::execute_action(const MenuItem& item) {
    if (!item.children.empty())
        return;

    if (item.needs_input) {
        input_mode_ = true;
        input_action_ = item.action;
        input_prompt_ = item.input_prompt;
        input_value_.clear();
        return;
    }

    if (!item.args.empty()) {
        run_command(item.action, item.args);
    }
}

void App::run_command(const std::string& action, const std::vector<std::string>& args) {
    is_running_ = true;
    current_task_ = action;

    std::string cmd_line;
    for (auto& a : args) {
        if (!cmd_line.empty())
            cmd_line += ' ';
        cmd_line += a;
    }
    log_lines_.push_back(std::format("  ▶ elmos {}", cmd_line));

    // Build full command
    std::string full_cmd = "elmos";
    for (auto& a : args) {
        full_cmd += ' ';
        full_cmd += a;
    }
    full_cmd += " 2>&1";

    // Execute and capture output
    std::array<char, 256> buf{};
    std::string output;
    FILE* pipe = popen(full_cmd.c_str(), "r");
    if (pipe) {
        while (std::fgets(buf.data(), static_cast<int>(buf.size()), pipe)) {
            output += buf.data();
        }
        int rc = pclose(pipe);
        if (!output.empty()) {
            // Split output into lines
            std::string line;
            for (char ch : output) {
                if (ch == '\n') {
                    log_lines_.push_back("  " + line);
                    line.clear();
                }
                else {
                    line += ch;
                }
            }
            if (!line.empty())
                log_lines_.push_back("  " + line);
        }
        if (rc == 0) {
            log_lines_.push_back("  ✓ Completed");
        }
        else {
            log_lines_.push_back(std::format("  ✗ Exited with code {}", rc));
        }
    }
    else {
        log_lines_.push_back("  ✗ Failed to execute command");
    }
    log_lines_.emplace_back();

    is_running_ = false;
    current_task_.clear();
}

void App::submit_input() {
    if (input_value_.empty())
        return;
    input_mode_ = false;
    auto args = action_to_args(input_action_, input_value_);
    run_command(input_action_, args);
}

auto App::action_to_args(const std::string& action, const std::string& value)
    -> std::vector<std::string> {
    if (action == "arch:set")
        return {"arch", value};
    if (action == "kernel:switch")
        return value.empty() ? std::vector<std::string>{"kernel", "switch"}
                             : std::vector<std::string>{"kernel", "switch", value};
    if (action == "kernel:config")
        return (value.empty() || value == "defconfig")
                   ? std::vector<std::string>{"kernel", "config"}
                   : std::vector<std::string>{"kernel", "config", value};
    if (action == "module:build")
        return value.empty() ? std::vector<std::string>{"module", "build"}
                             : std::vector<std::string>{"module", "build", value};
    if (action == "module:new")
        return {"module", "new", value};
    if (action == "app:build")
        return value.empty() ? std::vector<std::string>{"app", "build"}
                             : std::vector<std::string>{"app", "build", value};
    if (action == "app:new")
        return {"app", "new", value};
    if (action == "rootfs:build")
        return {"rootfs", "build", "-s", value};
    if (action == "toolchain:select")
        return {"toolchains", value};
    return {};
}

// ---- Main event loop ----

void App::run() {
    auto renderer = Renderer([this] {
        return vbox({
            hbox({render_left_panel(), render_right_panel()}),
            render_footer(),
        });
    });

    auto component = CatchEvent(renderer, [this](Event event) -> bool {
        // Input mode handling
        if (input_mode_) {
            if (event == Event::Escape) {
                input_mode_ = false;
                return true;
            }
            if (event == Event::Return) {
                submit_input();
                return true;
            }
            if (event == Event::Backspace && !input_value_.empty()) {
                input_value_.pop_back();
                return true;
            }
            if (event.is_character()) {
                input_value_ += event.character();
                return true;
            }
            return true;
        }

        // Normal mode
        if (event == Event::Character('q') || event == Event::Escape) {
            if (!menu_stack_.empty()) {
                pop_menu();
                return true;
            }
            screen_.Exit();
            return true;
        }
        if (event == Event::ArrowUp || event == Event::Character('k')) {
            if (cursor_ > 0)
                --cursor_;
            return true;
        }
        if (event == Event::ArrowDown || event == Event::Character('j')) {
            if (cursor_ < static_cast<int>(current_menu_->size()) - 1)
                ++cursor_;
            return true;
        }
        if (event == Event::Return) {
            if (cursor_ >= 0 && cursor_ < static_cast<int>(current_menu_->size())) {
                auto& item = (*current_menu_)[cursor_];
                if (!item.children.empty()) {
                    enter_submenu(item);
                }
                else {
                    execute_action(item);
                }
            }
            return true;
        }
        if (event == Event::Character('c')) {
            log_lines_.clear();
            return true;
        }
        return false;
    });

    screen_.Loop(component);
}

// ---- Entry point ----

void run_tui() {
    App app;
    app.run();
}

}  // namespace elmos::ui::tui
