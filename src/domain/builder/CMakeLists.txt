# ============================================================================
# src/domain/builder/CMakeLists.txt
# ============================================================================

add_library(elmos_builder STATIC
    kernel.cpp
    module.cpp
    app.cpp
)

target_include_directories(elmos_builder
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos_builder
    PUBLIC elmos_context
    PUBLIC elmos_toolchain
)
