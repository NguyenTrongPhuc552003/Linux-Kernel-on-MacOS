# cmake/Purge.cmake — Standalone purge script
# ============================================================================
# Deletes ALL build variant directories (build/, build-release/, build-ci/,
# build.*/  etc.) from the project source root.
#
# Invoked via the purge custom target (cmake --workflow --preset purge), or
# directly: cmake -P cmake/Purge.cmake
#
# Design rationale: this must be a standalone -P script (not embedded in a
# build target command) so that CMake's own process is NOT running from inside
# one of the directories being deleted.  A build target command runs from
# within CMAKE_BINARY_DIR; after file(REMOVE_RECURSE) kills that directory the
# build runner (ninja/make) recreates it when writing its log — defeating the
# purge.  By running as a child cmake -P process invoked from build/.meta/, the
# parent inode survives long enough for the script to finish, and the entire
# build/ tree (including build/.meta/) vanishes once both processes exit.
# ============================================================================

string(ASCII 27 ESC)
set(C_MAG "${ESC}[1;35m")
set(C_YLW "${ESC}[1;33m")
set(C_GRN "${ESC}[1;32m")
set(C_DIM "${ESC}[2m")
set(C_RST "${ESC}[0m")

# Resolve the source root: one directory above this script (cmake/).
get_filename_component(SOURCE_ROOT "${CMAKE_CURRENT_LIST_DIR}/.." ABSOLUTE)

message("${C_MAG}━━━ ELMOS Purge ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${C_RST}")

# Collect all build variant directories: build/, build-release/, build-ci/, ...
file(GLOB _candidates LIST_DIRECTORIES true "${SOURCE_ROOT}/build*")

set(_removed 0)
foreach(_dir ${_candidates})
    if(IS_DIRECTORY "${_dir}")
        message("  ${C_YLW}▸ Removing${C_RST}  ${C_DIM}${_dir}${C_RST}")
        file(REMOVE_RECURSE "${_dir}")
        math(EXPR _removed "${_removed} + 1")
    endif()
endforeach()

if(_removed EQUAL 0)
    message("  ${C_DIM}(nothing to remove — tree is already clean)${C_RST}")
else()
    message("")
    message("${C_GRN}✓ ${_removed} director(ies) removed${C_RST}")
endif()
message("${C_DIM}  Run 'cmake --preset default' to reconfigure${C_RST}")
