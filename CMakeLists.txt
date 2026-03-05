cmake_minimum_required(VERSION 3.25)

# ============================================================================
# ELMOS — Embedded Linux SDK
# Root CMakeLists.txt
#
# Supports two dependency strategies:
#   1. vcpkg (set VCPKG_ROOT env) — preferred for all platforms
#   2. System packages (apt) + FetchContent fallback — zero-setup on Linux
# ============================================================================

# vcpkg toolchain must be set before project()
if(DEFINED ENV{VCPKG_ROOT} AND NOT DEFINED CMAKE_TOOLCHAIN_FILE)
    set(CMAKE_TOOLCHAIN_FILE "$ENV{VCPKG_ROOT}/scripts/buildsystems/vcpkg.cmake"
        CACHE STRING "Vcpkg toolchain file")
endif()

project(elmos
    VERSION 3.2.0
    DESCRIPTION "Embedded Linux SDK — smarter, faster alternative to Buildroot"
    LANGUAGES CXX
)

# ── C++23 standard ──────────────────────────────────────────────────────────
set(CMAKE_CXX_STANDARD 23)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

# ── Custom cmake modules ────────────────────────────────────────────────────
list(APPEND CMAKE_MODULE_PATH "${CMAKE_CURRENT_SOURCE_DIR}/cmake")
include(Version)
include(Platform)
include(EmbedResources)
include(FetchContent)

# ── Compiler warnings ──────────────────────────────────────────────────────
add_compile_options(
    -Wall -Wextra -Wpedantic
    -Wno-unused-parameter
    -Wno-missing-field-initializers
    -Wshadow -Wnon-virtual-dtor
)

# ── Output directories ──────────────────────────────────────────────────────
set(CMAKE_RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin")
set(CMAKE_ARCHIVE_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/lib")
set(CMAKE_LIBRARY_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/lib")

# ── Find external dependencies ──────────────────────────────────────────────
# These are typically available via apt or vcpkg
find_package(CLI11 CONFIG REQUIRED)
find_package(yaml-cpp CONFIG REQUIRED)
find_package(nlohmann_json CONFIG REQUIRED)
find_package(spdlog CONFIG REQUIRED)
find_package(OpenSSL REQUIRED)

# httplib: apt package name differs from vcpkg
find_package(httplib CONFIG QUIET)
if(NOT httplib_FOUND)
    # System apt package installs as cpp-httplib
    find_path(HTTPLIB_INCLUDE_DIR NAMES httplib.h PATH_SUFFIXES httplib)
    if(HTTPLIB_INCLUDE_DIR)
        add_library(httplib::httplib INTERFACE IMPORTED)
        set_target_properties(httplib::httplib PROPERTIES
            INTERFACE_INCLUDE_DIRECTORIES "${HTTPLIB_INCLUDE_DIR}"
        )
        message(STATUS "Found system httplib at ${HTTPLIB_INCLUDE_DIR}")
    else()
        message(FATAL_ERROR "httplib not found. Install libcpp-httplib-dev or use vcpkg.")
    endif()
endif()

# inja: header-only, likely not in apt — FetchContent fallback
find_package(inja CONFIG QUIET)
if(NOT inja_FOUND)
    message(STATUS "inja not found via find_package, fetching via FetchContent...")
    # Disable inja's own tests, benchmarks, and install targets to avoid
    # missing doctest/hayai headers and install conflicts with nlohmann_json.
    set(INJA_USE_EMBEDDED_JSON ON CACHE BOOL "" FORCE)
    set(INJA_INSTALL OFF CACHE BOOL "" FORCE)
    set(INJA_EXPORT OFF CACHE BOOL "" FORCE)
    set(INJA_BUILD_TESTS OFF CACHE BOOL "" FORCE)
    set(BUILD_TESTING OFF CACHE BOOL "" FORCE)
    set(BUILD_BENCHMARK OFF CACHE BOOL "" FORCE)
    # Suppress CMake deprecation warning from inja's cmake_minimum_required(VERSION 3.5).
    # CMAKE_POLICY_VERSION_MINIMUM (CMake 4.0+) overrides the minimum version to 3.10,
    # silencing the warning without modifying the upstream source.
    if(CMAKE_VERSION VERSION_GREATER_EQUAL "4.0")
        set(CMAKE_POLICY_VERSION_MINIMUM 3.10 CACHE STRING "" FORCE)
    endif()
    FetchContent_Declare(inja
        GIT_REPOSITORY https://github.com/pantor/inja.git
        GIT_TAG        v3.4.0
        GIT_SHALLOW    TRUE
    )
    FetchContent_MakeAvailable(inja)
    if(CMAKE_VERSION VERSION_GREATER_EQUAL "4.0")
        unset(CMAKE_POLICY_VERSION_MINIMUM CACHE)
    endif()
    # Re-enable BUILD_TESTING for our own project tests
    set(BUILD_TESTING ON CACHE BOOL "" FORCE)
endif()

# ftxui: rarely in apt — FetchContent fallback
find_package(ftxui CONFIG QUIET)
if(NOT ftxui_FOUND)
    message(STATUS "ftxui not found via find_package, fetching via FetchContent...")
    FetchContent_Declare(ftxui
        GIT_REPOSITORY https://github.com/ArthurSonzogni/FTXUI.git
        GIT_TAG        v5.0.0
        GIT_SHALLOW    TRUE
    )
    set(FTXUI_BUILD_DOCS OFF CACHE BOOL "" FORCE)
    set(FTXUI_BUILD_EXAMPLES OFF CACHE BOOL "" FORCE)
    set(FTXUI_BUILD_TESTS OFF CACHE BOOL "" FORCE)
    set(FTXUI_ENABLE_INSTALL OFF CACHE BOOL "" FORCE)
    FetchContent_MakeAvailable(ftxui)
endif()

# ── Project-wide include path ───────────────────────────────────────────────
include_directories("${CMAKE_CURRENT_SOURCE_DIR}/include")

# ── Subdirectories ──────────────────────────────────────────────────────────
add_subdirectory(src)

option(ELMOS_BUILD_TESTS "Build unit and integration tests" ON)
if(ELMOS_BUILD_TESTS)
    enable_testing()
    add_subdirectory(tests)
endif()
