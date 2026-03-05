# ============================================================================
# Platform.cmake — Compile-time platform detection
# Replaces Go runtime.GOOS with #ifdef-based selection
# ============================================================================

if(CMAKE_SYSTEM_NAME STREQUAL "Darwin")
    set(ELMOS_PLATFORM "darwin")
    add_compile_definitions(ELMOS_PLATFORM_DARWIN=1)
    message(STATUS "ELMOS platform: macOS (Darwin)")

elseif(CMAKE_SYSTEM_NAME STREQUAL "Linux")
    set(ELMOS_PLATFORM "linux")
    add_compile_definitions(ELMOS_PLATFORM_LINUX=1)
    message(STATUS "ELMOS platform: Linux")

elseif(CMAKE_SYSTEM_NAME STREQUAL "Windows")
    set(ELMOS_PLATFORM "windows")
    add_compile_definitions(ELMOS_PLATFORM_WINDOWS=1)
    message(STATUS "ELMOS platform: Windows (WSL2 required)")

else()
    set(ELMOS_PLATFORM "linux")
    add_compile_definitions(ELMOS_PLATFORM_LINUX=1)
    message(WARNING "Unknown platform '${CMAKE_SYSTEM_NAME}', defaulting to Linux")
endif()

# Helper function: conditionally include platform-specific source files
function(elmos_platform_sources target)
    cmake_parse_arguments(PARSE_ARGV 1 ARG "" "" "DARWIN;LINUX;WINDOWS;COMMON")

    target_sources(${target} PRIVATE ${ARG_COMMON})

    if(ELMOS_PLATFORM STREQUAL "darwin" AND ARG_DARWIN)
        target_sources(${target} PRIVATE ${ARG_DARWIN})
    elseif(ELMOS_PLATFORM STREQUAL "linux" AND ARG_LINUX)
        target_sources(${target} PRIVATE ${ARG_LINUX})
    elseif(ELMOS_PLATFORM STREQUAL "windows" AND ARG_WINDOWS)
        target_sources(${target} PRIVATE ${ARG_WINDOWS})
    endif()
endfunction()
