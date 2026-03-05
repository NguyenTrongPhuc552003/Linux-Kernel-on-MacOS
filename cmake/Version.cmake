# ============================================================================
# Version.cmake — Git-based version injection
# Replaces Go ldflags: -X version.Version=... -X version.Commit=...
# ============================================================================

find_package(Git QUIET)

if(GIT_FOUND)
    execute_process(
        COMMAND ${GIT_EXECUTABLE} describe --tags --always --dirty
        WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
        OUTPUT_VARIABLE ELMOS_GIT_VERSION
        OUTPUT_STRIP_TRAILING_WHITESPACE
        ERROR_QUIET
    )
    execute_process(
        COMMAND ${GIT_EXECUTABLE} rev-parse --short HEAD
        WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
        OUTPUT_VARIABLE ELMOS_GIT_COMMIT
        OUTPUT_STRIP_TRAILING_WHITESPACE
        ERROR_QUIET
    )
endif()

if(NOT ELMOS_GIT_VERSION)
    set(ELMOS_GIT_VERSION "v${PROJECT_VERSION}-dev")
endif()
if(NOT ELMOS_GIT_COMMIT)
    set(ELMOS_GIT_COMMIT "unknown")
endif()

string(TIMESTAMP ELMOS_BUILD_DATE "%Y-%m-%dT%H:%M:%SZ" UTC)

message(STATUS "ELMOS version: ${ELMOS_GIT_VERSION} (${ELMOS_GIT_COMMIT}) built ${ELMOS_BUILD_DATE}")

# These are consumed by src/app/version/version.cpp via configure_file or add_definitions
add_compile_definitions(
    ELMOS_VERSION="${ELMOS_GIT_VERSION}"
    ELMOS_COMMIT="${ELMOS_GIT_COMMIT}"
    ELMOS_BUILD_DATE="${ELMOS_BUILD_DATE}"
)
