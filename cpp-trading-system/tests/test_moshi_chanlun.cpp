#include <gtest/gtest.h>
#include "indicator/moshi_chanlun/moshi_chanlun.h"

using namespace moshi;

// ============================================================================
// 测试辅助函数
// ============================================================================

static MarkPoint makePoint(PointType typ, int index, const std::string& timestamp, double price) {
    return {typ, index, timestamp, price, "sub-x1", 0};
}

static KLine makeDailyKLine(const std::string& timestamp,
                            double open, double high, double low, double close) {
    return {timestamp, open, high, low, close, 1000000.0};
}

// ============================================================================
// removeContainment 测试
// ============================================================================

TEST(Containment, NoContainment) {
    MoshiChanlunCalculator calc;
    std::vector<KLine> klines = {
        makeDailyKLine("2026-01-05", 10, 15, 5, 12),
        makeDailyKLine("2026-01-06", 12, 20, 10, 18),
        makeDailyKLine("2026-01-07", 18, 25, 15, 22),
    };
    auto merged = calc.removeContainment(klines);
    ASSERT_EQ(merged.size(), 3u);
    EXPECT_DOUBLE_EQ(merged[0].high, 15);
    EXPECT_DOUBLE_EQ(merged[1].high, 20);
    EXPECT_DOUBLE_EQ(merged[2].high, 25);
}

TEST(Containment, WithContainment) {
    MoshiChanlunCalculator calc;
    // K2包含K1: H2>=H1 && L2<=L1
    std::vector<KLine> klines = {
        makeDailyKLine("2026-01-05", 10, 15, 8, 12),
        makeDailyKLine("2026-01-06", 10, 20, 5, 15),  // 包含K1
        makeDailyKLine("2026-01-07", 16, 25, 12, 22),
    };
    auto merged = calc.removeContainment(klines);
    // K1和K2应合并为一根, 加K3共2根
    ASSERT_EQ(merged.size(), 2u);
}

// ============================================================================
// identifyFractals 测试
// ============================================================================

TEST(Fractals, SimpleTopBottom) {
    MoshiChanlunCalculator calc;
    // 构造3根无包含的合并K线: 中间高于两侧 → 顶分型
    std::vector<MergedKLine> merged = {
        {10.0, 5.0, 0, 0, "ts0", "ts0"},
        {20.0, 8.0, 1, 1, "ts1", "ts1"},  // 顶分型
        {15.0, 6.0, 2, 2, "ts2", "ts2"},
    };
    std::vector<KLine> klines = {
        makeDailyKLine("ts0", 7, 10, 5, 8),
        makeDailyKLine("ts1", 12, 20, 8, 18),
        makeDailyKLine("ts2", 10, 15, 6, 12),
    };
    auto fractals = calc.identifyFractals(merged, klines);
    ASSERT_GE(fractals.size(), 1u);
    EXPECT_EQ(fractals[0].type, PointType::H);
    EXPECT_DOUBLE_EQ(fractals[0].price, 20.0);
}

// ============================================================================
// deriveNextLevel 测试
// ============================================================================

TEST(DeriveNextLevel, BelowThreshold) {
    MoshiChanlunCalculator calc;
    int minRetrace = 5;

    // L(0) -> H(5) -> L(9): barCount = 9 - 5 = 4 < 5
    std::vector<MarkPoint> points = {
        makePoint(PointType::L, 0, "2026-01-05", 3200),
        makePoint(PointType::H, 5, "2026-01-10", 3350),
        makePoint(PointType::L, 9, "2026-01-14", 3180),
    };
    std::vector<KLine> klines = {makeDailyKLine("2026-01-05", 3200, 3220, 3180, 3210)};

    auto result = calc.deriveNextLevel(points, minRetrace, 1, klines);
    // 低于阈值: 起始L + 尾部候选H, 不超过2个点
    EXPECT_LE(result.size(), 2u);
}

