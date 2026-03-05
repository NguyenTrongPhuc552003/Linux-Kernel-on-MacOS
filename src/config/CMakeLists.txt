# ============================================================================
# src/config/CMakeLists.txt
# ============================================================================

add_library(elmos_config STATIC
    arch.cpp
    machine.cpp
    loader.cpp
    saver.cpp
    workspaces.cpp
)

target_include_directories(elmos_config
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos_config
    PUBLIC yaml-cpp::yaml-cpp
    PUBLIC elmos_executor
    PUBLIC elmos_platform
)
