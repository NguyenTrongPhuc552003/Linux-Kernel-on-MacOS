# ============================================================================
# src/domain/orchestrator/CMakeLists.txt
# ============================================================================

add_library(elmos_orchestrator STATIC
    pipeline.cpp
    fingerprint.cpp
)

target_include_directories(elmos_orchestrator
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos_orchestrator
    PUBLIC OpenSSL::SSL OpenSSL::Crypto
)
