#include "indicator/moshi_chanlun/moshi_chanlun.h"

namespace moshi {

// ============================================================================
// calculateSubLevelPoints - 组合步骤1-3: 计算sub-x1级别标注点
// 1. 处理K线包含关系
// 2. 识别顶底分型
// 3. 在分型数组上用2根K线阈值确认转折点
// ============================================================================
std::vector<MarkPoint> MoshiChanlunCalculator::calculateSubLevelPoints(
    const std::vector<KLine>& klines, int /*klineType*/) const
{
    if (klines.size() < 3) return {};

    // Step 1: 处理K线包含关系
    auto merged = removeContainment(klines);
    if (merged.size() < 3) return {};

    // Step 2: 识别顶底分型
    auto fractals = identifyFractals(merged, klines);
    if (fractals.empty()) return {};

    // Step 3: 在顶底分型数组上计算sub-x1级别转折点
    return calculateSubX1FromFractals(fractals, klines);
}

// ============================================================================
// calculateSubX1FromFractals - 在顶底分型数组上计算sub-x1级别转折点
//
// 确认条件 (满足其一即可):
//   条件1 (价格突破): 后续同类型分型价格突破/跌破前一确认点
//   条件2 (K线根数阈值): 从临时极值点开始回调K线根数 >= 2根
// ============================================================================
std::vector<MarkPoint> MoshiChanlunCalculator::calculateSubX1FromFractals(
    const std::vector<MarkPoint>& fractals,
    const std::vector<KLine>& klines) const
{
    if (fractals.empty()) return {};

    constexpr int minRetraceBars = 2;
    std::vector<MarkPoint> points;
    points.reserve(fractals.size());

    // 第一个分型直接确认
    points.push_back(fractals[0]);
    MarkPoint lastConfirmed = fractals[0];

    // 临时候选 (与lastConfirmed反向的极值候选)
    const MarkPoint* tempCandidate = nullptr;

    for (size_t i = 1; i < fractals.size(); ++i) {
        const auto& current = fractals[i];

        if (current.type != lastConfirmed.type) {
            // 反向分型: 更新tempCandidate (保留更极端的)
            if (!tempCandidate) {
                tempCandidate = &fractals[i];
            } else {
                bool moreExtreme =
                    (current.type == PointType::H && current.price > tempCandidate->price) ||
                    (current.type == PointType::L && current.price < tempCandidate->price);
                if (moreExtreme) {
                    tempCandidate = &fractals[i];
                }
            }
        } else {
            // 同向分型: 检查是否可以确认tempCandidate
            if (tempCandidate) {
                bool conditionMet = false;

                // 条件1: 价格突破
                if (lastConfirmed.type == PointType::L) {
                    // 新L跌破前一确认L → 确认中间的H
                    if (current.price < lastConfirmed.price) {
                        conditionMet = true;
                    }
                } else {
                    // 新H突破前一确认H → 确认中间的L
                    if (current.price > lastConfirmed.price) {
                        conditionMet = true;
                    }
                }

                // 条件2: 回调K线根数 >= 2
                if (!conditionMet) {
                    int barCount = current.index - tempCandidate->index;
                    if (barCount >= minRetraceBars) {
                        conditionMet = true;
                    }
                }

                if (conditionMet) {
                    // 确认tempCandidate
                    points.push_back(*tempCandidate);
                    lastConfirmed = *tempCandidate;
                    tempCandidate = &fractals[i]; // 当前点成为新的反向候选
                } else {
                    // 未确认, 当前同向点可能替代lastConfirmed的起始位置
                    // (连续同向但tempCandidate未通过确认: 更新候选为更极端的同向)
                    bool moreExtremeSame =
                        (current.type == PointType::H && current.price > lastConfirmed.price) ||
                        (current.type == PointType::L && current.price < lastConfirmed.price);
                    if (moreExtremeSame && points.size() > 0) {
                        // 放弃tempCandidate, 更新lastConfirmed为更极端的同向点
                        points.back() = current;
                        lastConfirmed = current;
                        tempCandidate = nullptr;
                    }
                }
            } else {
                // 无反向候选, 更新起始点为更极端的同向分型
                bool moreExtreme =
                    (current.type == PointType::H && current.price > lastConfirmed.price) ||
                    (current.type == PointType::L && current.price < lastConfirmed.price);
                if (moreExtreme && !points.empty()) {
                    points.back() = current;
                    lastConfirmed = current;
                }
            }
        }
    }

    // 追加尾部未确认候选点
    if (tempCandidate && !points.empty() &&
        tempCandidate->type != points.back().type &&
        tempCandidate->index > lastConfirmed.index) {
        points.push_back(*tempCandidate);
    }

    return points;
}

} // namespace moshi
