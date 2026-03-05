# ============================================================================
# src/domain/toolchain/CMakeLists.txt
# ============================================================================

add_library(elmos_toolchain STATIC
    manager.cpp
)

target_include_directories(elmos_toolchain
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos_toolchain
    PUBLIC elmos_config
    PUBLIC elmos_executor
    PUBLIC elmos_filesystem
)