TEST(DeriveNextLevel, FirstGrayZoneAutoConfirm) {
    MoshiChanlunCalculator calc;
    int minRetrace = 5;

    // L(0) -> H(5) -> L(12): barCount = 12 - 5 = 7, 灰色区间自动确认
    std::vector<MarkPoint> points = {
        makePoint(PointType::L, 0, "2026-01-05", 3200),
        makePoint(PointType::H, 5, "2026-01-10", 3350),
        makePoint(PointType::L, 12, "2026-01-17", 3150),
    };
    std::vector<KLine> klines = {makeDailyKLine("2026-01-05", 3200, 3220, 3180, 3210)};

    auto result = calc.deriveNextLevel(points, minRetrace, 1, klines);
    ASSERT_GE(result.size(), 3u);
    EXPECT_EQ(result[1].type, PointType::H);
    EXPECT_EQ(result[1].index, 5);
}

TEST(DeriveNextLevel, AboveDoubleThreshold) {
    MoshiChanlunCalculator calc;
    int minRetrace = 5;

    // L(0) -> H(5) -> L(15): barCount = 10 >= 10 → 明确确认
    std::vector<MarkPoint> points = {
        makePoint(PointType::L, 0, "2026-01-05", 3200),
        makePoint(PointType::H, 5, "2026-01-10", 3350),
        makePoint(PointType::L, 15, "2026-01-20", 3100),
    };
    std::vector<KLine> klines = {makeDailyKLine("2026-01-05", 3200, 3220, 3180, 3210)};

    auto result = calc.deriveNextLevel(points, minRetrace, 1, klines);
    ASSERT_GE(result.size(), 3u);
    EXPECT_EQ(result[1].type, PointType::H);
    EXPECT_EQ(result[1].index, 5);
}

TEST(DeriveNextLevel, ExactBoundary) {
    MoshiChanlunCalculator calc;
    int minRetrace = 5;

    struct TestCase {
        std::string name;
        int retraceBars;
        bool shouldConfirm;
    };
    std::vector<TestCase> tests = {
        {"4 bars (< 1x)", 4, false},
        {"5 bars (= 1x)", 5, true},
        {"7 bars (gray)", 7, true},
        {"9 bars (gray)", 9, true},
        {"10 bars (= 2x)", 10, true},
        {"15 bars (> 2x)", 15, true},
    };

    for (const auto& tt : tests) {
        std::vector<MarkPoint> points = {
            makePoint(PointType::L, 0, "2026-01-05", 3200),
            makePoint(PointType::H, 5, "2026-01-10", 3350),
            makePoint(PointType::L, 5 + tt.retraceBars, "2026-01-20", 3100),
        };
        std::vector<KLine> klines = {makeDailyKLine("2026-01-05", 3200, 3220, 3180, 3210)};

        auto result = calc.deriveNextLevel(points, minRetrace, 1, klines);
        bool confirmed = result.size() >= 3;

        EXPECT_EQ(confirmed, tt.shouldConfirm)
            << "Failed for " << tt.name << " (retraceBars=" << tt.retraceBars
            << ", resultSize=" << result.size() << ")";
    }
}

TEST(DeriveNextLevel, MultipleRetracesAllConfirmed) {
    MoshiChanlunCalculator calc;
    int minRetrace = 5;

    std::vector<MarkPoint> points = {
        makePoint(PointType::L, 0, "2026-01-01", 3200),
        makePoint(PointType::H, 7, "2026-01-08", 3350),
        makePoint(PointType::L, 14, "2026-01-15", 3150),
        makePoint(PointType::H, 19, "2026-01-20", 3300),
        makePoint(PointType::L, 28, "2026-01-29", 3100),
        makePoint(PointType::H, 40, "2026-02-10", 3400),
    };
    std::vector<KLine> klines = {makeDailyKLine("2026-01-01", 3200, 3220, 3180, 3210)};

    auto result = calc.deriveNextLevel(points, minRetrace, 1, klines);
    EXPECT_GE(result.size(), 5u);
}

