add_library(moshi_chanlun_indicator
    indicator/moshi_chanlun/containment.cpp
    indicator/moshi_chanlun/sub_level.cpp
    indicator/moshi_chanlun/level_derive.cpp
    indicator/moshi_chanlun/trend.cpp
    indicator/moshi_chanlun/moshi_chanlun.cpp
)

target_include_directories(moshi_chanlun_indicator PUBLIC
    ${CMAKE_SOURCE_DIR}/include
)

# Use UTF-8 encoding for source files
if(MSVC)
    target_compile_options(moshi_chanlun_indicator PRIVATE /utf-8)
endif()
