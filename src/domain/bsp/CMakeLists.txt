# ============================================================================
# src/domain/bsp/CMakeLists.txt
# ============================================================================

add_library(elmos_bsp STATIC
    firmware.cpp
    registry.cpp
)

target_include_directories(elmos_bsp
    PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}
    PUBLIC ${CMAKE_SOURCE_DIR}/include
    PUBLIC ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(elmos_bsp
    PUBLIC elmos_config
    PUBLIC yaml-cpp::yaml-cpp
    PUBLIC OpenSSL::SSL OpenSSL::Crypto
)
