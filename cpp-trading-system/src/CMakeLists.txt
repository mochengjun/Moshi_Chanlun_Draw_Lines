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