TEST(DeriveNextLevel, ShortRetraceNotConfirmed) {
    MoshiChanlunCalculator calc;
    int minRetrace = 5;

    std::vector<MarkPoint> points = {
        makePoint(PointType::L, 0, "2026-01-01", 3200),
        makePoint(PointType::H, 8, "2026-01-09", 3380),
        makePoint(PointType::L, 16, "2026-01-17", 3150),
        makePoint(PointType::H, 19, "2026-01-20", 3300),  // barCount = 3 < 5
        makePoint(PointType::L, 27, "2026-01-28", 3120),
    };
    std::vector<KLine> klines = {makeDailyKLine("2026-01-01", 3200, 3220, 3180, 3210)};

    auto result = calc.deriveNextLevel(points, minRetrace, 1, klines);
    for (const auto& p : result) {
        if (p.index == 19 && p.type == PointType::H) {
            FAIL() << "H at index 19 should NOT be confirmed (barCount=3 < 5)";
        }
    }
}

TEST(DeriveNextLevel, CascadeToX2) {
    MoshiChanlunCalculator calc;
    int minRetrace = 5;

    std::vector<MarkPoint> points = {
        makePoint(PointType::L, 0, "2026-01-01", 3100),
        makePoint(PointType::H, 10, "2026-01-11", 3400),
        makePoint(PointType::L, 22, "2026-01-23", 3050),
        makePoint(PointType::H, 37, "2026-02-07", 3500),
        makePoint(PointType::L, 50, "2026-02-20", 3000),
        makePoint(PointType::H, 65, "2026-03-07", 3550),
        makePoint(PointType::L, 80, "2026-03-22", 2950),
    };
    std::vector<KLine> klines = {makeDailyKLine("2026-01-01", 3100, 3120, 3080, 3110)};

    auto x1Points = calc.deriveNextLevel(points, minRetrace * 1, 1, klines);
    ASSERT_GE(x1Points.size(), 3u) << "x1Points needs >=3 for x2 derivation";

    auto x2Points = calc.deriveNextLevel(x1Points, minRetrace * 2, 2, klines);
    EXPECT_FALSE(x2Points.empty()) << "x2Points should not be empty";
}

// ============================================================================
// validateAndCorrectExtremePoints 测试
// ============================================================================

TEST(ValidateExtreme, HPointCorrection) {
    MoshiChanlunCalculator calc;

    std::vector<KLine> klines = {
        makeDailyKLine("2026-01-05", 3200, 3210, 3180, 3200),
        makeDailyKLine("2026-01-06", 3200, 3250, 3190, 3240),
        makeDailyKLine("2026-01-07", 3240, 3280, 3230, 3270),
        makeDailyKLine("2026-01-08", 3270, 3300, 3260, 3290),
        makeDailyKLine("2026-01-09", 3290, 3330, 3280, 3320),
        makeDailyKLine("2026-01-10", 3320, 3350, 3310, 3340),  // index=5: 原H1 H=3350
        makeDailyKLine("2026-01-13", 3340, 3380, 3330, 3370),
        makeDailyKLine("2026-01-14", 3370, 3420, 3360, 3400),  // index=7: 实际最高 H=3420
        makeDailyKLine("2026-01-15", 3400, 3390, 3350, 3360),
        makeDailyKLine("2026-01-16", 3360, 3340, 3300, 3310),
        makeDailyKLine("2026-01-19", 3310, 3320, 3280, 3290),
    };

    std::vector<MarkPoint> points = {
        {PointType::L, 0, "2026-01-05", 3180, "x1", 1},
        {PointType::H, 5, "2026-01-10", 3350, "x1", 1},  // 错误位置
        {PointType::L, 10, "2026-01-19", 3280, "x1", 1},
    };

    auto corrected = calc.validateAndCorrectExtremePoints(points, klines);
    ASSERT_EQ(corrected.size(), 3u);
    EXPECT_EQ(corrected[1].index, 7);
    EXPECT_DOUBLE_EQ(corrected[1].price, 3420);
}

