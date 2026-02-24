#include "indicator/moshi_chanlun/moshi_chanlun.h"
#include <algorithm>
#include <cmath>

namespace moshi {

// ============================================================================
// hasContainment - 检查两根K线是否存在包含关系
// ============================================================================
bool MoshiChanlunCalculator::hasContainment(double h1, double l1, double h2, double l2) {
    return (h1 >= h2 && l1 <= l2) || (h2 >= h1 && l2 <= l1);
}

// ============================================================================
// determineMergeDirection - 确定包含关系合并方向
// 根据当前趋势方向确定：UP-上升趋势(高高), DOWN-下降趋势(低低)
// ============================================================================
std::string MoshiChanlunCalculator::determineMergeDirection(
    const std::vector<MergedKLine>& merged, 
    const std::string& currentTrend) {
    // 如果已有明确趋势，使用当前趋势方向
    if (currentTrend == "UP" || currentTrend == "DOWN") {
        return currentTrend;
    }
    
    // 初始状态：根据前后K线变化确定方向
    if (merged.size() < 2) return "UP";
    
    const auto& prev = merged[merged.size() - 2];
    const auto& curr = merged[merged.size() - 1];
    
    // 无包含关系时
    if (!hasContainment(prev.high, prev.low, curr.high, curr.low)) {
        if (curr.high > prev.high && curr.low > prev.low) {
            return "UP";
        }
        if (curr.high < prev.high && curr.low < prev.low) {
            return "DOWN";
        }
    }
    
    // 有包含关系时：比较最高点
    return (curr.high > prev.high) ? "UP" : "DOWN";
}

// ============================================================================
// checkAndUpdateTrend - 检查是否形成新的分型，更新趋势方向
// 返回值：true-形成新分型并更新了趋势, false-未形成分型
// ============================================================================
bool MoshiChanlunCalculator::checkAndUpdateTrend(
    const std::vector<MergedKLine>& merged,
    std::string& currentTrend) 
{
    if (merged.size() < 3) return false;
    
    int n = static_cast<int>(merged.size());
    const auto& left = merged[n - 3];
    const auto& mid = merged[n - 2];
    const auto& right = merged[n - 1];
    
    bool hasTopFractal = (mid.high > left.high && mid.high > right.high);
    bool hasBottomFractal = (mid.low < left.low && mid.low < right.low);
    
    // 顶分型形成：趋势转为向下
    if (hasTopFractal) {
        currentTrend = "DOWN";
        return true;
    }
    
    // 底分型形成：趋势转为向上
    if (hasBottomFractal) {
        currentTrend = "UP";
        return true;
    }
    
    return false;
}

// ============================================================================
// removeContainment - 处理K线包含关系（同步版）
// 在处理包含关系的过程中同时识别分型，确定趋势方向
// 上升趋势: max(High), max(Low)
// 下降趋势: min(High), min(Low)
// ============================================================================
std::vector<MergedKLine> MoshiChanlunCalculator::removeContainmentSync(
    const std::vector<KLine>& klines,
    std::vector<MarkPoint>& fractals) const
{
    if (klines.empty()) return {};

    std::vector<MergedKLine> result;
    result.reserve(klines.size());
    
    // 当前趋势方向：初始为空(UNDEFINED)，确定分型后变为UP或DOWN
    std::string currentTrend = "UNDEFINED";
    
    // 第一根K线直接加入
    result.push_back({
        klines[0].high, klines[0].low,
        0, 0,
        klines[0].timestamp, klines[0].timestamp
    });

    for (int i = 1; i < static_cast<int>(klines.size()); ++i) {
        auto& last = result.back();
        const auto& k = klines[i];

        if (hasContainment(last.high, last.low, k.high, k.low)) {
            // 确定合并方向（根据当前趋势或默认规则）
            std::string dir = determineMergeDirection(result, currentTrend);

            if (dir == "UP") {
                // 上升趋势: 取较高的High和较高的Low
                if (k.high > last.high) {
                    last.high = k.high;
                    last.highOrigIdx = i;
                    last.highTimestamp = k.timestamp;
                }
                if (k.low > last.low) {
                    last.low = k.low;
                    last.lowOrigIdx = i;
                    last.lowTimestamp = k.timestamp;
                }
            } else {
                // 下降趋势: 取较低的High和较低的Low
                if (k.high < last.high) {
                    last.high = k.high;
                    last.highOrigIdx = i;
                    last.highTimestamp = k.timestamp;
                }
                if (k.low < last.low) {
                    last.low = k.low;
                    last.lowOrigIdx = i;
                    last.lowTimestamp = k.timestamp;
                }
            }
            
            // 处理完包含关系后，检查是否形成分型并更新趋势
            if (result.size() >= 3) {
                checkAndUpdateTrend(result, currentTrend);
            }
        } else {
            // 无包含关系, 先添加新K线
            result.push_back({
                k.high, k.low,
                i, i,
                k.timestamp, k.timestamp
            });
            
            // 检查是否形成分型并更新趋势
            if (result.size() >= 3) {
                int n = static_cast<int>(result.size());
                const auto& left = result[n - 3];
                const auto& mid = result[n - 2];
                const auto& right = result[n - 1];
                
                // 识别顶分型
                if (mid.high > left.high && mid.high > right.high) {
                    int origIdx = mid.highOrigIdx;
                    fractals.push_back({
                        PointType::H, origIdx,
                        klines[origIdx].timestamp, mid.high,
                        "sub-x1", 0
                    });
                    // 顶分型形成，趋势转为向下
                    currentTrend = "DOWN";
                }
                
                // 识别底分型（与顶分型可共用同一根K线）
                if (mid.low < left.low && mid.low < right.low) {
                    int origIdx = mid.lowOrigIdx;
                    fractals.push_back({
                        PointType::L, origIdx,
                        klines[origIdx].timestamp, mid.low,
                        "sub-x1", 0
                    });
                    // 底分型形成，趋势转为向上
                    currentTrend = "UP";
                }
            }
        }
    }

    return result;
}

// ============================================================================
// removeContainment - 处理K线包含关系（兼容旧版）
// 将具有包含关系的相邻K线合并为一根无包含关系的K线
// 上升趋势: max(High), max(Low)
// 下降趋势: min(High), min(Low)
// ============================================================================
std::vector<MergedKLine> MoshiChanlunCalculator::removeContainment(
    const std::vector<KLine>& klines) const
{
    if (klines.empty()) return {};

    std::vector<MergedKLine> result;
    result.reserve(klines.size());

    // 第一根K线直接加入
    result.push_back({
        klines[0].high, klines[0].low,
        0, 0,
        klines[0].timestamp, klines[0].timestamp
    });

    for (int i = 1; i < static_cast<int>(klines.size()); ++i) {
        auto& last = result.back();
        const auto& k = klines[i];

        if (hasContainment(last.high, last.low, k.high, k.low)) {
            // 确定合并方向
            std::string dir = determineMergeDirection(result, "UNDEFINED");

            if (dir == "UP") {
                // 上升High和较高的Low趋势: 取较高的
                if (k.high > last.high) {
                    last.high = k.high;
                    last.highOrigIdx = i;
                    last.highTimestamp = k.timestamp;
                }
                if (k.low > last.low) {
                    last.low = k.low;
                    last.lowOrigIdx = i;
                    last.lowTimestamp = k.timestamp;
                }
            } else {
                // 下降趋势: 取较低的High和较低的Low
                if (k.high < last.high) {
                    last.high = k.high;
                    last.highOrigIdx = i;
                    last.highTimestamp = k.timestamp;
                }
                if (k.low < last.low) {
                    last.low = k.low;
                    last.lowOrigIdx = i;
                    last.lowTimestamp = k.timestamp;
                }
            }
        } else {
            // 无包含关系, 直接添加
            result.push_back({
                k.high, k.low,
                i, i,
                k.timestamp, k.timestamp
            });
        }
    }

    return result;
}

// ============================================================================
// identifyFractals - 基于无包含关系的K线识别顶底分型
// 简单定义:
//   顶分型: mid.High > left.High && mid.High > right.High
//   底分型: mid.Low  < left.Low  && mid.Low  < right.Low
// ============================================================================
std::vector<MarkPoint> MoshiChanlunCalculator::identifyFractals(
    const std::vector<MergedKLine>& merged,
    const std::vector<KLine>& klines) const
{
    if (merged.size() < 3) return {};

    std::vector<MarkPoint> fractals;
    int n = static_cast<int>(merged.size());
    int kn = static_cast<int>(klines.size());

    for (int i = 1; i < n - 1; ++i) {
        const auto& left  = merged[i - 1];
        const auto& mid   = merged[i];
        const auto& right = merged[i + 1];

        // 顶分型
        if (mid.high > left.high && mid.high > right.high) {
            int origIdx = mid.highOrigIdx;
            if (origIdx < kn) {
                fractals.push_back({
                    PointType::H, origIdx,
                    klines[origIdx].timestamp, mid.high,
                    "sub-x1", 0
                });
            }
        }

        // 底分型
        if (mid.low < left.low && mid.low < right.low) {
            int origIdx = mid.lowOrigIdx;
            if (origIdx < kn) {
                fractals.push_back({
                    PointType::L, origIdx,
                    klines[origIdx].timestamp, mid.low,
                    "sub-x1", 0
                });
            }
        }
    }

    return fractals;
}

} // namespace moshi
