#pragma once

#include "common/types.h"
#include <map>
#include <string>
#include <vector>
#include <optional>

namespace moshi {

/// 莫氏缠论画线指标计算器
/// 处理流程: K线包含关系处理 → 顶底分型识别 → sub-x1 → x1 → x2 → x4 → x8
class MoshiChanlunCalculator {
public:
    /// 主计算入口
    IndicatorResult calculate(const std::vector<KLine>& klines,
                              const std::map<std::string, double>& params) const;

    // ====== Step 1: K线包含关系处理 ======

    /// 处理K线包含关系，将具有包含关系的相邻K线合并
    std::vector<MergedKLine> removeContainment(const std::vector<KLine>& klines) const;

    // ====== Step 2: 顶底分型识别 ======

    /// 基于无包含关系的K线识别顶底分型
    std::vector<MarkPoint> identifyFractals(const std::vector<MergedKLine>& merged,
                                            const std::vector<KLine>& klines) const;

    // ====== Step 3: sub-x1 级别 ======

    /// 组合步骤1-3: 从原始K线计算sub-x1级别标注点
    std::vector<MarkPoint> calculateSubLevelPoints(const std::vector<KLine>& klines,
                                                   int klineType) const;

    /// 在顶底分型数组上计算sub-x1级别转折点 (2根K线阈值)
    std::vector<MarkPoint> calculateSubX1FromFractals(const std::vector<MarkPoint>& fractals,
                                                      const std::vector<KLine>& klines) const;

    // ====== Step 4-5: 级别递推 ======

    /// 从前一级别推导下一级别的H/L点
    std::vector<MarkPoint> deriveNextLevel(const std::vector<MarkPoint>& prevPoints,
                                           int minRetraceBars, int multiplier,
                                           const std::vector<KLine>& klines) const;

    /// 检测并插入缺失的超阈值转折点
    std::vector<MarkPoint> insertMissingThresholdPoints(
        const std::vector<MarkPoint>& points,
        const std::vector<MarkPoint>& prevPoints,
        int minRetraceBars, int multiplier,
        const std::vector<KLine>& klines) const;

    /// 验证并修正点序列中的极值点
    std::vector<MarkPoint> validateAndCorrectExtremePoints(
        const std::vector<MarkPoint>& points,
        const std::vector<KLine>& klines) const;

    /// 确保回调段间距不小于最小K线根数阈值
    std::vector<MarkPoint> enforceMinBarDistance(const std::vector<MarkPoint>& points,
                                                 int minBars) const;

    /// 追加尾部追踪点
    std::vector<MarkPoint> appendTrailingPoints(const std::vector<MarkPoint>& points,
                                                const std::vector<KLine>& klines,
                                                int minBars, int multiplier) const;

    // ====== 走势识别 ======

    /// 识别同级别走势 (带K线极值验证)
    std::vector<SameLevelTrend> identifySameLevelTrendsWithKlines(
        const std::vector<MarkPoint>& points, int multiplier,
        const std::vector<KLine>& klines) const;

    /// 识别同级别走势 (核心逻辑)
    std::vector<SameLevelTrend> identifySameLevelTrends(
        const std::vector<MarkPoint>& points, int multiplier) const;

private:
    // ---- 包含关系辅助 ----
    static bool hasContainment(double h1, double l1, double h2, double l2);
    /// 确定包含关系合并方向
    /// - 无包含关系时：high2>high1且low2>low1返回"UP"，high2<high1且low2<low1返回"DOWN"
    /// - 有包含关系时：比较最高点确定方向
    static std::string determineMergeDirection(const std::vector<MergedKLine>& merged);

    // ---- 级别递推辅助 ----
    struct ScanResult {
        std::optional<MarkPoint> highPt;
        std::optional<MarkPoint> lowPt;
    };
    ScanResult scanRetroactiveRetrace(const std::vector<MarkPoint>& prevPoints,
                                      int startIdx, int endIdx,
                                      PointType lastType, int minRetraceBars) const;
    ScanResult scanKLineRetrace(const std::vector<KLine>& klines,
                                int startIdx, int endIdx,
                                PointType lastType, int minRetraceBars,
                                const std::vector<MarkPoint>& prevPoints) const;

    std::vector<MarkPoint> findOverThresholdSegments(
        const std::vector<MarkPoint>& betweenPoints,
        const MarkPoint& startPoint, const MarkPoint& endPoint,
        int minRetraceBars, const std::string& level, int multiplier,
        const std::vector<KLine>& klines) const;

    // ---- 极值查找 ----
    struct ExtremeResult { int index; double price; };

    std::optional<ExtremeResult> findHighestInRange(const std::vector<KLine>& klines,
                                                    int startIdx, int endIdx) const;
    std::optional<ExtremeResult> findLowestInRange(const std::vector<KLine>& klines,
                                                   int startIdx, int endIdx) const;
    std::optional<ExtremeResult> findHighestInRangeExcluding(
        const std::vector<KLine>& klines, int startIdx, int endIdx,
        const std::vector<int>& excludeIndices) const;
    std::optional<ExtremeResult> findLowestInRangeExcluding(
        const std::vector<KLine>& klines, int startIdx, int endIdx,
        const std::vector<int>& excludeIndices) const;

    std::vector<MarkPoint> removeSameIndexPairs(const std::vector<MarkPoint>& points) const;

    // ---- 走势构建 ----
    struct TrendBuildResult {
        std::optional<SameLevelTrend> trend;
        int endIdx = 0;
    };

    TrendBuildResult tryBuildUpTrend(const std::vector<MarkPoint>& points,
                                     int startIdx, int multiplier) const;
    TrendBuildResult tryBuildDownTrend(const std::vector<MarkPoint>& points,
                                       int startIdx, int multiplier) const;
    TrendBuildResult tryBuildUpConvergentPivot(const std::vector<MarkPoint>& points,
                                               int startIdx, int multiplier) const;
    TrendBuildResult tryBuildUpDivergentPivot(const std::vector<MarkPoint>& points,
                                              int startIdx, int multiplier) const;
};

} // namespace moshi
