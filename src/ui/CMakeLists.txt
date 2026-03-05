# ============================================================================
# src/ui/CMakeLists.txt
# ============================================================================

add_subdirectory(tui)

# The printer and help formatter are header-only.
# Create an interface target for consumers.
add_library(elmos_ui INTERFACE)

target_include_directories(elmos_ui
    INTERFACE ${CMAKE_CURRENT_SOURCE_DIR}
    INTERFACE ${CMAKE_SOURCE_DIR}/include
    INTERFACE ${CMAKE_SOURCE_DIR}/src
)

find_package(CLI11 CONFIG REQUIRED)

target_link_libraries(elmos_ui
    INTERFACE CLI11::CLI11
    INTERFACE elmos_tui
)
