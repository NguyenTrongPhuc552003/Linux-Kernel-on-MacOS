# ============================================================================
# tests/CMakeLists.txt — Unit and integration tests
# ============================================================================

find_package(Catch2 3 CONFIG REQUIRED)
include(CTest)
# This provides catch_discover_tests
include(Catch)

# Unit tests
if(IS_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}/unit")
    file(GLOB_RECURSE UNIT_TEST_SOURCES "unit/*.cpp")
    if(UNIT_TEST_SOURCES)
        add_executable(elmos_unit_tests ${UNIT_TEST_SOURCES})
        target_link_libraries(elmos_unit_tests PRIVATE
            Catch2::Catch2WithMain
            elmos_app
        )
        target_include_directories(elmos_unit_tests PRIVATE
            "${CMAKE_SOURCE_DIR}/src"
            "${CMAKE_SOURCE_DIR}/include"
        )
        catch_discover_tests(elmos_unit_tests)
    endif()
endif()

# Integration tests
if(IS_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}/integration")
    file(GLOB_RECURSE INTEGRATION_TEST_SOURCES "integration/*.cpp")
    if(INTEGRATION_TEST_SOURCES)
        add_executable(elmos_integration_tests ${INTEGRATION_TEST_SOURCES})
        target_link_libraries(elmos_integration_tests PRIVATE
            Catch2::Catch2WithMain
            elmos_app
        )
        target_include_directories(elmos_integration_tests PRIVATE
            "${CMAKE_SOURCE_DIR}/src"
            "${CMAKE_SOURCE_DIR}/include"
        )
        catch_discover_tests(elmos_integration_tests)
    endif()
endif()
