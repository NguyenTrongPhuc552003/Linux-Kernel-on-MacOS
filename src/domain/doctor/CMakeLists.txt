# ============================================================================
# src/domain/doctor/CMakeLists.txt
# ============================================================================

add_library(elmos_doctor STATIC
    checker.cpp
)

target_include_directories(elmos_doctor
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos_doctor
    PUBLIC elmos_config
    PUBLIC elmos_executor
    PUBLIC elmos_filesystem
    PUBLIC elmos_platform
    PUBLIC elmos_toolchain
)