TEST(ValidateExtreme, LPointCorrection) {
    MoshiChanlunCalculator calc;

    std::vector<KLine> klines = {
        makeDailyKLine("2026-01-05", 3400, 3420, 3380, 3400),
        makeDailyKLine("2026-01-06", 3400, 3390, 3350, 3360),
        makeDailyKLine("2026-01-07", 3360, 3340, 3300, 3310),
        makeDailyKLine("2026-01-08", 3310, 3300, 3280, 3290),  // index=3: 原L1 L=3280
        makeDailyKLine("2026-01-09", 3290, 3270, 3250, 3260),
        makeDailyKLine("2026-01-10", 3260, 3240, 3200, 3220),  // index=5: 实际最低 L=3200
        makeDailyKLine("2026-01-13", 3220, 3280, 3210, 3270),
        makeDailyKLine("2026-01-14", 3270, 3320, 3260, 3310),
        makeDailyKLine("2026-01-15", 3310, 3380, 3300, 3370),
        makeDailyKLine("2026-01-16", 3370, 3450, 3360, 3440),
    };

    std::vector<MarkPoint> points = {
        {PointType::H, 0, "2026-01-05", 3420, "x1", 1},
        {PointType::L, 3, "2026-01-08", 3280, "x1", 1},
        {PointType::H, 9, "2026-01-16", 3450, "x1", 1},
    };

    auto corrected = calc.validateAndCorrectExtremePoints(points, klines);
    ASSERT_EQ(corrected.size(), 3u);
    EXPECT_EQ(corrected[1].index, 5);
    EXPECT_DOUBLE_EQ(corrected[1].price, 3200);
}

TEST(ValidateExtreme, NoCorrection) {
    MoshiChanlunCalculator calc;

    std::vector<KLine> klines = {
        makeDailyKLine("2026-01-05", 3200, 3210, 3180, 3200),
        makeDailyKLine("2026-01-06", 3200, 3250, 3190, 3240),
        makeDailyKLine("2026-01-07", 3240, 3320, 3230, 3310),  // 最高点
        makeDailyKLine("2026-01-08", 3310, 3300, 3260, 3270),
        makeDailyKLine("2026-01-09", 3270, 3260, 3220, 3230),  // 最低点
        makeDailyKLine("2026-01-10", 3230, 3280, 3220, 3270),
        makeDailyKLine("2026-01-13", 3270, 3350, 3260, 3340),  // 最高点
    };

    std::vector<MarkPoint> points = {
        {PointType::L, 0, "2026-01-05", 3180, "x1", 1},
        {PointType::H, 2, "2026-01-07", 3320, "x1", 1},
        {PointType::L, 4, "2026-01-09", 3220, "x1", 1},
        {PointType::H, 6, "2026-01-13", 3350, "x1", 1},
    };

    auto corrected = calc.validateAndCorrectExtremePoints(points, klines);
    ASSERT_EQ(corrected.size(), 4u);
    for (size_t i = 0; i < corrected.size(); ++i) {
        EXPECT_EQ(corrected[i].index, points[i].index);
        EXPECT_DOUBLE_EQ(corrected[i].price, points[i].price);
    }
}

// ============================================================================
// enforceMinBarDistance 测试
// ============================================================================

TEST(EnforceMinBar, ImpulseAllowed) {
    MoshiChanlunCalculator calc;
    int minBars = 5;

    // 推动段3 bars < 5, 回调段7 bars >= 5
    std::vector<MarkPoint> points = {
        {PointType::L, 0, "", 100, "x1", 1},
        {PointType::H, 3, "", 120, "x1", 1},
        {PointType::L, 10, "", 105, "x1", 1},
        {PointType::H, 13, "", 125, "x1", 1},
        {PointType::L, 20, "", 108, "x1", 1},
    };

    auto result = calc.enforceMinBarDistance(points, minBars);
    EXPECT_EQ(result.size(), 5u);
}

TEST(EnforceMinBar, ShortRetracementRemoved) {
    MoshiChanlunCalculator calc;
    int minBars = 5;

    // H(8)→L(10): 回调段2 bars < 5, 应被移除
    std::vector<MarkPoint> points = {
        {PointType::L, 0, "", 100, "x1", 1},
        {PointType::H, 8, "", 120, "x1", 1},
        {PointType::L, 10, "", 105, "x1", 1},
        {PointType::H, 18, "", 130, "x1", 1},
    };

    auto result = calc.enforceMinBarDistance(points, minBars);
    EXPECT_EQ(result.size(), 2u);
    if (result.size() >= 2) {
        EXPECT_EQ(result[0].type, PointType::L);
        EXPECT_EQ(result[0].index, 0);
        EXPECT_EQ(result[1].type, PointType::H);
        EXPECT_EQ(result[1].index, 18);
    }
}

// ============================================================================
// identifySameLevelTrends 测试
// ============================================================================

