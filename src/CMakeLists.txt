# ============================================================================
# src/CMakeLists.txt — Aggregates all ELMOS library layers + final executable
# ============================================================================

# Infrastructure layers (no domain dependencies)
add_subdirectory(infra)

# Configuration layer
add_subdirectory(config)

# Context layer (depends on config + infra)
add_subdirectory(context)

# Plugin framework (depends on context + ui)
add_subdirectory(plugin)

# Domain layers (depends on infra + config + context)
add_subdirectory(domain)

# UI layer (depends on common types only)
add_subdirectory(ui)

# Application layer (depends on everything)
add_subdirectory(app)

# ---- Final executable ----
add_executable(elmos main.cpp)

target_include_directories(elmos
    PRIVATE ${CMAKE_SOURCE_DIR}/include
    PRIVATE ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos PRIVATE elmos_app)

# Install
install(TARGETS elmos RUNTIME DESTINATION bin)
