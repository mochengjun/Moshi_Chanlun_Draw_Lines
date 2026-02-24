#pragma once

#include <cstdint>
#include <string>
#include <vector>
#include <optional>

namespace moshi {

// ============================================================================
// K线数据结构
// ============================================================================

struct KLine {
    std::string timestamp;
    double open   = 0.0;
    double high   = 0.0;
    double low    = 0.0;
    double close  = 0.0;
    double volume = 0.0;
};

// ============================================================================
// 标注点
// ============================================================================

enum class PointType { L, H };

struct MarkPoint {
    PointType   type       = PointType::L;
    int         index      = 0;       // 原始K线索引
    std::string timestamp;
    double      price      = 0.0;
    std::string level;                // "sub-x1", "x1", "x2", "x4", "x8"
    int         multiplier = 0;       // 0/1/2/4/8
};

// ============================================================================
// 分型标记（用于前端显示）
// ============================================================================

struct FractalMarker {
    int         index      = 0;       // 原始K线索引
    std::string timestamp;
    std::string type;                 // "top" / "bottom"
    double      price      = 0.0;
    double      zoneLow    = 0.0;
    double      zoneHigh   = 0.0;
};

// ============================================================================
// 无包含关系的合并K线
// ============================================================================

struct MergedKLine {
    double      high          = 0.0;
    double      low           = 0.0;
    int         highOrigIdx   = 0;   // High来自的原始K线索引
    int         lowOrigIdx    = 0;   // Low来自的原始K线索引
    std::string highTimestamp;
    std::string lowTimestamp;
};

// ============================================================================
// 同级别走势
// ============================================================================

struct SameLevelTrend {
    std::string type;                 // "up" / "down"
    std::string pattern;              // "trend" / "convergent" / "divergent"
    int         multiplier    = 0;    // 级别: 1(x1), 2(x2), 4(x4), 8(x8)
    int         startIndex    = 0;
    int         endIndex      = 0;
    std::string startTimestamp;
    std::string endTimestamp;
    MarkPoint   highPoint;
    MarkPoint   lowPoint;
    std::vector<MarkPoint> points;
    bool        upgraded      = false;
    std::vector<MarkPoint> parentPoints;
};

// ============================================================================
// 指标计算结果
// ============================================================================

struct IndicatorResult {
    std::string type = "moshi_chanlun";
    std::string name = "莫氏缠论";
    std::vector<MarkPoint>      markPoints;
    std::vector<SameLevelTrend> sameLevelTrends;
    std::vector<int>            activeLevels;
    std::vector<FractalMarker>  fractalMarkers;  // 顶底分型标记
    int                         klineType = 10;
};

// ============================================================================
// 级别阈值配置
// ============================================================================

/// 根据K线周期返回x1级别的最小回调K线根数
/// 各级别阈值 = baseMinRetraceBars * multiplier
inline int getMinRetraceBars(int klineType) {
    switch (klineType) {
        case 1:  return 4;   // 1分钟
        case 2:  return 6;   // 5分钟
        case 4:  return 5;   // 3分钟
        case 5:  return 16;  // 15分钟
        case 6:  return 8;   // 30分钟
        case 3:  return 4;   // 60分钟
        case 8:  return 4;   // 120分钟
        case 7:  return 5;   // 半日线
        case 10: return 5;   // 日K
        case 11: return 8;   // 周K
        case 20: return 15;  // 月K
        case 21: return 5;   // 季K
        case 30: return 10;  // 年K
        default: return 5;
    }
}

/// 根据multiplier返回级别名称
inline std::string getLevelName(int multiplier) {
    switch (multiplier) {
        case 0: return "sub-x1";
        case 1: return "x1";
        case 2: return "x2";
        case 4: return "x4";
        case 8: return "x8";
        default: return "unknown";
    }
}

} // namespace moshi