TEST(Trends, UpTrend) {
    MoshiChanlunCalculator calc;

    std::vector<MarkPoint> points = {
        {PointType::L, 0, "2026-01-05", 3200, "x1", 1},
        {PointType::H, 5, "2026-01-10", 3350, "x1", 1},
        {PointType::L, 10, "2026-01-15", 3250, "x1", 1},
        {PointType::H, 15, "2026-01-20", 3400, "x1", 1},
    };

    auto trends = calc.identifySameLevelTrends(points, 1);
    ASSERT_EQ(trends.size(), 1u);
    EXPECT_EQ(trends[0].type, "up");
    EXPECT_EQ(trends[0].pattern, "trend");
    EXPECT_DOUBLE_EQ(trends[0].lowPoint.price, 3200);
    EXPECT_DOUBLE_EQ(trends[0].highPoint.price, 3400);
    EXPECT_EQ(trends[0].points.size(), 4u);
}

TEST(Trends, DownTrend) {
    MoshiChanlunCalculator calc;

    std::vector<MarkPoint> points = {
        {PointType::H, 0, "2026-01-05", 3500, "x1", 1},
        {PointType::L, 5, "2026-01-10", 3300, "x1", 1},
        {PointType::H, 10, "2026-01-15", 3400, "x1", 1},
        {PointType::L, 15, "2026-01-20", 3200, "x1", 1},
    };

    auto trends = calc.identifySameLevelTrends(points, 1);
    ASSERT_EQ(trends.size(), 1u);
    EXPECT_EQ(trends[0].type, "down");
    EXPECT_EQ(trends[0].pattern, "trend");
    EXPECT_DOUBLE_EQ(trends[0].highPoint.price, 3500);
    EXPECT_DOUBLE_EQ(trends[0].lowPoint.price, 3200);
}

TEST(Trends, ExtendedUpTrend6Points) {
    MoshiChanlunCalculator calc;

    std::vector<MarkPoint> points = {
        {PointType::L, 0, "2026-01-05", 3200, "x1", 1},
        {PointType::H, 5, "2026-01-10", 3350, "x1", 1},
        {PointType::L, 10, "2026-01-15", 3250, "x1", 1},
        {PointType::H, 15, "2026-01-20", 3400, "x1", 1},
        {PointType::L, 20, "2026-01-25", 3300, "x1", 1},
        {PointType::H, 25, "2026-01-30", 3500, "x1", 1},
    };

    auto trends = calc.identifySameLevelTrends(points, 1);
    ASSERT_EQ(trends.size(), 1u);
    EXPECT_EQ(trends[0].type, "up");
    EXPECT_EQ(trends[0].points.size(), 6u);
    EXPECT_DOUBLE_EQ(trends[0].lowPoint.price, 3200);
    EXPECT_DOUBLE_EQ(trends[0].highPoint.price, 3500);
}

TEST(Trends, InvalidUpTrendSplitsIntoSmall) {
    MoshiChanlunCalculator calc;

    // L2(3150) < L1(3200) → 不是完整4点上涨走势
    std::vector<MarkPoint> points = {
        {PointType::L, 0, "2026-01-05", 3200, "x1", 1},
        {PointType::H, 5, "2026-01-10", 3350, "x1", 1},
        {PointType::L, 10, "2026-01-15", 3150, "x1", 1},  // 违规
        {PointType::H, 15, "2026-01-20", 3400, "x1", 1},
    };

    auto trends = calc.identifySameLevelTrends(points, 1);
    for (const auto& tr : trends) {
        if (tr.type == "up" && tr.points.size() >= 4) {
            FAIL() << "Should not have a 4-point up trend when L2 < L1";
        }
    }
}

TEST(Trends, SimpleTwoPointTrend) {
    MoshiChanlunCalculator calc;

    std::vector<MarkPoint> points = {
        {PointType::L, 0, "2026-01-05", 3200, "x2", 2},
        {PointType::H, 10, "2026-01-15", 3500, "x2", 2},
    };

    auto trends = calc.identifySameLevelTrends(points, 2);
    ASSERT_EQ(trends.size(), 1u);
    EXPECT_EQ(trends[0].points.size(), 2u);
    EXPECT_DOUBLE_EQ(trends[0].lowPoint.price, 3200);
    EXPECT_DOUBLE_EQ(trends[0].highPoint.price, 3500);
}

