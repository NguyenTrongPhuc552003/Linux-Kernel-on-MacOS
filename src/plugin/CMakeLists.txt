# ============================================================================
# src/plugin/CMakeLists.txt
# ============================================================================

add_library(elmos_plugin STATIC
    executor.cpp
    loader.cpp
)

target_include_directories(elmos_plugin
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos_plugin
    PUBLIC elmos_context
)
