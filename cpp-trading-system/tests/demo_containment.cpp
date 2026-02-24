/// @file demo_containment.cpp
/// @brief 莫氏缠论K线包含关系处理和顶底分型识别演示程序
/// 
/// 展示内容：
/// 1. 原始K线数据
/// 2. 包含关系处理后的K线序列
/// 3. 识别出的顶底分型
/// 4. 各级别标注点（sub-x1, x1, x2, x4, x8）

#include "indicator/moshi_chanlun/moshi_chanlun.h"
#include <iostream>
#include <iomanip>
#include <vector>
#include <string>

using namespace moshi;
using namespace std;

// 辅助函数：创建K线
static KLine makeKLine(const string& timestamp, double open, double high, double low, double close, double volume = 1000000) {
    return {timestamp, open, high, low, close, volume};
}

// 打印分隔线
void printSeparator(const string& title) {
    cout << "\n" << string(60, '=') << "\n";
    cout << "  " << title << "\n";
    cout << string(60, '=') << "\n";
}

// 打印原始K线数据
void printOriginalKLines(const vector<KLine>& klines) {
    cout << "\n┌─────┬────────────┬─────────┬─────────┬─────────┬─────────┬─────────┐\n";
    cout << "│ 序号 │   时间戳   │   开盘  │   最高  │   最低  │   收盘  │ 成交量  │\n";
    cout << "├─────┼────────────┼─────────┼─────────┼─────────┼─────────┼─────────┤\n";
    
    for (size_t i = 0; i < klines.size(); ++i) {
        const auto& k = klines[i];
        cout << "│" << setw(5) << i 
             << " │" << setw(10) << k.timestamp 
             << " │" << setw(7) << fixed << setprecision(2) << k.open
             << " │" << setw(7) << k.high
             << " │" << setw(7) << k.low
             << " │" << setw(7) << k.close
             << " │" << setw(7) << (int)k.volume << " │\n";
    }
    cout << "└─────┴────────────┴─────────┴─────────┴─────────┴─────────┴─────────┘\n";
}

// 打印包含关系分析
void printContainmentAnalysis(const vector<KLine>& klines) {
    cout << "\n【包含关系分析】\n";
    for (size_t i = 1; i < klines.size(); ++i) {
        const auto& prev = klines[i-1];
        const auto& curr = klines[i];
        
        bool hasContainment = MoshiChanlunCalculator::hasContainment(
            prev.high, prev.low, curr.high, curr.low);
        
        string relation = "无包含";
        if (hasContainment) {
            if (curr.high >= prev.high && curr.low <= prev.low) {
                relation = "后包含前 (K" + to_string(i) + " 包含 K" + to_string(i-1) + ")";
            } else {
                relation = "前包含后 (K" + to_string(i-1) + " 包含 K" + to_string(i) + ")";
            }
        }
        
        cout << "  K" << (i-1) << " vs K" << i << ": " << relation;
        if (hasContainment) cout << " ⚠️";
        cout << "\n";
    }
}

// 打印处理后的K线序列
void printMergedKLines(const vector<MergedKLine>& merged) {
    cout << "\n【包含关系处理后的K线序列】\n";
    cout << "┌─────┬────────────┬─────────┬─────────┬──────────────────────┐\n";
    cout << "│ 序号 │   时间戳   │   高点  │   低点  │    原始K线索引范围    │\n";
    cout << "├─────┼────────────┼─────────┼─────────┼──────────────────────┤\n";
    
    for (size_t i = 0; i < merged.size(); ++i) {
        const auto& m = merged[i];
        string idxRange = "[" + to_string(m.highOrigIdx) + ", " + to_string(m.lowOrigIdx) + "]";
        
        cout << "│" << setw(5) << i 
             << " │" << setw(10) << m.timestamp 
             << " │" << setw(7) << fixed << setprecision(2) << m.high
             << " │" << setw(7) << m.low
             << " │" << setw(20) << idxRange << " │\n";
    }
    cout << "└─────┴────────────┴─────────┴─────────┴──────────────────────┘\n";
}

