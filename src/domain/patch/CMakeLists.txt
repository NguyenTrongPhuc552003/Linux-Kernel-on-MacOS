# ============================================================================
# src/domain/patch/CMakeLists.txt
# ============================================================================

add_library(elmos_patch STATIC
    patcher.cpp
)

target_include_directories(elmos_patch
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos_patch
    PUBLIC elmos_executor
    PUBLIC elmos_filesystem
)
