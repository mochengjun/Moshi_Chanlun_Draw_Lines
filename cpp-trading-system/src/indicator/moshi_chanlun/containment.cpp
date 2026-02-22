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
// ============================================================================
std::string MoshiChanlunCalculator::determineMergeDirection(const std::vector<MergedKLine>& merged) {
    if (merged.size() < 2) return "UP";
    const auto& prev = merged[merged.size() - 2];
    const auto& curr = merged[merged.size() - 1];
    return (curr.high > prev.high) ? "UP" : "DOWN";
}

// ============================================================================
// removeContainment - 处理K线包含关系
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
            std::string dir = determineMergeDirection(result);

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
