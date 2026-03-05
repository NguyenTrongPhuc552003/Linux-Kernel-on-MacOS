# ============================================================================
# src/infra/homebrew/CMakeLists.txt
# ============================================================================

add_library(elmos_homebrew STATIC
    resolver.cpp
)

target_include_directories(elmos_homebrew
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos_homebrew PUBLIC elmos_executor)
