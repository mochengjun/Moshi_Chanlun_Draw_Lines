#include "indicator/moshi_chanlun/moshi_chanlun.h"
#include <algorithm>
#include <climits>
#include <cmath>
#include <set>

namespace moshi {

// ============================================================================
// deriveNextLevel - 从前一级别的H/L点推导出下一级别的H/L点
//
// 规则:
//   barCount >= minRetraceBars  → 确认跳级
//   barCount <  minRetraceBars  → 同级别波动, 合并到前一段
//   推动段可短于阈值, 回调段必须满足阈值
// ============================================================================
std::vector<MarkPoint> MoshiChanlunCalculator::deriveNextLevel(
    const std::vector<MarkPoint>& prevPoints,
    int minRetraceBars, int multiplier,
    const std::vector<KLine>& klines) const
{
    if (prevPoints.size() < 3) return {};

    std::string level = getLevelName(multiplier);
    std::vector<MarkPoint> result;

    // 添加第一个点作为起点
    auto first = prevPoints[0];
    result.push_back({first.type, first.index, first.timestamp, first.price, level, multiplier});

    PointType lastConfirmedType  = first.type;
    int       lastConfirmedIndex = first.index;
    double    lastConfirmedPrice = first.price;
    bool      impulseAllowed     = true; // 推动/回调交替: 首个候选为推动段

    // candidate: 追踪反方向的最佳极值候选
    struct CandidateInfo {
        MarkPoint point;
        bool      valid = false;
    } candidate;

    for (size_t i = 1; i < prevPoints.size(); ++i) {
        const auto& pt = prevPoints[i];

        if (pt.type != lastConfirmedType) {
            // 反方向点: 更新候选极值
            if (!candidate.valid) {
                candidate = {pt, true};
            } else {
                bool isMoreExtreme =
                    (pt.type == PointType::H && pt.price > candidate.point.price) ||
                    (pt.type == PointType::L && pt.price < candidate.point.price);
                if (isMoreExtreme) {
                    // 动态回溯检测
                    int trendDistance = pt.index - lastConfirmedIndex;
                    if (trendDistance >= minRetraceBars) {
                        auto [retroH, retroL] = scanRetroactiveRetrace(
                            prevPoints, lastConfirmedIndex, pt.index,
                            lastConfirmedType, minRetraceBars);

                        if (!retroH && !retroL) {
                            auto [kRetroH, kRetroL] = scanKLineRetrace(
                                klines, lastConfirmedIndex, pt.index,
                                lastConfirmedType, minRetraceBars, prevPoints);
                            retroH = kRetroH;
                            retroL = kRetroL;
                        }

                        if (retroH && retroL) {
                            // 找到合格回调对
                            if (lastConfirmedType == PointType::L) {
                                result.push_back({PointType::H, retroH->index, retroH->timestamp,
                                                  retroH->price, level, multiplier});
                                result.push_back({PointType::L, retroL->index, retroL->timestamp,
                                                  retroL->price, level, multiplier});
                                lastConfirmedType  = PointType::L;
                                lastConfirmedIndex = retroL->index;
                                lastConfirmedPrice = retroL->price;
                            } else {
                                result.push_back({PointType::L, retroL->index, retroL->timestamp,
                                                  retroL->price, level, multiplier});
                                result.push_back({PointType::H, retroH->index, retroH->timestamp,
                                                  retroH->price, level, multiplier});
                                lastConfirmedType  = PointType::H;
                                lastConfirmedIndex = retroH->index;
                                lastConfirmedPrice = retroH->price;
                            }
                            impulseAllowed = true;
                            candidate = {pt, true};
                            continue;
                        } else {
                            // 未找到中间回调, 直接确认当前更极端的反向点
                            result.push_back({pt.type, pt.index, pt.timestamp,
                                              pt.price, level, multiplier});
                            lastConfirmedType  = pt.type;
                            lastConfirmedIndex = pt.index;
                            lastConfirmedPrice = pt.price;
                            impulseAllowed = false;
                            candidate.valid = false;
                            continue;
                        }
                    }
                    candidate = {pt, true};
                }
            }
        } else {
            // 同方向点: 代表从candidate的回调/反弹
            if (!candidate.valid) continue;

            int barCount = pt.index - candidate.point.index;
            bool shouldConfirm = false;

            if (barCount >= minRetraceBars * 2) {
                shouldConfirm = true;
            } else if (barCount >= minRetraceBars) {
                shouldConfirm = true;
            }

            // 推动段规则
            if (!shouldConfirm && impulseAllowed) {
                if ((lastConfirmedType == PointType::L && pt.price >= lastConfirmedPrice) ||
                    (lastConfirmedType == PointType::H && pt.price <= lastConfirmedPrice)) {
                    shouldConfirm = true;
                }
            }

            if (shouldConfirm) {
                result.push_back({candidate.point.type, candidate.point.index,
                                  candidate.point.timestamp, candidate.point.price,
                                  level, multiplier});
                lastConfirmedType  = candidate.point.type;
                lastConfirmedIndex = candidate.point.index;
                lastConfirmedPrice = candidate.point.price;
                impulseAllowed = !impulseAllowed;
                candidate = {pt, true};
            }
        }
    }

    // 追加尾部候选点
    if (candidate.valid && !result.empty() &&
        candidate.point.type != result.back().type) {
        result.push_back({candidate.point.type, candidate.point.index,
                          candidate.point.timestamp, candidate.point.price,
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