// ============================================================================
// 中枢盘整走势测试
// ============================================================================

TEST(Trends, ConvergentPivot) {
    MoshiChanlunCalculator calc;

    // L1 < L2 < L3 且 H1 > H2 < H3 (H3 > H1)
    std::vector<MarkPoint> points = {
        {PointType::L, 0, "2026-01-05", 3200, "x1", 1},
        {PointType::H, 5, "2026-01-10", 3500, "x1", 1},
        {PointType::L, 10, "2026-01-15", 3300, "x1", 1},
        {PointType::H, 15, "2026-01-20", 3400, "x1", 1},  // H2 < H1
        {PointType::L, 20, "2026-01-25", 3350, "x1", 1},
        {PointType::H, 25, "2026-01-30", 3550, "x1", 1},  // H3 > H1
    };

    auto trends = calc.identifySameLevelTrends(points, 1);
    ASSERT_EQ(trends.size(), 1u);
    EXPECT_EQ(trends[0].type, "up");
    EXPECT_EQ(trends[0].pattern, "convergent");
    EXPECT_DOUBLE_EQ(trends[0].lowPoint.price, 3200);
    EXPECT_DOUBLE_EQ(trends[0].highPoint.price, 3550);
    EXPECT_EQ(trends[0].points.size(), 6u);
}

TEST(Trends, DivergentPivot) {
    MoshiChanlunCalculator calc;

    // L1 < L2 > L3 (L3 > L1) 且 H1 < H2 < H3
    std::vector<MarkPoint> points = {
        {PointType::L, 0, "2026-01-05", 3200, "x1", 1},
        {PointType::H, 5, "2026-01-10", 3400, "x1", 1},
        {PointType::L, 10, "2026-01-15", 3350, "x1", 1},
        {PointType::H, 15, "2026-01-20", 3500, "x1", 1},
        {PointType::L, 20, "2026-01-25", 3250, "x1", 1},  // L3 < L2 但 L3 > L1
        {PointType::H, 25, "2026-01-30", 3600, "x1", 1},
    };

    auto trends = calc.identifySameLevelTrends(points, 1);
    ASSERT_EQ(trends.size(), 1u);
    EXPECT_EQ(trends[0].type, "up");
    EXPECT_EQ(trends[0].pattern, "divergent");
    EXPECT_DOUBLE_EQ(trends[0].lowPoint.price, 3200);
    EXPECT_DOUBLE_EQ(trends[0].highPoint.price, 3600);
}

TEST(Trends, InvalidConvergent_H3BelowH1) {
    MoshiChanlunCalculator calc;

    // H3 < H1, 不满足收敛条件
    std::vector<MarkPoint> points = {
        {PointType::L, 0, "2026-01-05", 3200, "x1", 1},
        {PointType::H, 5, "2026-01-10", 3500, "x1", 1},
        {PointType::L, 10, "2026-01-15", 3300, "x1", 1},
        {PointType::H, 15, "2026-01-20", 3400, "x1", 1},
        {PointType::L, 20, "2026-01-25", 3350, "x1", 1},
        {PointType::H, 25, "2026-01-30", 3450, "x1", 1},  // H3 < H1=3500
    };

    auto trends = calc.identifySameLevelTrends(points, 1);
    for (const auto& tr : trends) {
        EXPECT_NE(tr.pattern, "convergent")
            << "Should not identify as convergent (H3 < H1)";
    }
}

// ============================================================================
// 收敛型中枢级别升级测试
// ============================================================================

