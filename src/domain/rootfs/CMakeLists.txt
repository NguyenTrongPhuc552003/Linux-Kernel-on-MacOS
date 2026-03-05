# ============================================================================
# src/domain/rootfs/CMakeLists.txt
# ============================================================================

add_library(elmos_rootfs STATIC
    builder.cpp
)

target_include_directories(elmos_rootfs
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos_rootfs
    PUBLIC elmos_context
)
