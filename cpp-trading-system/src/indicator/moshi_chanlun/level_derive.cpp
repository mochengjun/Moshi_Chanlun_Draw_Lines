#include "indicator/moshi_chanlun/moshi_chanlun.h"
#include <algorithm>
#include <climits>
#include <cmath>
#include <set>

namespace moshi {

// ============================================================================
// deriveNextLevel - 从前一级别的H/L点推导出下一级别的H/L点
//
// 莫氏缠论级别递推规则:
//   1. 从sub-x1级别的第一个点开始
//   2. 使用临时极值点(Hx1temp, Lx1temp)追踪走势
//   3. 根据K线根数阈值和价格比较确认正式标注点
//   4. 推动段可短于阈值，回调段必须满足阈值
// ============================================================================
std::vector<MarkPoint> MoshiChanlunCalculator::deriveNextLevel(
    const std::vector<MarkPoint>& prevPoints,
    int minRetraceBars, int multiplier,
    const std::vector<KLine>& klines) const
{
    if (prevPoints.size() < 3) return {};

    std::string level = getLevelName(multiplier);
    std::vector<MarkPoint> result;
    
    // 用于追踪临时极值点的索引
    struct TempPoints {
        int hx1tempIdx = -1;  // 临时高点在prevPoints中的索引
        int lx1tempIdx = -1;  // 临时低点在prevPoints中的索引
        int nextIdx = 0;       // 下一个要处理的prevPoints索引
    } temp;
    
    // 初始化：从第一个点开始，假设为高点则作为Hx1temp1，否则作为Lx1temp1
    temp.nextIdx = 1;
    if (prevPoints[0].type == PointType::H) {
        temp.hx1tempIdx = 0;
    } else {
        temp.lx1tempIdx = 0;
    }
    
    // 追踪下跌段最低点/上涨段最高点（基于K线）
    int klTempIdx = -1;  // 下跌段临时最低点K线索引
    int khTempIdx = -1;  // 上涨段临时最高点K线索引
    
    // 处理主循环
    while (temp.nextIdx < static_cast<int>(prevPoints.size())) {
        const auto& pt = prevPoints[temp.nextIdx];
        
        // 如果当前点是高点
        if (pt.type == PointType::H) {
            if (temp.hx1tempIdx >= 0) {
                // 更新临时高点：取更高的高点
                if (pt.price > prevPoints[temp.hx1tempIdx].price) {
                    temp.hx1tempIdx = temp.nextIdx;
                }
            } else {
                // 第一个高点，作为Hx1temp1或Lx1temp2
                temp.hx1tempIdx = temp.nextIdx;
                
                // 如果有临时低点Lx1temp1，检查是否满足确认条件
                if (temp.lx1tempIdx >= 0) {
                    const auto& hx1temp1 = prevPoints[temp.hx1tempIdx];
                    const auto& lx1temp1 = prevPoints[temp.lx1tempIdx];
                    
                    // 从低点Lx1temp1到当前高点Hx1temp2的K线数
                    int barCount = hx1temp1.index - lx1temp1.index;
                    
                    // 检查是否形成有效的上涨段
                    if (barCount >= minRetraceBars) {
                        // 查找自Lx1temp1上涨以来的最高点（K线）
                        int startSearch = lx1temp1.index;
                        int endSearch = hx1temp1.index;
                        khTempIdx = startSearch;
                        for (int i = startSearch + 1; i <= endSearch; ++i) {
                            if (klines[i].high > klines[khTempIdx].high) {
                                khTempIdx = i;
                            }
                        }
                        
                        // 如果KHtemp > Hx1temp1，确认上涨段
                        if (klines[khTempIdx].high > hx1temp1.price) {
                            // 确认Lx1temp1、Hx1temp1、Lx1temp2为x1级别点
                            result.push_back({PointType::L, lx1temp1.index, lx1temp1.timestamp,
                                              lx1temp1.price, level, multiplier});
                            result.push_back({PointType::H, hx1temp1.index, hx1temp1.timestamp,
                                              hx1temp1.price, level, multiplier});
                            
                            // 将当前高点作为新的临时高点起点
                            temp.lx1tempIdx = -1;  // 重置
                            temp.hx1tempIdx = temp.nextIdx;
                        }
                    }
                }
            }
        }
        // 如果当前点是低点
        else {
            if (temp.lx1tempIdx >= 0) {
                // 更新临时低点：取更低的低点
                if (pt.price < prevPoints[temp.lx1tempIdx].price) {
                    temp.lx1tempIdx = temp.nextIdx;
                }
            } else {
                // 第一个低点，作为Lx1temp1
                temp.lx1tempIdx = temp.nextIdx;
                
                // 如果有临时高点Hx1temp1，检查是否满足确认条件
                if (temp.hx1tempIdx >= 0) {
                    const auto& hx1temp1 = prevPoints[temp.hx1tempIdx];
                    const auto& lx1temp1 = prevPoints[temp.lx1tempIdx];
                    
                    // 从高点Hx1temp1到当前低点Lx1temp2的K线数
                    int barCount = lx1temp1.index - hx1temp1.index;
                    
                    // 检查是否形成有效的下跌段
                    if (barCount >= minRetraceBars) {
                        // 查找自Hx1temp1下跌以来的最低点（K线）
                        int startSearch = hx1temp1.index;
                        int endSearch = lx1temp1.index;
                        klTempIdx = startSearch;
                        for (int i = startSearch + 1; i <= endSearch; ++i) {
                            if (klines[i].low < klines[klTempIdx].low) {
                                klTempIdx = i;
                            }
                        }
                        
                        // 如果KLtemp < Lx1temp1，确认下跌段
                        if (klines[klTempIdx].low < lx1temp1.price) {
                            // 确认Hx1temp1、Lx1temp1为x1级别点
                            result.push_back({PointType::H, hx1temp1.index, hx1temp1.timestamp,
                                              hx1temp1.price, level, multiplier});
                            result.push_back({PointType::L, lx1temp1.index, lx1temp1.timestamp,
                                              lx1temp1.price, level, multiplier});
                            
                            // 将当前低点作为新的临时低点起点
                            temp.hx1tempIdx = -1;  // 重置
                            temp.lx1tempIdx = temp.nextIdx;
                        }
                    }
                }
            }
        }
        
        temp.nextIdx++;
    }
    
    // 处理尾部：如果有未确认的临时极值点
    if (temp.hx1tempIdx >= 0 && !result.empty()) {
        const auto& pt = prevPoints[temp.hx1tempIdx];
        // 检查是否需要添加（与上一个点类型不同）
        if (result.back().type != pt.type) {
            result.push_back({pt.type, pt.index, pt.timestamp, pt.price, level, multiplier});
        }
    }
    if (temp.lx1tempIdx >= 0 && !result.empty()) {
        const auto& pt = prevPoints[temp.lx1tempIdx];
        if (result.back().type != pt.type) {
            result.push_back({pt.type, pt.index, pt.timestamp, pt.price, level, multiplier});
        }
    }
    
    // 确保至少包含起始点
    if (result.empty() && !prevPoints.empty()) {
        result.push_back({prevPoints[0].type, prevPoints[0].index, 
                         prevPoints[0].timestamp, prevPoints[0].price, 
                         level, multiplier});
    }
    
    return result;
}

// ============================================================================
// scanRetroactiveRetrace - 在prevPoints中扫描回溯回调对
// ============================================================================
MoshiChanlunCalculator::ScanResult MoshiChanlunCalculator::scanRetroactiveRetrace(
    const std::vector<MarkPoint>& prevPoints,
    int startIdx, int endIdx,
    PointType lastType, int minRetraceBars) const
{
    // 筛选范围内的点 (不含两端)
    std::vector<MarkPoint> filtered;
    for (const auto& pt : prevPoints) {
        if (pt.index > startIdx && pt.index < endIdx) {
            filtered.push_back(pt);
        }
    }
    if (filtered.size() < 2) return {};

    for (size_t i = 0; i + 1 < filtered.size(); ++i) {
        const auto& p1 = filtered[i];
        const auto& p2 = filtered[i + 1];

        if (lastType == PointType::L) {
            if (p1.type == PointType::H && p2.type == PointType::L &&
                p2.index - p1.index >= minRetraceBars) {
                return {p1, p2};
            }
        } else {
            if (p1.type == PointType::L && p2.type == PointType::H &&
                p2.index - p1.index >= minRetraceBars) {
                return {p2, p1};
            }
        }
    }
    return {};
}

// ============================================================================
// scanKLineRetrace - 在原始K线数据中扫描回调 (后备方案)
// ============================================================================
MoshiChanlunCalculator::ScanResult MoshiChanlunCalculator::scanKLineRetrace(
    const std::vector<KLine>& klines,
    int startIdx, int endIdx,
    PointType lastType, int minRetraceBars,
    const std::vector<MarkPoint>& prevPoints) const
{
    int scanStart = startIdx + 1;
    int scanEnd   = endIdx - 1;
    int kn = static_cast<int>(klines.size());
    if (scanStart > scanEnd || scanStart < 0 || scanEnd >= kn) return {};

    // 将K线极值对齐到最近的prevPoints
    auto snapToNearest = [&](int targetIdx, PointType targetType) -> std::optional<MarkPoint> {
        const MarkPoint* best = nullptr;
        int bestDist = INT_MAX;
        for (const auto& pt : prevPoints) {
            if (pt.type != targetType || pt.index <= startIdx || pt.index >= endIdx) continue;
            int dist = std::abs(targetIdx - pt.index);
            if (dist < bestDist) {
                bestDist = dist;
                best = &pt;
            }
        }
        return best ? std::optional<MarkPoint>(*best) : std::nullopt;
    };

    if (lastType == PointType::L) {
        // 找区间内最高点
        int peakIdx = scanStart;
        for (int i = scanStart + 1; i <= scanEnd; ++i) {
            if (klines[i].high > klines[peakIdx].high) peakIdx = i;
        }
        int troughStart = peakIdx + minRetraceBars;
        if (troughStart > scanEnd) return {};
        int troughIdx = troughStart;
        for (int i = troughStart + 1; i <= scanEnd; ++i) {
            if (klines[i].low < klines[troughIdx].low) troughIdx = i;
        }
        if (troughIdx - peakIdx >= minRetraceBars) {
            auto snH = snapToNearest(peakIdx, PointType::H);
            auto snL = snapToNearest(troughIdx, PointType::L);
            if (snH && snL && snL->index - snH->index >= minRetraceBars) {
                return {snH, snL};
            }
        }
    } else {
        int troughIdx = scanStart;
        for (int i = scanStart + 1; i <= scanEnd; ++i) {
            if (klines[i].low < klines[troughIdx].low) troughIdx = i;
        }
        int peakStart = troughIdx + minRetraceBars;
        if (peakStart > scanEnd) return {};
        int peakIdx = peakStart;
        for (int i = peakStart + 1; i <= scanEnd; ++i) {
            if (klines[i].high > klines[peakIdx].high) peakIdx = i;
        }
        if (peakIdx - troughIdx >= minRetraceBars) {
            auto snH = snapToNearest(peakIdx, PointType::H);
            auto snL = snapToNearest(troughIdx, PointType::L);
            if (snH && snL && snH->index - snL->index >= minRetraceBars) {
                return {snH, snL};
            }
        }
    }
    return {};
}

// ============================================================================
// insertMissingThresholdPoints - 检测并插入缺失的超阈值转折点
// ============================================================================
std::vector<MarkPoint> MoshiChanlunCalculator::insertMissingThresholdPoints(
    const std::vector<MarkPoint>& points,
    const std::vector<MarkPoint>& prevPoints,
    int minRetraceBars, int multiplier,
    const std::vector<KLine>& klines) const
{
    if (points.size() < 2 || prevPoints.size() < 2) return points;

    std::string level = getLevelName(multiplier);
    auto current = points;

    for (int iteration = 0; iteration < 10; ++iteration) {
        std::vector<MarkPoint> result;
        result.reserve(current.size() * 2);
        bool changed = false;

        for (size_t i = 0; i < current.size(); ++i) {
            result.push_back(current[i]);

            if (i + 1 < current.size()) {
                const auto& cur  = current[i];
                const auto& next = current[i + 1];

                std::vector<MarkPoint> between;
                for (const auto& pt : prevPoints) {
                    if (pt.index > cur.index && pt.index < next.index) {
                        between.push_back(pt);
                    }
                }

                auto inserts = findOverThresholdSegments(
                    between, cur, next, minRetraceBars, level, multiplier, klines);
                if (!inserts.empty()) {
                    result.insert(result.end(), inserts.begin(), inserts.end());
                    changed = true;
                }
            }
        }

        if (!changed) return result;

        // 排序并去重
        std::sort(result.begin(), result.end(),
                  [](const MarkPoint& a, const MarkPoint& b) { return a.index < b.index; });
        std::vector<MarkPoint> deduped;
        deduped.reserve(result.size());
        for (size_t i = 0; i < result.size(); ++i) {
            if (i == 0 || result[i].index != result[i - 1].index) {
                deduped.push_back(result[i]);
            }
        }
        current = std::move(deduped);
    }
    return current;
}

// ============================================================================
// findOverThresholdSegments - 在子级别点序列中查找超过阈值的段
// ============================================================================
std::vector<MarkPoint> MoshiChanlunCalculator::findOverThresholdSegments(
    const std::vector<MarkPoint>& betweenPoints,
    const MarkPoint& startPoint, const MarkPoint& endPoint,
    int minRetraceBars, const std::string& level, int multiplier,
    const std::vector<KLine>& /*klines*/) const
{
    if (betweenPoints.size() < 2) return {};

    std::vector<MarkPoint> inserts;
    for (size_t i = 0; i + 1 < betweenPoints.size(); ++i) {
        const auto& p1 = betweenPoints[i];
        const auto& p2 = betweenPoints[i + 1];

        if (p1.type == p2.type) continue;
        if (p2.index - p1.index < minRetraceBars) continue;

        if (p1.index != startPoint.index && p1.index != endPoint.index) {
            inserts.push_back({p1.type, p1.index, p1.timestamp, p1.price, level, multiplier});
        }
        if (p2.index != startPoint.index && p2.index != endPoint.index) {
            inserts.push_back({p2.type, p2.index, p2.timestamp, p2.price, level, multiplier});
        }
    }
    return inserts;
}

// ============================================================================
// validateAndCorrectExtremePoints - 验证并修正点序列中的极值点
// ============================================================================
std::vector<MarkPoint> MoshiChanlunCalculator::validateAndCorrectExtremePoints(
    const std::vector<MarkPoint>& points,
    const std::vector<KLine>& klines) const
{
    if (points.size() < 2 || klines.empty()) return points;

    auto result = points;

    for (int iteration = 0; iteration < 10; ++iteration) {
        bool changed = false;

        for (size_t i = 0; i < result.size(); ++i) {
            auto& pt = result[i];

            if (pt.type == PointType::H) {
                // 找前一个L和下一个L
                const MarkPoint* prevL = nullptr;
                const MarkPoint* nextL = nullptr;
                for (int j = static_cast<int>(i) - 1; j >= 0; --j) {
                    if (result[j].type == PointType::L) { prevL = &result[j]; break; }
                }
                for (size_t j = i + 1; j < result.size(); ++j) {
                    if (result[j].type == PointType::L) { nextL = &result[j]; break; }
                }

                if (prevL && nextL) {
                    auto corrected = findHighestInRange(klines, prevL->index, nextL->index);
                    if (corrected && corrected->index != pt.index) {
                        if (corrected->index == prevL->index || corrected->index == nextL->index) {
                            corrected = findHighestInRangeExcluding(
                                klines, prevL->index, nextL->index,
                                {prevL->index, nextL->index});
                            if (!corrected || corrected->index == pt.index) continue;
                        }
                        pt.index     = corrected->index;
                        pt.timestamp = klines[corrected->index].timestamp;
                        pt.price     = corrected->price;
                        changed = true;
                    }
                } else if (prevL) {
                    auto corrected = findHighestInRange(klines, prevL->index, pt.index);
                    if (corrected && corrected->index != pt.index) {
                        pt.index     = corrected->index;
                        pt.timestamp = klines[corrected->index].timestamp;
                        pt.price     = corrected->price;
                        changed = true;
                    }
                }
            } else { // PointType::L
                const MarkPoint* prevH = nullptr;
                const MarkPoint* nextH = nullptr;
                for (int j = static_cast<int>(i) - 1; j >= 0; --j) {
                    if (result[j].type == PointType::H) { prevH = &result[j]; break; }
                }
                for (size_t j = i + 1; j < result.size(); ++j) {
                    if (result[j].type == PointType::H) { nextH = &result[j]; break; }
                }

                if (prevH && nextH) {
                    auto corrected = findLowestInRange(klines, prevH->index, nextH->index);
                    if (corrected && corrected->index != pt.index) {
                        if (corrected->index == prevH->index || corrected->index == nextH->index) {
                            corrected = findLowestInRangeExcluding(
                                klines, prevH->index, nextH->index,
                                {prevH->index, nextH->index});
                            if (!corrected || corrected->index == pt.index) continue;
                        }
                        pt.index     = corrected->index;
                        pt.timestamp = klines[corrected->index].timestamp;
                        pt.price     = corrected->price;
                        changed = true;
                    }
                } else if (prevH) {
                    auto corrected = findLowestInRange(klines, prevH->index, pt.index);
                    if (corrected && corrected->index != pt.index) {
                        pt.index     = corrected->index;
                        pt.timestamp = klines[corrected->index].timestamp;
                        pt.price     = corrected->price;
                        changed = true;
                    }
                }
            }
        }

        if (!changed) break;
    }

    return removeSameIndexPairs(result);
}

// ============================================================================
// enforceMinBarDistance - 确保回调段间距不小于最小K线根数阈值
// ============================================================================
std::vector<MarkPoint> MoshiChanlunCalculator::enforceMinBarDistance(
    const std::vector<MarkPoint>& points, int minBars) const
{
    if (points.size() < 3) return points;

    auto current = points;

    for (int pass = 0; pass < 10; ++pass) {
        bool changed = false;
        std::vector<MarkPoint> result;
        result.push_back(current[0]);

        for (size_t i = 1; i < current.size(); ++i) {
            auto cur  = current[i];
            auto last = result.back();

            // 同类型合并
            if (cur.type == last.type) {
                changed = true;
                if ((cur.type == PointType::H && cur.price > last.price) ||
                    (cur.type == PointType::L && cur.price < last.price)) {
                    result.back() = cur;
                }
                continue;
            }

            // 判断是否为回调段
            bool isRetracement = false;
            if (result.size() >= 2) {
                const auto& prev = result[result.size() - 2];
                if (prev.type == PointType::L && cur.type == PointType::L) {
                    isRetracement = (cur.price >= prev.price);
                } else if (prev.type == PointType::H && cur.type == PointType::H) {
                    isRetracement = (cur.price <= prev.price);
                }
            }

            if (cur.index - last.index >= minBars || !isRetracement) {
                result.push_back(cur);
            } else {
                changed = true;
                if (result.size() >= 2) {
                    auto prev = result[result.size() - 2];
                    result.pop_back();
                    if (prev.type == cur.type) {
                        if ((cur.type == PointType::H && cur.price > prev.price) ||
                            (cur.type == PointType::L && cur.price < prev.price)) {
                            result.back() = cur;
                        }
                    } else {
                        result.push_back(cur);
                    }
                }
            }
        }

        if (!changed) break;
        current = std::move(result);
    }
    return current;
}

// ============================================================================
// appendTrailingPoints - 追加尾部追踪点
// ============================================================================
std::vector<MarkPoint> MoshiChanlunCalculator::appendTrailingPoints(
    const std::vector<MarkPoint>& points,
    const std::vector<KLine>& klines,
    int minBars, int multiplier) const
{
    if (points.size() < 2 || klines.size() < 2) return points;

    auto result = points;
    const auto& lastPt = result.back();
    int totalKLines = static_cast<int>(klines.size());
    int trailingGap = totalKLines - 1 - lastPt.index;

    if (trailingGap < minBars) return result;

    std::string level = getLevelName(multiplier);
    int startIdx = lastPt.index + 1;
    if (startIdx >= totalKLines) return result;

    if (lastPt.type == PointType::L) {
        int highIdx = startIdx;
        for (int i = startIdx + 1; i < totalKLines; ++i) {
            if (klines[i].high > klines[highIdx].high) highIdx = i;
        }
        if (highIdx > lastPt.index) {
            result.push_back({PointType::H, highIdx, klines[highIdx].timestamp,
                              klines[highIdx].high, level, multiplier});
            if (highIdx < totalKLines - 1) {
                int lowIdx = highIdx + 1;
                for (int i = highIdx + 2; i < totalKLines; ++i) {
                    if (klines[i].low < klines[lowIdx].low) lowIdx = i;
                }
                if (lowIdx > highIdx) {
                    result.push_back({PointType::L, lowIdx, klines[lowIdx].timestamp,
                                      klines[lowIdx].low, level, multiplier});
                }
            }
        }
    } else {
        int lowIdx = startIdx;
        for (int i = startIdx + 1; i < totalKLines; ++i) {
            if (klines[i].low < klines[lowIdx].low) lowIdx = i;
        }
        if (lowIdx > lastPt.index) {
            result.push_back({PointType::L, lowIdx, klines[lowIdx].timestamp,
                              klines[lowIdx].low, level, multiplier});
            if (lowIdx < totalKLines - 1) {
                int highIdx = lowIdx + 1;
                for (int i = lowIdx + 2; i < totalKLines; ++i) {
                    if (klines[i].high > klines[highIdx].high) highIdx = i;
                }
                if (highIdx > lowIdx) {
                    result.push_back({PointType::H, highIdx, klines[highIdx].timestamp,
                                      klines[highIdx].high, level, multiplier});
                }
            }
        }
    }
    return result;
}

// ============================================================================
// 极值查找辅助函数
// ============================================================================

std::optional<MoshiChanlunCalculator::ExtremeResult>
MoshiChanlunCalculator::findHighestInRange(const std::vector<KLine>& klines,
                                           int startIdx, int endIdx) const
{
    int s = std::max(startIdx, 0);
    int e = std::min(endIdx, static_cast<int>(klines.size()) - 1);
    if (s > e) return std::nullopt;

    int bestIdx = s;
    double bestPrice = klines[s].high;
    for (int i = s + 1; i <= e; ++i) {
        if (klines[i].high > bestPrice) { bestPrice = klines[i].high; bestIdx = i; }
    }
    return ExtremeResult{bestIdx, bestPrice};
}

std::optional<MoshiChanlunCalculator::ExtremeResult>
MoshiChanlunCalculator::findLowestInRange(const std::vector<KLine>& klines,
                                          int startIdx, int endIdx) const
{
    int s = std::max(startIdx, 0);
    int e = std::min(endIdx, static_cast<int>(klines.size()) - 1);
    if (s > e) return std::nullopt;

    int bestIdx = s;
    double bestPrice = klines[s].low;
    for (int i = s + 1; i <= e; ++i) {
        if (klines[i].low < bestPrice) { bestPrice = klines[i].low; bestIdx = i; }
    }
    return ExtremeResult{bestIdx, bestPrice};
}

std::optional<MoshiChanlunCalculator::ExtremeResult>
MoshiChanlunCalculator::findHighestInRangeExcluding(
    const std::vector<KLine>& klines, int startIdx, int endIdx,
    const std::vector<int>& excludeIndices) const
{
    int s = std::max(startIdx, 0);
    int e = std::min(endIdx, static_cast<int>(klines.size()) - 1);
    if (s > e) return std::nullopt;

    std::set<int> excluded(excludeIndices.begin(), excludeIndices.end());
    int bestIdx = -1;
    double bestPrice = 0.0;
    for (int i = s; i <= e; ++i) {
        if (excluded.count(i)) continue;
        if (bestIdx == -1 || klines[i].high > bestPrice) {
            bestPrice = klines[i].high;
            bestIdx = i;
        }
    }
    return (bestIdx >= 0) ? std::optional<ExtremeResult>({bestIdx, bestPrice}) : std::nullopt;
}

std::optional<MoshiChanlunCalculator::ExtremeResult>
MoshiChanlunCalculator::findLowestInRangeExcluding(
    const std::vector<KLine>& klines, int startIdx, int endIdx,
    const std::vector<int>& excludeIndices) const
{
    int s = std::max(startIdx, 0);
    int e = std::min(endIdx, static_cast<int>(klines.size()) - 1);
    if (s > e) return std::nullopt;

    std::set<int> excluded(excludeIndices.begin(), excludeIndices.end());
    int bestIdx = -1;
    double bestPrice = 0.0;
    for (int i = s; i <= e; ++i) {
        if (excluded.count(i)) continue;
        if (bestIdx == -1 || klines[i].low < bestPrice) {
            bestPrice = klines[i].low;
            bestIdx = i;
        }
    }
    return (bestIdx >= 0) ? std::optional<ExtremeResult>({bestIdx, bestPrice}) : std::nullopt;
}

// ============================================================================
// removeSameIndexPairs - 移除同索引的H/L点对
// ============================================================================
std::vector<MarkPoint> MoshiChanlunCalculator::removeSameIndexPairs(
    const std::vector<MarkPoint>& points) const
{
    if (points.size() < 2) return points;

    std::vector<MarkPoint> result;
    result.reserve(points.size());

    for (size_t i = 0; i < points.size(); ) {
        if (i + 1 < points.size() && points[i].index == points[i + 1].index) {
            if (!result.empty()) {
                PointType lastType = result.back().type;
                if (points[i].type != lastType) {
                    result.push_back(points[i]);
                } else if (points[i + 1].type != lastType) {
                    result.push_back(points[i + 1]);
                }
            }
            i += 2;
        } else {
            result.push_back(points[i]);
            ++i;
        }
    }
    return result;
}

} // namespace moshi
