#include "indicator/moshi_chanlun/moshi_chanlun.h"
#include <algorithm>

namespace moshi {

// ============================================================================
// 参数解析辅助
// ============================================================================
static int getIntParam(const std::map<std::string, double>& params,
                       const std::string& key, int defaultVal)
{
    auto it = params.find(key);
    return (it != params.end()) ? static_cast<int>(it->second) : defaultVal;
}

static bool getBoolParam(const std::map<std::string, double>& params,
                         const std::string& key, bool defaultVal)
{
    auto it = params.find(key);
    return (it != params.end()) ? (it->second != 0.0) : defaultVal;
}

// ============================================================================
// calculate - 主计算入口
//
// 流程:
//   sub-x1 → x1 (递推+插入+验证+间距+验证) → x1走势
//            → x2 (递推+插入+验证+间距+验证) → x2走势
//            → x4 (递推+插入+验证+间距+验证) → x4走势
//            → x8 (递推+插入+验证+间距+验证)
//   各级别追加尾部追踪点 → 收集标注点和走势
// ============================================================================
IndicatorResult MoshiChanlunCalculator::calculate(
    const std::vector<KLine>& klines,
    const std::map<std::string, double>& params) const
{
    IndicatorResult result;
    result.type = "moshi_chanlun";
    result.name = "莫氏缠论";

    if (klines.size() < 3) return result;

    // 参数解析
    int klineType      = getIntParam(params, "kline_type", 10);
    bool showSubX1     = getBoolParam(params, "show_level_sub_x1", false);
    bool showLevel1x   = getBoolParam(params, "show_level_1x", true);
    bool showLevel2x   = getBoolParam(params, "show_level_2x", true);
    bool showLevel4x   = getBoolParam(params, "show_level_4x", true);
    bool showLevel8x   = getBoolParam(params, "show_level_8x", true);

    result.klineType = klineType;
    int baseMinRetraceBars = getMinRetraceBars(klineType);

    // === 层级递推计算 ===

    // Step 0: sub-x1级别 (基准=2根K线)
    auto subX1Points = calculateSubLevelPoints(klines, klineType);

    // Step 1: sub-x1 → x1
    auto x1Points = deriveNextLevel(subX1Points, baseMinRetraceBars * 1, 1, klines);
    x1Points = insertMissingThresholdPoints(x1Points, subX1Points, baseMinRetraceBars * 1, 1, klines);
    x1Points = validateAndCorrectExtremePoints(x1Points, klines);
    x1Points = enforceMinBarDistance(x1Points, baseMinRetraceBars * 1);
    x1Points = validateAndCorrectExtremePoints(x1Points, klines);

    // Step 1.5: x1走势识别
    auto x1Trends = identifySameLevelTrendsWithKlines(x1Points, 1, klines);

    // Step 2: x1 → x2
    auto x2Points = deriveNextLevel(x1Points, baseMinRetraceBars * 2, 2, klines);
    x2Points = insertMissingThresholdPoints(x2Points, x1Points, baseMinRetraceBars * 2, 2, klines);
    x2Points = validateAndCorrectExtremePoints(x2Points, klines);
    x2Points = enforceMinBarDistance(x2Points, baseMinRetraceBars * 2);
    x2Points = validateAndCorrectExtremePoints(x2Points, klines);

    // Step 2.5: x2走势识别
    auto x2Trends = identifySameLevelTrendsWithKlines(x2Points, 2, klines);

    // Step 3: x2 → x4
    auto x4Points = deriveNextLevel(x2Points, baseMinRetraceBars * 4, 4, klines);
    x4Points = insertMissingThresholdPoints(x4Points, x2Points, baseMinRetraceBars * 4, 4, klines);
    x4Points = validateAndCorrectExtremePoints(x4Points, klines);
    x4Points = enforceMinBarDistance(x4Points, baseMinRetraceBars * 4);
    x4Points = validateAndCorrectExtremePoints(x4Points, klines);

    // Step 3.5: x4走势识别
    auto x4Trends = identifySameLevelTrendsWithKlines(x4Points, 4, klines);

    // Step 4: x4 → x8
    auto x8Points = deriveNextLevel(x4Points, baseMinRetraceBars * 8, 8, klines);
    x8Points = insertMissingThresholdPoints(x8Points, x4Points, baseMinRetraceBars * 8, 8, klines);
    x8Points = validateAndCorrectExtremePoints(x8Points, klines);
    x8Points = enforceMinBarDistance(x8Points, baseMinRetraceBars * 8);
    x8Points = validateAndCorrectExtremePoints(x8Points, klines);

    // 追加尾部追踪点
    x1Points = appendTrailingPoints(x1Points, klines, baseMinRetraceBars * 1, 1);
    x2Points = appendTrailingPoints(x2Points, klines, baseMinRetraceBars * 2, 2);
    x4Points = appendTrailingPoints(x4Points, klines, baseMinRetraceBars * 4, 4);
    x8Points = appendTrailingPoints(x8Points, klines, baseMinRetraceBars * 8, 8);

    // 收集标注点
    if (showSubX1 && !subX1Points.empty()) {
        result.markPoints.insert(result.markPoints.end(), subX1Points.begin(), subX1Points.end());
        result.activeLevels.push_back(0);
    }
    if (showLevel1x && !x1Points.empty()) {
        result.markPoints.insert(result.markPoints.end(), x1Points.begin(), x1Points.end());
        result.activeLevels.push_back(1);
    }
    if (showLevel2x && !x2Points.empty()) {
        result.markPoints.insert(result.markPoints.end(), x2Points.begin(), x2Points.end());
        result.activeLevels.push_back(2);
    }
    if (showLevel4x && !x4Points.empty()) {
        result.markPoints.insert(result.markPoints.end(), x4Points.begin(), x4Points.end());
        result.activeLevels.push_back(4);
    }
    if (showLevel8x && !x8Points.empty()) {
        result.markPoints.insert(result.markPoints.end(), x8Points.begin(), x8Points.end());
        result.activeLevels.push_back(8);
    }

    // 收集走势
    result.sameLevelTrends.insert(result.sameLevelTrends.end(), x1Trends.begin(), x1Trends.end());
    result.sameLevelTrends.insert(result.sameLevelTrends.end(), x2Trends.begin(), x2Trends.end());
    result.sameLevelTrends.insert(result.sameLevelTrends.end(), x4Trends.begin(), x4Trends.end());

    // 收集分型标记（从sub-x1点中提取）
    for (const auto& mp : subX1Points) {
        FractalMarker fm;
        fm.index = mp.index;
        fm.timestamp = mp.timestamp;
        fm.price = mp.price;
        fm.type = (mp.type == PointType::H) ? "top" : "bottom";
        
        // 查找相邻K线确定区间
        if (mp.index > 0 && mp.index < static_cast<int>(klines.size())) {
            if (mp.type == PointType::H) {
                fm.zoneHigh = mp.price;
                fm.zoneLow = klines[mp.index].low;
            } else {
                fm.zoneLow = mp.price;
                fm.zoneHigh = klines[mp.index].high;
            }
        }
        result.fractalMarkers.push_back(fm);
    }

    return result;
}

} // namespace moshi
