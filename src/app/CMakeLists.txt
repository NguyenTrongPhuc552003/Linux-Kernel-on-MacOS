# ============================================================================
# src/app/CMakeLists.txt
# ============================================================================

add_library(elmos_app STATIC
    app.cpp
    commands/registry.cpp
    commands/init.cpp
    commands/kernel.cpp
    commands/module.cpp
    commands/apps.cpp
    commands/doctor.cpp
    commands/arch.cpp
    commands/qemu.cpp
    commands/toolchain.cpp
    commands/rootfs.cpp
    commands/bootloader.cpp
    commands/patch.cpp
    commands/plugins.cpp
    commands/version.cpp
    commands/tui.cpp
    commands/status.cpp
    commands/bsp.cpp
)

target_include_directories(elmos_app
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

find_package(CLI11 CONFIG REQUIRED)

target_link_libraries(elmos_app
    PUBLIC CLI11::CLI11
    PUBLIC elmos_context
    PUBLIC elmos_plugin
    PUBLIC elmos_builtin_plugins
    PUBLIC elmos_builder
    PUBLIC elmos_emulator
    PUBLIC elmos_doctor
    PUBLIC elmos_rootfs
    PUBLIC elmos_patch
    PUBLIC elmos_toolchain
    PUBLIC elmos_bsp
    PUBLIC elmos_ui
    PUBLIC elmos_tui
)
