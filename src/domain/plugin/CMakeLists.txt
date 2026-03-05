# ============================================================================
# src/domain/plugin/CMakeLists.txt
# ============================================================================

add_library(elmos_builtin_plugins STATIC
    builtin.cpp
)

target_include_directories(elmos_builtin_plugins
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos_builtin_plugins
    PUBLIC elmos_plugin
    PUBLIC elmos_context
)
