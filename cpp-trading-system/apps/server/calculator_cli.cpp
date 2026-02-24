/// calculator_cli - C++ CLI tool for Moshi Chanlun indicator calculation
/// Reads JSON from stdin, computes indicators, outputs JSON to stdout.
///
/// Input JSON:
/// {
///   "klines": [{"timestamp":"...", "open":..., "high":..., "low":..., "close":..., "volume":...}, ...],
///   "params": {"kline_type": 10, "show_level_sub_x1": 1, ...}
/// }
///
/// Output JSON: IndicatorResult in frontend-compatible format

#include "indicator/moshi_chanlun/moshi_chanlun.h"
#include "json.hpp"

#include <iostream>
#include <string>
#include <sstream>
#include <chrono>
#include <map>
#include <vector>
#include <algorithm>

using json = nlohmann::json;
using namespace moshi;

// ============================================================================
// JSON serialization helpers
// ============================================================================

static json markPointToJson(const MarkPoint& mp) {
    return {
        {"type", mp.type == PointType::H ? "H" : "L"},
        {"index", mp.index},
        {"timestamp", mp.timestamp},
        {"price", mp.price},
        {"level", mp.level},
        {"multiplier", mp.multiplier}
    };
}

static json sameLevelTrendToJson(const SameLevelTrend& t) {
    json pts = json::array();
    for (const auto& p : t.points) {
        pts.push_back(markPointToJson(p));
    }

    json result = {
        {"type", t.type},
        {"pattern", t.pattern},
        {"multiplier", t.multiplier},
        {"start_index", t.startIndex},
        {"end_index", t.endIndex},
        {"start_timestamp", t.startTimestamp},
        {"end_timestamp", t.endTimestamp},
        {"high_point", markPointToJson(t.highPoint)},
        {"low_point", markPointToJson(t.lowPoint)},
        {"points", pts},
        {"upgraded", t.upgraded}
    };

    if (!t.parentPoints.empty()) {
        json pp = json::array();
        for (const auto& p : t.parentPoints) {
            pp.push_back(markPointToJson(p));
        }
        result["parent_points"] = pp;
    }

    return result;
}

/// Generate bi_markers from mark_points.
/// Groups points by multiplier, then creates line segments between consecutive points.
static json generateBiMarkers(const std::vector<MarkPoint>& markPoints,
                               const std::vector<KLine>& klines) {
    // Group points by multiplier
    std::map<int, std::vector<const MarkPoint*>> grouped;
    for (const auto& mp : markPoints) {
        grouped[mp.multiplier].push_back(&mp);
    }

    json biMarkers = json::array();

    for (auto& [mult, pts] : grouped) {
        // Sort by index
        std::sort(pts.begin(), pts.end(),
                  [](const MarkPoint* a, const MarkPoint* b) { return a->index < b->index; });

        // Create bi_markers for consecutive point pairs
        for (size_t i = 0; i + 1 < pts.size(); ++i) {
            const auto& start = *pts[i];
            const auto& end = *pts[i + 1];

            // Calculate up/down count between start and end
            int upCount = 0, downCount = 0;
            int si = std::min(start.index, static_cast<int>(klines.size()) - 1);
            int ei = std::min(end.index, static_cast<int>(klines.size()) - 1);

            for (int k = si; k <= ei && k < static_cast<int>(klines.size()); ++k) {
                if (klines[k].close >= klines[k].open)
                    ++upCount;
                else
                    ++downCount;
            }

            bool isUp = (start.type == PointType::L);
            int length = (ei - si > 0) ? (ei - si) : 0;

            json bi = {
                {"start_index", start.index},
                {"end_index", end.index},
                {"start_timestamp", start.timestamp},
                {"end_timestamp", end.timestamp},
                {"start_price", start.price},
                {"end_price", end.price},
                {"direction", isUp ? "UP" : "DOWN"},
                {"multiplier", mult},
                {"up_count", upCount},
                {"down_count", downCount},
                {"length", length}
            };
            biMarkers.push_back(bi);
        }
    }

    return biMarkers;
}

// ============================================================================
// Main
// ============================================================================
int main() {
    // Read all of stdin
    std::ostringstream ss;
    ss << std::cin.rdbuf();
    std::string input = ss.str();

    if (input.empty()) {
        json err = {{"error", "Empty input"}};
        std::cout << err.dump() << std::endl;
        return 1;
    }

    try {
        json j = json::parse(input);

        // Parse klines
        std::vector<KLine> klines;
        for (const auto& kj : j["klines"]) {
            KLine k;
            k.timestamp = kj["timestamp"].get<std::string>();
            k.open = kj["open"].get<double>();
            k.high = kj["high"].get<double>();
            k.low = kj["low"].get<double>();
            k.close = kj["close"].get<double>();
            k.volume = kj.value("volume", 0.0);
            klines.push_back(k);
        }

        // Parse params
        std::map<std::string, double> params;
        if (j.contains("params")) {
            for (auto& [key, val] : j["params"].items()) {
                if (val.is_number()) {
                    params[key] = val.get<double>();
                } else if (val.is_boolean()) {
                    params[key] = val.get<bool>() ? 1.0 : 0.0;
                }
            }
        }

        // Run calculation
        auto t0 = std::chrono::high_resolution_clock::now();

        MoshiChanlunCalculator calculator;
        IndicatorResult result = calculator.calculate(klines, params);

        auto t1 = std::chrono::high_resolution_clock::now();
        double elapsedMs = std::chrono::duration<double, std::milli>(t1 - t0).count();

        // Build output JSON in frontend-compatible format
        json markPointsJson = json::array();
        for (const auto& mp : result.markPoints) {
            markPointsJson.push_back(markPointToJson(mp));
        }

        json trendsJson = json::array();
        for (const auto& t : result.sameLevelTrends) {
            trendsJson.push_back(sameLevelTrendToJson(t));
        }

        json biMarkers = generateBiMarkers(result.markPoints, klines);

        json output = {
            {"type", result.type},
            {"name", result.name},
            {"bi_markers", biMarkers},
            {"extra", {
                {"mark_points", markPointsJson},
                {"kline_type", result.klineType},
                {"levels", result.activeLevels},
                {"same_level_trends", trendsJson}
            }},
            {"computation_time_ms", static_cast<int>(elapsedMs + 0.5)}
        };

        std::cout << output.dump() << std::endl;
        return 0;

    } catch (const std::exception& e) {
        json err = {{"error", std::string("Calculation error: ") + e.what()}};
        std::cout << err.dump() << std::endl;
        return 1;
    }
}
