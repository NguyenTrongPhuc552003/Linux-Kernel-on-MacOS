# ============================================================================
# src/ui/tui/CMakeLists.txt
# ============================================================================

add_library(elmos_tui STATIC
    types.cpp
    app.cpp
)

target_include_directories(elmos_tui
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

# ftxui targets are available via find_package or FetchContent from root
target_link_libraries(elmos_tui
    PUBLIC ftxui::screen
    PUBLIC ftxui::dom
    PUBLIC ftxui::component
)
