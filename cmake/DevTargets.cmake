# cmake/DevTargets.cmake — Custom developer targets with colored output
# Invoked via: cmake --build build --target <name>
#   or via: cmake --build --preset <name>
# Colors (C_RST, C_GRN, ...) are already in scope from the root include(Colors).

# ── fmt / fmt-check ────────────────────────────────────────────────────────
find_program(CLANG_FORMAT clang-format)
if(CLANG_FORMAT)
    file(GLOB_RECURSE ALL_CXX_SOURCES
        "${CMAKE_SOURCE_DIR}/src/*.cpp"
        "${CMAKE_SOURCE_DIR}/src/*.hpp"
        "${CMAKE_SOURCE_DIR}/include/elmos/*.hpp"
    )
    add_custom_target(fmt
        COMMAND ${CLANG_FORMAT} -i ${ALL_CXX_SOURCES}
        COMMAND ${CMAKE_COMMAND} -E echo "${C_GRN}✓ Formatted all C++ sources${C_RST}"
        WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
        COMMENT "${C_BLU}▸ Formatting${C_RST} C++ sources..."
        VERBATIM
    )
    add_custom_target(fmt-check
        COMMAND ${CLANG_FORMAT} --dry-run --Werror ${ALL_CXX_SOURCES}
        COMMAND ${CMAKE_COMMAND} -E echo "${C_GRN}✓ All files correctly formatted${C_RST}"
        WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
        COMMENT "${C_BLU}▸ Checking${C_RST} format compliance..."
        VERBATIM
    )
endif()

# ── loc (lines of code) ────────────────────────────────────────────────────
# Uses cmake -P script mode to avoid Makefile escaping issues with ANSI codes
file(WRITE "${CMAKE_BINARY_DIR}/loc.cmake" "
string(ASCII 27 ESC)
set(C \"$\{ESC}[1;36m\")
set(M \"$\{ESC}[1;35m\")
set(G \"$\{ESC}[1;32m\")
set(W \"$\{ESC}[1;37m\")
set(D \"$\{ESC}[2m\")
set(R \"$\{ESC}[0m\")
message(\"\$\{M}━━━ ELMOS Source Line Counts ━━━\$\{R}\")
foreach(dir src include tests)
    if(IS_DIRECTORY \"${CMAKE_SOURCE_DIR}/$\{dir}\")
        file(GLOB_RECURSE _files
            \"${CMAKE_SOURCE_DIR}/$\{dir}/*.cpp\"
            \"${CMAKE_SOURCE_DIR}/$\{dir}/*.hpp\"
            \"${CMAKE_SOURCE_DIR}/$\{dir}/*.h\"
        )
        set(_total 0)
        foreach(_f $\{_files})
            file(STRINGS \"$\{_f}\" _lines)
            list(LENGTH _lines _len)
            math(EXPR _total \"$\{_total} + $\{_len}\")
        endforeach()
        message(\"  \$\{C}$\{dir}/\$\{R}            $\{_total} lines\")
    endif()
endforeach()
")
add_custom_target(loc
    COMMAND ${CMAKE_COMMAND} -P "${CMAKE_BINARY_DIR}/loc.cmake"
    COMMENT ""
)

# ── purge (nuke ALL build* dirs via standalone cmake -P script) ───────────
# Runs cmake/Purge.cmake as a child process so CMake is NOT running from
# inside a directory it is about to delete.  build/.meta/ (this target's
# binary dir) is inside build/ and gets cleaned up when its parent inode
# expires after the child cmake -P process exits.
add_custom_target(purge
    COMMAND ${CMAKE_COMMAND} -P "${CMAKE_SOURCE_DIR}/cmake/Purge.cmake"
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    COMMENT ""
    VERBATIM
)

# ── doctor ──────────────────────────────────────────────────────────────────
add_custom_target(doctor
    COMMAND $<TARGET_FILE:elmos> doctor
    DEPENDS elmos
    COMMENT "${C_BLU}▸ Running${C_RST} elmos doctor..."
    VERBATIM
)

# ── install-system ────────────────────────────────────────────────────────
# Installs elmos to /usr/local/bin using sudo cmake --install.
# Requires a prior build (DEPENDS elmos).
set(ELMOS_INSTALL_PREFIX "/usr/local" CACHE PATH "System install prefix for elmos")
add_custom_target(install-system
    COMMAND ${CMAKE_COMMAND} -E echo "${C_CYN}▸ Installing${C_RST} elmos → ${ELMOS_INSTALL_PREFIX}/bin/"
    COMMAND sudo ${CMAKE_COMMAND} --install "${CMAKE_BINARY_DIR}" --prefix "${ELMOS_INSTALL_PREFIX}"
    COMMAND ${CMAKE_COMMAND} -E echo "${C_GRN}✓ Installed${C_RST} ${ELMOS_INSTALL_PREFIX}/bin/elmos"
    COMMAND ${ELMOS_INSTALL_PREFIX}/bin/elmos version
    DEPENDS elmos
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    COMMENT ""
    VERBATIM
)
