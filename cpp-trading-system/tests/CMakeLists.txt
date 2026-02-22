include(FetchContent)

FetchContent_Declare(
    googletest
    GIT_REPOSITORY https://github.com/google/googletest.git
    GIT_TAG        v1.14.0
)
FetchContent_MakeAvailable(googletest)

add_executable(test_moshi_chanlun
    test_moshi_chanlun.cpp
)

target_link_libraries(test_moshi_chanlun
    moshi_chanlun_indicator
    GTest::gtest_main
)

include(GoogleTest)
gtest_discover_tests(test_moshi_chanlun)
