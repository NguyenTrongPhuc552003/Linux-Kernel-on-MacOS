# ============================================================================
# src/infra/filesystem/CMakeLists.txt
# ============================================================================

add_library(elmos_filesystem STATIC
    os_filesystem.cpp
)

target_include_directories(elmos_filesystem
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
)
