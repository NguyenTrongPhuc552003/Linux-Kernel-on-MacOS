# ============================================================================
# src/infra/platform/CMakeLists.txt
# ============================================================================

add_library(elmos_platform STATIC
    detect.cpp
)

# Conditionally add platform-specific source files
elmos_platform_sources(elmos_platform
    DARWIN darwin.cpp
    LINUX linux.cpp
    WINDOWS windows.cpp
)

target_include_directories(elmos_platform
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos_platform PUBLIC elmos_executor)