// 打印分型识别结果
void printFractals(const vector<MarkPoint>& fractals, const vector<KLine>& klines) {
    cout << "\n【顶底分型识别结果】\n";
    
    if (fractals.empty()) {
        cout << "  (无分型识别结果)\n";
        return;
    }
    
    cout << "┌─────┬────────────┬────────┬─────────┬─────────┬─────────┬────────────┐\n";
    cout << "│ 序号 │   时间戳   │ 类型   │  价格   │ K线索引 │  级别   │  判断逻辑  │\n";
    cout << "├─────┼────────────┼────────┼─────────┼─────────┼─────────┼────────────┤\n";
    
    for (size_t i = 0; i < fractals.size(); ++i) {
        const auto& f = fractals[i];
        string type = (f.type == PointType::H) ? "顶分型" : "底分型";
        string logic = (f.type == PointType::H) ? "mid.H > left.H && mid.H > right.H" 
                                                   : "mid.L < left.L && mid.L < right.L";
        
        cout << "│" << setw(5) << i 
             << " │" << setw(10) << f.timestamp 
             << " │" << setw(6) << type
             << " │" << setw(7) << fixed << setprecision(2) << f.price
             << " │" << setw(7) << f.index
             << " │" << setw(7) << f.level
             << " │" << setw(10) << " " << "│\n";
        
        // 打印判断逻辑
        if (i == 0 || fractals[i].type != fractals[i-1].type) {
            cout << "│     │            │        │         │         │         │ " << setw(10) << logic << "│\n";
        }
    }
    cout << "└─────┴────────────┴────────┴─────────┴─────────┴─────────┴────────────┘\n";
    
    cout << "\n  判断逻辑说明:\n";
    cout << "    - 顶分型: 中间K线高点 > 左侧K线高点 且 中间K线高点 > 右侧K线高点\n";
    cout << "    - 底分型: 中间K线低点 < 左侧K线低点 且 中间K线低点 < 右侧K线低点\n";
}

// 打印各级别标注点
void printMarkPointsByLevel(const vector<MarkPoint>& points) {
    if (points.empty()) {
        cout << "\n【各级别标注点】\n  (无标注点)\n";
        return;
    }
    
    // 按级别分组
    map<int, vector<const MarkPoint*>> grouped;
    for (const auto& p : points) {
        grouped[p.multiplier].push_back(&p);
    }
    
    cout << "\n【各级别标注点】\n";
    
    for (const auto& [level, pts] : grouped) {
        string levelName = getLevelName(level);
        
        cout << "\n--- " << levelName << " (级别倍数: " << level << ") ---\n";
        cout << "┌─────┬────────────┬────────┬─────────┬─────────┐\n";
        cout << "│ 序号 │   时间戳   │ 类型   │  价格   │ K线索引 │\n";
        cout << "├─────┼────────────┼────────┼─────────┼─────────┤\n";
        
        for (size_t i = 0; i < pts.size(); ++i) {
            const auto* p = pts[i];
            string type = (p->type == PointType::H) ? "高点(H)" : "低点(L)";
            
            cout << "│" << setw(5) << i 
                 << " │" << setw(10) << p->timestamp 
                 << " │" << setw(6) << type
                 << " │" << setw(7) << fixed << setprecision(2) << p->price
                 << " │" << setw(7) << p->index << " │\n";
        }
        cout << "└─────┴────────────┴────────┴─────────┴─────────┘\n";
    }
}