TEST(Trends, ConvergentUpgrade) {
    MoshiChanlunCalculator calc;

    // H1→L2: 5天, H2→L3: 16天 → 16 > 5*2=10, 触发升级
    std::vector<MarkPoint> points = {
        {PointType::L, 0, "2026-01-05", 3200, "x1", 1},
        {PointType::H, 5, "2026-01-10", 3500, "x1", 1},
        {PointType::L, 10, "2026-01-15", 3300, "x1", 1},
        {PointType::H, 15, "2026-01-20", 3400, "x1", 1},
        {PointType::L, 31, "2026-02-05", 3350, "x1", 1},  // 16天
        {PointType::H, 36, "2026-02-10", 3550, "x1", 1},
    };

    auto trends = calc.identifySameLevelTrends(points, 1);
    ASSERT_EQ(trends.size(), 1u);
    EXPECT_EQ(trends[0].pattern, "convergent");
    EXPECT_TRUE(trends[0].upgraded);
    ASSERT_EQ(trends[0].parentPoints.size(), 4u);
    EXPECT_DOUBLE_EQ(trends[0].parentPoints[0].price, 3200);  // L1
    EXPECT_DOUBLE_EQ(trends[0].parentPoints[1].price, 3400);  // H2
    EXPECT_DOUBLE_EQ(trends[0].parentPoints[2].price, 3350);  // L3
    EXPECT_DOUBLE_EQ(trends[0].parentPoints[3].price, 3550);  // H3
}

TEST(Trends, ConvergentNoUpgrade) {
    MoshiChanlunCalculator calc;

    // H1→L2: 5天, H2→L3: 5天 → 5 <= 5*2=10, 不升级
    std::vector<MarkPoint> points = {
        {PointType::L, 0, "2026-01-05", 3200, "x1", 1},
        {PointType::H, 5, "2026-01-10", 3500, "x1", 1},
        {PointType::L, 10, "2026-01-15", 3300, "x1", 1},
        {PointType::H, 15, "2026-01-20", 3400, "x1", 1},
        {PointType::L, 20, "2026-01-25", 3350, "x1", 1},
        {PointType::H, 25, "2026-01-30", 3550, "x1", 1},
    };

    auto trends = calc.identifySameLevelTrends(points, 1);
    ASSERT_EQ(trends.size(), 1u);
    EXPECT_EQ(trends[0].pattern, "convergent");
    EXPECT_FALSE(trends[0].upgraded);
    EXPECT_TRUE(trends[0].parentPoints.empty());
}

// ============================================================================
// X2级别走势识别测试
// ============================================================================

TEST(Trends, X2LevelTrend) {
    MoshiChanlunCalculator calc;

    std::vector<MarkPoint> points = {
        {PointType::L, 0, "2026-01-05", 3200, "x2", 2},
        {PointType::H, 10, "2026-01-15", 3500, "x2", 2},
        {PointType::L, 25, "2026-01-30", 3300, "x2", 2},
        {PointType::H, 40, "2026-02-14", 3600, "x2", 2},
    };

    auto trends = calc.identifySameLevelTrends(points, 2);
    ASSERT_EQ(trends.size(), 1u);
    EXPECT_EQ(trends[0].multiplier, 2);
    EXPECT_EQ(trends[0].type, "up");
    EXPECT_EQ(trends[0].pattern, "trend");
}

TEST(Trends, X2ConvergentPivot) {
    MoshiChanlunCalculator calc;

    std::vector<MarkPoint> points = {
        {PointType::L, 0, "2026-01-05", 3200, "x2", 2},
        {PointType::H, 10, "2026-01-15", 3600, "x2", 2},
        {PointType::L, 20, "2026-01-25", 3350, "x2", 2},
        {PointType::H, 30, "2026-02-04", 3450, "x2", 2},  // H2 < H1
        {PointType::L, 40, "2026-02-14", 3400, "x2", 2},
        {PointType::H, 50, "2026-02-24", 3700, "x2", 2},  // H3 > H1
    };

    auto trends = calc.identifySameLevelTrends(points, 2);
    ASSERT_EQ(trends.size(), 1u);
    EXPECT_EQ(trends[0].pattern, "convergent");
    EXPECT_EQ(trends[0].multiplier, 2);
}

// ============================================================================
// calculate 主流程集成测试
// ============================================================================

