# ============================================================================
# EmbedResources.cmake — Compile-time resource embedding
# Replaces Go //go:embed directive
#
# Strategy: Generate a C++ source file that contains the resource data
# as constexpr arrays, with a lookup function.
# This avoids the cmrc dependency while being fully self-contained.
# ============================================================================

set(ELMOS_RESOURCES_DIR "${CMAKE_SOURCE_DIR}/resources")

# Generate embedded resource source from files in resources/
function(elmos_embed_resources target)
    set(GENERATED_HPP "${CMAKE_BINARY_DIR}/generated/embedded_resources.hpp")
    set(GENERATED_CPP "${CMAKE_BINARY_DIR}/generated/embedded_resources.cpp")

    file(MAKE_DIRECTORY "${CMAKE_BINARY_DIR}/generated")

    # Collect all resource files
    file(GLOB_RECURSE RESOURCE_FILES
        "${ELMOS_RESOURCES_DIR}/templates/*"
        "${ELMOS_RESOURCES_DIR}/schemas/*"
        "${ELMOS_RESOURCES_DIR}/toolchains/configs/*"
    )

    # Generate header
    file(WRITE "${GENERATED_HPP}"
"#pragma once
// Auto-generated — do not edit
#include <optional>
#include <string>
#include <string_view>
#include <vector>

namespace elmos::resources {

/// Look up an embedded resource by its path (relative to resources/).
/// Returns std::nullopt if not found.
auto get(std::string_view path) -> std::optional<std::string_view>;

/// Returns a list of all embedded resource paths.
auto list() -> std::vector<std::string_view>;

} // namespace elmos::resources
")

    # Start generating the cpp file
    set(CPP_CONTENT
"// Auto-generated — do not edit
#include \"embedded_resources.hpp\"
#include <array>
#include <algorithm>
#include <vector>

namespace elmos::resources {
namespace detail {
")

    set(RESOURCE_COUNT 0)
    set(ENTRIES "")

    foreach(RESOURCE_FILE ${RESOURCE_FILES})
        file(RELATIVE_PATH REL_PATH "${ELMOS_RESOURCES_DIR}" "${RESOURCE_FILE}")
        file(READ "${RESOURCE_FILE}" FILE_CONTENT)
        string(LENGTH "${FILE_CONTENT}" CONTENT_LENGTH)

        # Sanitize name for C++ identifier
        string(REGEX REPLACE "[^a-zA-Z0-9]" "_" SAFE_NAME "${REL_PATH}")

        # Escape content for C++ raw string
        string(APPEND CPP_CONTENT
            "static constexpr char data_${SAFE_NAME}[] = R\"ELMOS_RES(${FILE_CONTENT})ELMOS_RES\";\n\n")

        string(APPEND ENTRIES
            "    {\"${REL_PATH}\", std::string_view{data_${SAFE_NAME}, sizeof(data_${SAFE_NAME}) - 1}},\n")

        math(EXPR RESOURCE_COUNT "${RESOURCE_COUNT} + 1")
    endforeach()

    string(APPEND CPP_CONTENT
"
struct Entry {
    std::string_view path;
    std::string_view data;
};

static constexpr std::array<Entry, ${RESOURCE_COUNT}> entries = {{
${ENTRIES}}};

} // namespace detail

auto get(std::string_view path) -> std::optional<std::string_view> {
    for (const auto& e : detail::entries) {
        if (e.path == path) return e.data;
    }
    return std::nullopt;
}

auto list() -> std::vector<std::string_view> {
    std::vector<std::string_view> paths;
    paths.reserve(detail::entries.size());
    for (const auto& e : detail::entries) {
        paths.push_back(e.path);
    }
    return paths;
}

} // namespace elmos::resources
")

    file(WRITE "${GENERATED_CPP}" "${CPP_CONTENT}")

    target_sources(${target} PRIVATE "${GENERATED_CPP}")
    target_include_directories(${target} PUBLIC "${CMAKE_BINARY_DIR}/generated")
endfunction()