// 打印完整计算结果
void printFullResult(const IndicatorResult& result) {
    printSeparator("莫氏缠论画线指标计算结果");
    
    cout << "\n【基本信息】\n";
    cout << "  类型: " << result.type << "\n";
    cout << "  名称: " << result.name << "\n";
    
    cout << "\n【活跃级别】\n";
    for (int level : result.activeLevels) {
        cout << "  - " << getLevelName(level) << " (倍数: " << level << ")\n";
    }
    
    printMarkPointsByLevel(result.markPoints);
    
    cout << "\n【走势数量】: " << result.sameLevelTrends.size() << "\n";
}

int main() {
    cout << string(70, '*') << "\n";
    cout << "*                                                                     *\n";
    cout << "*         莫氏缠论 K线包含关系处理 & 顶底分型识别 演示程序            *\n";
    cout << "*                                                                     *\n";
    cout << string(70, '*') << "\n";
    
    // 创建测试数据 - 构造一段包含各种情况的K线序列
    // 场景：上涨后下跌再反弹，包含关系处理测试
    vector<KLine> klines = {
        // 上涨段
        makeKLine("2026-01-02", 3200, 3230, 3190, 3220),
        makeKLine("2026-01-03", 3220, 3260, 3210, 3250),
        makeKLine("2026-01-06", 3250, 3280, 3240, 3270),  // 包含关系测试
        makeKLine("2026-01-07", 3260, 3290, 3230, 3280),  // 被包含
        makeKLine("2026-01-08", 3280, 3320, 3270, 3310),
        makeKLine("2026-01-09", 3310, 3350, 3300, 3340),
        
        // 包含关系: K8(3320-3380) 包含 K7(3290-3330)
        makeKLine("2026-01-10", 3320, 3380, 3310, 3370),  // 包含前面
        makeKLine("2026-01-13", 3370, 3400, 3360, 3390),
        
        // 下跌段
        makeKLine("2026-01-14", 3390, 3410, 3340, 3350),
        makeKLine("2026-01-15", 3350, 3360, 3280, 3290),
        makeKLine("2026-01-16", 3290, 3300, 3240, 3250),
        makeKLine("2026-01-17", 3250, 3260, 3200, 3210),
        makeKLine("2026-01-20", 3210, 3220, 3170, 3180),
        
        // 反弹段
        makeKLine("2026-01-21", 3180, 3190, 3140, 3150),
        makeKLine("2026-01-22", 3150, 3190, 3140, 3180),  // 包含
        makeKLine("2026-01-23", 3160, 3200, 3150, 3190),  // 被包含
        
        // 继续上涨
        makeKLine("2026-01-24", 3190, 3230, 3180, 3220),
        makeKLine("2026-01-27", 3220, 3260, 3210, 3250),
        makeKLine("2026-01-28", 3250, 3290, 3240, 3280),
        makeKLine("2026-01-29", 3280, 3320, 3270, 3310),
        makeKLine("2026-01-30", 3310, 3350, 3300, 3340),
    };
    
    printSeparator("第一步：原始K线数据");
    printOriginalKLines(klines);
    
    // 创建计算器实例
    MoshiChanlunCalculator calculator;
    
    // 步骤1: 包含关系处理
    printSeparator("第二步：K线包含关系处理");
    printContainmentAnalysis(klines);
    
    vector<MergedKLine> merged = calculator.removeContainment(klines);
    printMergedKLines(merged);
    
    // 步骤2: 顶底分型识别
    printSeparator("第三步：顶底分型识别");
    vector<MarkPoint> fractals = calculator.identifyFractals(merged, klines);
    printFractals(fractals, klines);
    
    // 步骤3: 计算完整莫氏缠论指标
    printSeparator("第四步：完整莫氏缠论指标计算");
    
    map<string, double> params = {
        {"kline_type", 10},      // 日K
        {"show_level_sub_x1", 1},
        {"show_level_1x", 1},
        {"show_level_2x", 1},
        {"show_level_4x", 1},
        {"show_level_8x", 1},
    };
    
    IndicatorResult result = calculator.calculate(klines, params);
    printFullResult(result);
    
    printSeparator("演示结束");
    
    return 0;
}
