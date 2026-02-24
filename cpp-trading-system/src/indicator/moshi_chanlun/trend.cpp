#include "indicator/moshi_chanlun/moshi_chanlun.h"
#include <algorithm>

namespace moshi {

// ============================================================================
// identifySameLevelTrendsWithKlines - 带K线极值验证的同级别走势识别
//
// 流程:
//   1. 对点序列进行区间极值验证和修正
//   2. 使用修正后的点序列进行走势识别
// ============================================================================
std::vector<SameLevelTrend> MoshiChanlunCalculator::identifySameLevelTrendsWithKlines(
    const std::vector<MarkPoint>& points, int multiplier,
    const std::vector<KLine>& klines) const
{
    if (points.size() < 2 || klines.empty()) return {};

    auto validatedPoints = validateAndCorrectExtremePoints(points, klines);
    return identifySameLevelTrends(validatedPoints, multiplier);
}

// ============================================================================
// identifySameLevelTrends - 同级别走势识别核心逻辑
//
// 优先级:
//   1. 收敛型中枢盘整 (convergent) - 最高, 需6+点
//   2. 扩张型中枢盘整 (divergent)  - 次高, 需6+点
//   3. 趋势型走势     (trend)      - 最低, 需2+点 (允许简单L-H/H-L走势)
// ============================================================================
std::vector<SameLevelTrend> MoshiChanlunCalculator::identifySameLevelTrends(
    const std::vector<MarkPoint>& points, int multiplier) const
{
    if (points.size() < 2) return {};

    std::vector<SameLevelTrend> trends;
    int i = 0;
    int n = static_cast<int>(points.size());

    while (i < n) {
        // 从L点出发: 尝试上涨走势
        if (points[i].type == PointType::L) {
            // 1. 收敛型中枢 (最高优先级)
            auto [trend1, end1] = tryBuildUpConvergentPivot(points, i, multiplier);
            if (trend1) {
                trends.push_back(*trend1);
                i = end1;
                continue;
            }

            // 2. 扩张型中枢
            auto [trend2, end2] = tryBuildUpDivergentPivot(points, i, multiplier);
            if (trend2) {
                trends.push_back(*trend2);
                i = end2;
                continue;
            }

            // 3. 普通上涨趋势 (允许2+点的简单走势)
            auto [trend3, end3] = tryBuildUpTrend(points, i, multiplier);
            if (trend3 && trend3->points.size() >= 2) {
                trends.push_back(*trend3);
                i = end3;
                continue;
            }
        }

        // 从H点出发: 尝试下跌走势
        if (points[i].type == PointType::H) {
            auto [trend4, end4] = tryBuildDownTrend(points, i, multiplier);
            if (trend4 && trend4->points.size() >= 2) {
                trends.push_back(*trend4);
                i = end4;
                continue;
            }
        }

        ++i;
    }

    return trends;
}

// ============================================================================
// tryBuildUpTrend - 构建上涨趋势走势
//
// 规则:
//   - 起始点为L, 高点上移, 低点上移
//   - 至少2个点 (允许简单L-H)
//   - LowPoint = 第一个L, HighPoint = 最后一个H
// ============================================================================
MoshiChanlunCalculator::TrendBuildResult MoshiChanlunCalculator::tryBuildUpTrend(
    const std::vector<MarkPoint>& points, int startIdx, int multiplier) const
{
    int n = static_cast<int>(points.size());
    if (startIdx >= n || points[startIdx].type != PointType::L) {
        return {std::nullopt, startIdx};
    }

    std::vector<MarkPoint> trendPts;
    trendPts.push_back(points[startIdx]);

    const MarkPoint* lastL    = &points[startIdx];
    const MarkPoint* lowestL  = lastL;
    const MarkPoint* highestH = nullptr;
    int endIdx = startIdx;

    for (int j = startIdx + 1; j < n; ++j) {
        const auto& pt = points[j];

        if (pt.type == PointType::H) {
            // 高点必须上移
            if (highestH && pt.price <= highestH->price) break;
            // 高点必须高于最近的低点
            if (lastL && pt.price <= lastL->price) break;

            trendPts.push_back(pt);
            if (!highestH || pt.price > highestH->price) {
                highestH = &points[j];
            }
            endIdx = j;
        } else { // PointType::L
            // 低点不能跌破起始低点
            if (pt.price <= lowestL->price) break;
            // 低点必须上移
            if (lastL && pt.price <= lastL->price) break;

            trendPts.push_back(pt);
            lastL = &points[j];
            endIdx = j;
        }
    }

    // 至少2点且包含H和L
    if (trendPts.size() < 2 || !highestH) {
        return {std::nullopt, startIdx};
    }

    int lCount = 0, hCount = 0;
    for (const auto& p : trendPts) {
        if (p.type == PointType::L) ++lCount; else ++hCount;
    }
    if (lCount < 1 || hCount < 1) {
        return {std::nullopt, startIdx};
    }

    SameLevelTrend trend;
    trend.type           = "up";
    trend.pattern        = "trend";
    trend.multiplier     = multiplier;
    trend.startIndex     = trendPts.front().index;
    trend.endIndex       = trendPts.back().index;
    trend.startTimestamp = trendPts.front().timestamp;
    trend.endTimestamp   = trendPts.back().timestamp;
    trend.lowPoint       = *lowestL;
    trend.highPoint      = *highestH;
    trend.points         = std::move(trendPts);
    trend.upgraded       = false;

    return {trend, endIdx};
}

// ============================================================================
// tryBuildDownTrend - 构建下跌趋势走势
//
// 规则:
//   - 起始点为H, 高点下移, 低点下移
//   - 至少2个点 (允许简单H-L)
//   - HighPoint = 第一个H, LowPoint = 最后一个L
// ============================================================================
MoshiChanlunCalculator::TrendBuildResult MoshiChanlunCalculator::tryBuildDownTrend(
    const std::vector<MarkPoint>& points, int startIdx, int multiplier) const
{
    int n = static_cast<int>(points.size());
    if (startIdx >= n || points[startIdx].type != PointType::H) {
        return {std::nullopt, startIdx};
    }

    std::vector<MarkPoint> trendPts;
    trendPts.push_back(points[startIdx]);

    const MarkPoint* lastH    = &points[startIdx];
    const MarkPoint* highestH = lastH;
    const MarkPoint* lowestL  = nullptr;
    int endIdx = startIdx;

    for (int j = startIdx + 1; j < n; ++j) {
        const auto& pt = points[j];

        if (pt.type == PointType::L) {
            // 低点必须下移
            if (lowestL && pt.price >= lowestL->price) break;
            // 低点必须低于最近的高点
            if (lastH && pt.price >= lastH->price) break;

            trendPts.push_back(pt);
            if (!lowestL || pt.price < lowestL->price) {
                lowestL = &points[j];
            }
            endIdx = j;
        } else { // PointType::H
            // 高点不能突破起始高点
            if (pt.price >= highestH->price) break;
            // 高点必须下移
            if (lastH && pt.price >= lastH->price) break;

            trendPts.push_back(pt);
            lastH = &points[j];
            endIdx = j;
        }
    }

    if (trendPts.size() < 2 || !lowestL) {
        return {std::nullopt, startIdx};
    }

    int lCount = 0, hCount = 0;
    for (const auto& p : trendPts) {
        if (p.type == PointType::L) ++lCount; else ++hCount;
    }
    if (lCount < 1 || hCount < 1) {
        return {std::nullopt, startIdx};
    }

    SameLevelTrend trend;
    trend.type           = "down";
    trend.pattern        = "trend";
    trend.multiplier     = multiplier;
    trend.startIndex     = trendPts.front().index;
    trend.endIndex       = trendPts.back().index;
    trend.startTimestamp = trendPts.front().timestamp;
    trend.endTimestamp   = trendPts.back().timestamp;
    trend.highPoint      = *highestH;
    trend.lowPoint       = *lowestL;
    trend.points         = std::move(trendPts);
    trend.upgraded       = false;

    return {trend, endIdx};
}

// ============================================================================
// tryBuildUpConvergentPivot - 构建上涨收敛型中枢盘整走势
//
// 规则 (至少6点: L1-H1-L2-H2-L3-H3):
//   - L1 < L2 < L3 (低点上移)
//   - H1 > H2 < H3 且 H3 > H1 (高点先下移再上移, H3突破H1)
//
// 级别升级:
//   当 (H2到L3的barCount) > (H1到L2的barCount) * 2 时触发
//   父级别点映射: L1->L1, H1->H2, L2->L3, H2->H3
// ============================================================================
MoshiChanlunCalculator::TrendBuildResult MoshiChanlunCalculator::tryBuildUpConvergentPivot(
    const std::vector<MarkPoint>& points, int startIdx, int multiplier) const
{
    int n = static_cast<int>(points.size());
    if (startIdx >= n || points[startIdx].type != PointType::L) {
        return {std::nullopt, startIdx};
    }

    std::vector<MarkPoint> trendPts;
    std::vector<MarkPoint> lows;
    std::vector<MarkPoint> highs;

    trendPts.push_back(points[startIdx]);
    lows.push_back(points[startIdx]);
    int endIdx = startIdx;

    for (int j = startIdx + 1; j < n; ++j) {
        const auto& pt = points[j];

        // 低点不能跌破第一个低点
        if (pt.type == PointType::L && pt.price <= lows[0].price) break;

        trendPts.push_back(pt);
        if (pt.type == PointType::L) {
            lows.push_back(pt);
        } else {
            highs.push_back(pt);
        }
        endIdx = j;

        // 检查是否满足收敛型条件
        if (lows.size() >= 3 && highs.size() >= 3) {
            const auto& l1 = lows[0];
            const auto& l2 = lows[1];
            const auto& l3 = lows[2];
            const auto& h1 = highs[0];
            const auto& h2 = highs[1];
            const auto& h3 = highs[2];

            bool lowsAscending   = l1.price < l2.price && l2.price < l3.price;
            bool highsConvergent = h1.price > h2.price && h2.price < h3.price && h3.price > h1.price;

            if (lowsAscending && highsConvergent) {
                SameLevelTrend trend;
                trend.type           = "up";
                trend.pattern        = "convergent";
                trend.multiplier     = multiplier;
                trend.startIndex     = trendPts.front().index;
                trend.endIndex       = trendPts.back().index;
                trend.startTimestamp = trendPts.front().timestamp;
                trend.endTimestamp   = trendPts.back().timestamp;
                trend.lowPoint       = l1;
                trend.highPoint      = highs.back();
                trend.points         = trendPts;
                trend.upgraded       = false;

                // 检查级别升级条件
                int firstBarCount  = l2.index - h1.index;
                int secondBarCount = l3.index - h2.index;

                if (secondBarCount > firstBarCount * 2) {
                    trend.upgraded = true;
                    trend.parentPoints = {l1, h2, l3, h3};
                }

                return {trend, endIdx};
            }
        }
    }

    return {std::nullopt, startIdx};
}

// ============================================================================
// tryBuildUpDivergentPivot - 构建上涨扩张型中枢盘整走势
//
// 规则 (至少6点: L1-H1-L2-H2-L3-H3):
//   - L1 < L2 > L3 且 L3 > L1 (低点先上移再下移, L3仍高于L1)
//   - H1 < H2 < H3 (高点持续上移)
// ============================================================================
MoshiChanlunCalculator::TrendBuildResult MoshiChanlunCalculator::tryBuildUpDivergentPivot(
    const std::vector<MarkPoint>& points, int startIdx, int multiplier) const
{
    int n = static_cast<int>(points.size());
    if (startIdx >= n || points[startIdx].type != PointType::L) {
        return {std::nullopt, startIdx};
    }

    std::vector<MarkPoint> trendPts;
    std::vector<MarkPoint> lows;
    std::vector<MarkPoint> highs;

    trendPts.push_back(points[startIdx]);
    lows.push_back(points[startIdx]);
    int endIdx = startIdx;

    for (int j = startIdx + 1; j < n; ++j) {
        const auto& pt = points[j];

        // 低点不能跌破第一个低点
        if (pt.type == PointType::L && pt.price <= lows[0].price) break;

        trendPts.push_back(pt);
        if (pt.type == PointType::L) {
            lows.push_back(pt);
        } else {
            highs.push_back(pt);
        }
        endIdx = j;

        // 检查是否满足扩张型条件
        if (lows.size() >= 3 && highs.size() >= 3) {
            const auto& l1 = lows[0];
            const auto& l2 = lows[1];
            const auto& l3 = lows[2];
            const auto& h1 = highs[0];
            const auto& h2 = highs[1];
            const auto& h3 = highs[2];

            bool lowsDivergent  = l1.price < l2.price && l2.price > l3.price && l3.price > l1.price;
            bool highsAscending = h1.price < h2.price && h2.price < h3.price;

            if (lowsDivergent && highsAscending) {
                SameLevelTrend trend;
                trend.type           = "up";
                trend.pattern        = "divergent";
                trend.multiplier     = multiplier;
                trend.startIndex     = trendPts.front().index;
                trend.endIndex       = trendPts.back().index;
                trend.startTimestamp = trendPts.front().timestamp;
                trend.endTimestamp   = trendPts.back().timestamp;
                trend.lowPoint       = l1;
                trend.highPoint      = highs.back();
                trend.points         = trendPts;
                trend.upgraded       = false;

                return {trend, endIdx};
            }
        }
    }

    return {std::nullopt, startIdx};
}

} // namespace moshi
