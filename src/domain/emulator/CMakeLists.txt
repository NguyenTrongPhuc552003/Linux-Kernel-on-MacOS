# ============================================================================
# src/domain/emulator/CMakeLists.txt
# ============================================================================

add_library(elmos_emulator STATIC
    qemu.cpp
)

target_include_directories(elmos_emulator
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos_emulator
    PUBLIC elmos_context
)
