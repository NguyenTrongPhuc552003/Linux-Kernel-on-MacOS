# ============================================================================
# src/context/CMakeLists.txt
# ============================================================================

add_library(elmos_context STATIC
    context.cpp
)

target_include_directories(elmos_context
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos_context
    PUBLIC elmos_config
    PUBLIC elmos_executor
    PUBLIC elmos_filesystem
    PUBLIC elmos_platform
    PUBLIC elmos_homebrew
)
