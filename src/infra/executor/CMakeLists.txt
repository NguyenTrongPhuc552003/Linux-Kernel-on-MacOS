# ============================================================================
# src/infra/executor/CMakeLists.txt
# ============================================================================

add_library(elmos_executor STATIC
    shell.cpp
    mock.cpp
    env.cpp
)

target_include_directories(elmos_executor
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
)

target_link_libraries(elmos_executor PUBLIC pthread)