TEST(Calculate, BasicIntegration) {
    MoshiChanlunCalculator calc;

    // 构造30根日K数据
    std::vector<KLine> klines;
    struct PriceData { double o, h, l, c; };
    std::vector<PriceData> prices = {
        {3200,3230,3190,3220}, {3220,3260,3210,3250}, {3250,3290,3240,3280},
        {3280,3320,3270,3310}, {3310,3350,3300,3340}, {3340,3370,3330,3360},
        {3360,3390,3350,3380}, {3380,3420,3370,3400},
        // 下跌
        {3400,3410,3340,3350}, {3350,3360,3280,3290}, {3290,3300,3240,3250},
        {3250,3260,3200,3210}, {3210,3220,3170,3180}, {3180,3190,3140,3150},
        // 反弹
        {3150,3190,3140,3180}, {3180,3220,3170,3210}, {3210,3250,3200,3240},
        {3240,3280,3230,3270}, {3270,3310,3260,3300}, {3300,3340,3290,3330},
        {3330,3370,3320,3360},
        // 下跌
        {3360,3370,3310,3320}, {3320,3330,3270,3280}, {3280,3290,3230,3240},
        {3240,3250,3190,3200}, {3200,3210,3160,3170}, {3170,3180,3130,3140},
        // 后续
        {3140,3160,3130,3150}, {3150,3170,3140,3160}, {3160,3180,3150,3170},
    };

    for (size_t i = 0; i < prices.size(); ++i) {
        char ts[32];
        snprintf(ts, sizeof(ts), "2026-01-%02zu", i + 5);
        klines.push_back(makeDailyKLine(ts, prices[i].o, prices[i].h, prices[i].l, prices[i].c));
    }

    std::map<std::string, double> params = {
        {"kline_type", 10},
        {"show_level_sub_x1", 1},
        {"show_level_1x", 1},
        {"show_level_2x", 1},
        {"show_level_4x", 1},
        {"show_level_8x", 1},
    };

    auto result = calc.calculate(klines, params);

    EXPECT_EQ(result.type, "moshi_chanlun");
    EXPECT_FALSE(result.markPoints.empty()) << "Should have mark points";
    EXPECT_FALSE(result.activeLevels.empty()) << "Should have active levels";

    // 检查各级别标注点
    int subX1Count = 0, x1Count = 0;
    for (const auto& mp : result.markPoints) {
        if (mp.multiplier == 0) ++subX1Count;
        if (mp.multiplier == 1) ++x1Count;
    }
    EXPECT_GT(subX1Count, 0) << "Should have sub-x1 mark points";
    // x1 可能需要足够长数据才有
}

TEST(Calculate, LevelHierarchy) {
    MoshiChanlunCalculator calc;

    // 需要足够长的数据来验证层级关系
    // 生成100根渐进式K线
    std::vector<KLine> klines;
    for (int i = 0; i < 100; ++i) {
        double base = 3000 + (i % 20 < 10 ? i * 5 : (20 - i % 20) * 5);
        char ts[32];
        snprintf(ts, sizeof(ts), "2026-%02d-%02d", 1 + i / 30, 1 + i % 30);
        klines.push_back(makeDailyKLine(ts, base, base + 30, base - 20, base + 10));
    }

    std::map<std::string, double> params = {
        {"kline_type", 10},
        {"show_level_1x", 1},
        {"show_level_2x", 1},
    };

    auto result = calc.calculate(klines, params);
    EXPECT_EQ(result.type, "moshi_chanlun");
    // 基本健壮性: 不崩溃, 有输出
    EXPECT_FALSE(result.markPoints.empty());
}

// ============================================================================
// getMinRetraceBars 测试
// ============================================================================

TEST(Config, MinRetraceBars) {
    EXPECT_EQ(getMinRetraceBars(10), 5);   // 日K
    EXPECT_EQ(getMinRetraceBars(11), 8);   // 周K
    EXPECT_EQ(getMinRetraceBars(20), 15);  // 月K
    EXPECT_EQ(getMinRetraceBars(1), 4);    // 1分钟
    EXPECT_EQ(getMinRetraceBars(99), 5);   // 默认值
}

TEST(Config, LevelName) {
    EXPECT_EQ(getLevelName(0), "sub-x1");
    EXPECT_EQ(getLevelName(1), "x1");
    EXPECT_EQ(getLevelName(2), "x2");
    EXPECT_EQ(getLevelName(4), "x4");
    EXPECT_EQ(getLevelName(8), "x8");
    EXPECT_EQ(getLevelName(99), "unknown");
}
