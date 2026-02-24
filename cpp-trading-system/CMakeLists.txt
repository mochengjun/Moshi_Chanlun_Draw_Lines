cmake_minimum_required(VERSION 3.16)
project(moshi_chanlun_trading VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

# Use UTF-8 encoding for MSVC
if(MSVC)
    add_compile_options(/utf-8)
endif()

# Include directories
include_directories(${CMAKE_SOURCE_DIR}/include)
include_directories(${CMAKE_SOURCE_DIR}/third_party)

# Indicator library
add_subdirectory(src)

# CLI calculator tool
add_executable(calculator_cli apps/server/calculator_cli.cpp)
target_link_libraries(calculator_cli moshi_chanlun_indicator)

# Tests
option(BUILD_TESTS "Build unit tests" OFF)
if(BUILD_TESTS)
    enable_testing()
    add_subdirectory(tests)
endif()
