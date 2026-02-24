package moshi

import (
	"kline-indicator-service/internal/calculator"
	"kline-indicator-service/internal/models"
	"sort"
	"time"
)

func init() {
	calculator.Register(models.IndicatorTypeMoshiChanlun, NewMoshiChanlunCalculator)
}

// 走势级别定义
type TrendLevel struct {
	Name       string  // 级别名称
	MinMinutes float64 // 最小时间(分钟)
	MaxMinutes float64 // 最大时间(分钟)
	MinKLines  int     // 最小K线数
	MaxKLines  int     // 最大K线数
	BasePeriod string  // 基础周期
}

// 标注点类型
type PointType string

const (
	PointL PointType = "L" // 低点
	PointH PointType = "H" // 高点
)

// 标注点
type MarkPoint struct {
	Type       PointType `json:"type"`       // 点类型 L/H
	Index      int       `json:"index"`      // K线索引
	Timestamp  string    `json:"timestamp"`  // 时间
	Price      float64   `json:"price"`      // 价格
	Level      string    `json:"level"`      // 所属级别
	Multiplier int       `json:"multiplier"` // 时间倍数 1/2/4/8
}

// SameLevelTrend 同级别走势（适用于x1/x2/x4/x8各级别）
// 走势类型(Type): "up"(上涨), "down"(下跌)
// 形态类型(Pattern): "trend"(趋势), "convergent"(收敛型中枢), "divergent"(扩张型中枢)
type SameLevelTrend struct {
	Type           string      `json:"type"`            // "up" 或 "down"
	Pattern        string      `json:"pattern"`         // "trend"(趋势), "convergent"(收敛), "divergent"(扩张)
	Multiplier     int         `json:"multiplier"`      // 走势所属级别: 1(x1), 2(x2), 4(x4), 8(x8)
	StartIndex     int         `json:"start_index"`     // 起始K线索引
	EndIndex       int         `json:"end_index"`       // 结束K线索引
	StartTimestamp string      `json:"start_timestamp"` // 起始时间
	EndTimestamp   string      `json:"end_timestamp"`   // 结束时间
	HighPoint      MarkPoint   `json:"high_point"`      // 走势最高点（上涨=最后H，下跌=第一H）
	LowPoint       MarkPoint   `json:"low_point"`       // 走势最低点（上涨=第一L，下跌=最后L）
	Points         []MarkPoint `json:"points"`          // 组成走势的所有L/H点序列
	// 级别升级相关字段
	Upgraded       bool        `json:"upgraded"`                  // 是否已升级为父级别
	ParentPoints   []MarkPoint `json:"parent_points,omitempty"`   // 升级后的父级别点序列
}

// 走势级别完全分类表
var TrendLevels = []TrendLevel{
	// 1分钟周期
	{"小1分钟", 5, 8, 5, 8, "1min"},
	{"中1分钟", 8, 15, 8, 15, "1min"},
	{"大1分钟", 15, 30, 15, 30, "1min"},
	// 5分钟周期
	{"小5分钟", 30, 60, 6, 12, "5min"},
	{"中5分钟", 60, 120, 12, 24, "5min"},
	{"大5分钟", 120, 240, 24, 48, "5min"},
	// 30分钟周期
	{"小30分钟", 240, 480, 8, 16, "30min"},
	{"中30分钟", 480, 960, 16, 32, "30min"},
	{"大30分钟", 960, 1920, 32, 64, "30min"},
	// 日线周期
	{"小日线", 5 * 24 * 60, 10 * 24 * 60, 5, 10, "day"},
	{"中日线", 10 * 24 * 60, 20 * 24 * 60, 10, 20, "day"},
	{"大日线", 20 * 24 * 60, 40 * 24 * 60, 20, 40, "day"},
	// 周线周期
	{"小周线", 8 * 7 * 24 * 60, 16 * 7 * 24 * 60, 8, 16, "week"},
	{"中周线", 16 * 7 * 24 * 60, 32 * 7 * 24 * 60, 16, 32, "week"},
	{"大周线", 32 * 7 * 24 * 60, 64 * 7 * 24 * 60, 32, 64, "week"},
	// 月线周期
	{"小月线", 15 * 30 * 24 * 60, 30 * 30 * 24 * 60, 15, 30, "month"},
	{"中月线", 30 * 30 * 24 * 60, 60 * 30 * 24 * 60, 30, 60, "month"},
	{"大月线", 60 * 30 * 24 * 60, 120 * 30 * 24 * 60, 60, 120, "month"},
	// 年线周期
	{"年线", 10 * 365 * 24 * 60, 20 * 365 * 24 * 60, 10, 20, "year"},
}

// 获取单根K线的时间跨度(分钟)
func getKLineDurationMinutes(klineType int) float64 {
	switch klineType {
	case 1: // 1分钟
		return 1
	case 4: // 3分钟
		return 3
	case 2: // 5分钟
		return 5
	case 5: // 15分钟
		return 15
	case 6: // 30分钟
		return 30
	case 3: // 60分钟
		return 60
	case 8: // 120分钟
		return 120
	case 7: // 半日线
		return 4 * 60 // 假设半日线=4小时
	case 10: // 日K
		return 24 * 60
	case 11: // 周K
		return 7 * 24 * 60
	case 20: // 月K
		return 30 * 24 * 60
	default:
		return 5 // 默认5分钟
	}
}

// 根据K线周期获取最小级别回调时间(分钟)
// klineType值基于外部API reqtype=150协议实际验证
func getMinRetraceMinutes(klineType int) float64 {
	switch klineType {
	case 1: // 1分钟
		return 5
	case 4: // 3分钟
		return 15
	case 2: // 5分钟
		return 30
	case 5: // 15分钟
		return 60
	case 6: // 30分钟
		return 240
	case 3: // 60分钟
		return 480
	case 8: // 120分钟
		return 960
	case 7: // 半日线
		return 5 * 24 * 60
	case 10: // 日K
		return 5 * 24 * 60
	case 11: // 周K
		return 8 * 7 * 24 * 60
	case 20: // 月K
		return 15 * 30 * 24 * 60
	default:
		return 30 // 默认30分钟
	}
}

// getMinRetraceBars 根据K线周期返回x1级别的最小回调K线根数
// 新规则：使用K线根数代替时间作为级别判断阈值
// klineType值基于外部API reqtype=150协议实际验证
func getMinRetraceBars(klineType int) int {
	switch klineType {
	case 1: // 1分钟
		return 4
	case 2: // 5分钟
		return 6
	case 4: // 3分钟
		return 5 // 插值
	case 5: // 15分钟
		return 16
	case 6: // 30分钟
		return 8
	case 3: // 60分钟
		return 4
	case 8: // 120分钟
		return 4 // 插值
	case 7: // 半日线
		return 5 // 插值
	case 10: // 日K
		return 5
	case 11: // 周K
		return 8
	case 20: // 月K
		return 15
	case 21: // 季K
		return 5
	case 30: // 年K
		return 10
	default:
		return 5 // 默认5根K线
	}
}

// 根据时间确定走势级别
func determineTrendLevel(minutes float64) string {
	for _, level := range TrendLevels {
		if minutes > level.MinMinutes && minutes <= level.MaxMinutes {
			return level.Name
		}
	}
	return "未知级别"
}

// getLevelNameByMultiplier 根据multiplier返回级别名称
// multiplier: 1=x1, 2=x2, 4=x4, 8=x8
func getLevelNameByMultiplier(multiplier int) string {
	switch multiplier {
	case 0:
		return "sub-x1"
	case 1:
		return "x1"
	case 2:
		return "x2"
	case 4:
		return "x4"
	case 8:
		return "x8"
	default:
		return "未知级别"
	}
}

// MoshiChanlunCalculator 莫氏缠论指标计算器
type MoshiChanlunCalculator struct{}

// NewMoshiChanlunCalculator 创建莫氏缠论计算器
func NewMoshiChanlunCalculator() calculator.Indicator {
	return &MoshiChanlunCalculator{}
}

// Metadata 返回指标元数据
func (c *MoshiChanlunCalculator) Metadata() models.IndicatorMetadata {
	return models.IndicatorMetadata{
		Type:        models.IndicatorTypeMoshiChanlun,
		Name:        "莫氏缠论",
		Category:    models.CategoryChanlun,
		Description: "莫氏缠论画线指标，基于层级递推（sub-x1→x1→x2→x4→x8）识别走势级别和关键转折点(L/H标注点)，支持多级别显示",
		ParamsDef: []models.ParameterDef{
			{
				Name:         "kline_type",
				Type:         "int",
				Required:     false,
				DefaultValue: 10,
				Min:          1,
				Max:          30,
				Description:  "K线类型(1:1分,4:3分,2:5分,5:15分,6:30分,3:60分,8:120分,7:半日线,10:日K,11:周K,20:月K)",
			},
			{
				Name:         "show_level_sub_x1",
				Type:         "bool",
				Required:     false,
				DefaultValue: false,
				Description:  "显示sub-x1级别标注点（基准时间=2根K线）",
			},
			{
				Name:         "show_level_1x",
				Type:         "bool",
				Required:     false,
				DefaultValue: true,
				Description:  "显示1倍级别标注点",
			},
			{
				Name:         "show_level_2x",
				Type:         "bool",
				Required:     false,
				DefaultValue: true,
				Description:  "显示2倍级别标注点",
			},
			{
				Name:         "show_level_4x",
				Type:         "bool",
				Required:     false,
				DefaultValue: true,
				Description:  "显示4倍级别标注点",
			},
			{
				Name:         "show_level_8x",
				Type:         "bool",
				Required:     false,
				DefaultValue: true,
				Description:  "显示8倍级别标注点",
			},
		},
	}
}

// Validate 验证参数
func (c *MoshiChanlunCalculator) Validate(params map[string]interface{}) error {
	// 莫氏缠论参数都有默认值，无需强制验证
	return nil
}

// Calculate 计算莫氏缠论标注点
func (c *MoshiChanlunCalculator) Calculate(klines []models.KLine, params map[string]interface{}) (*models.IndicatorResult, error) {
	if len(klines) < 3 {
		return &models.IndicatorResult{
			Type: models.IndicatorTypeMoshiChanlun,
			Name: "莫氏缠论",
		}, nil
	}

	// 获取参数
	klineType := getIntParam(params, "kline_type", 10)
	showLevelSubX1 := getBoolParam(params, "show_level_sub_x1", false)
	showLevel1x := getBoolParam(params, "show_level_1x", true)
	showLevel2x := getBoolParam(params, "show_level_2x", true)
	showLevel4x := getBoolParam(params, "show_level_4x", true)
	showLevel8x := getBoolParam(params, "show_level_8x", true)

	// 基础回调K线根数（新规则：使用K线根数代替时间阈值）
	baseMinRetraceBars := getMinRetraceBars(klineType)

	// === 层级递推计算 ===
	// Step 0: 计算sub-x1级别（基准=2根K线）
	subX1Points := c.calculateSubLevelPoints(klines, klineType)

	// Step 1: 从sub-x1推导x1级别，检测超阈值段，验证极值点，强制最小间距
	x1PointsRaw := c.deriveNextLevel(subX1Points, baseMinRetraceBars*1, 1, klines)
	x1PointsRaw = c.insertMissingThresholdPoints(x1PointsRaw, subX1Points, baseMinRetraceBars*1, 1, klines)
	x1Points := c.validateAndCorrectExtremePoints(x1PointsRaw, klines)
	x1Points = c.enforceMinBarDistance(x1Points, baseMinRetraceBars*1)
	x1Points = c.validateAndCorrectExtremePoints(x1Points, klines) // 重新验证：enforceMinBarDistance可能改变相邻H点，需确保L点仍是区间最低

	// Step 1.5: 识别x1同级别走势（带K线极值验证，仅用于前端可视化和形态分析）
	x1Trends := c.identifySameLevelTrendsWithKlines(x1Points, 1, klines)

	// Step 2: 从x1推导x2级别（使用已验证的x1Points），检测超阈值段，验证极值点，强制最小间距
	x2PointsRaw := c.deriveNextLevel(x1Points, baseMinRetraceBars*2, 2, klines)
	x2PointsRaw = c.insertMissingThresholdPoints(x2PointsRaw, x1Points, baseMinRetraceBars*2, 2, klines)
	x2Points := c.validateAndCorrectExtremePoints(x2PointsRaw, klines)
	x2Points = c.enforceMinBarDistance(x2Points, baseMinRetraceBars*2)
	x2Points = c.validateAndCorrectExtremePoints(x2Points, klines) // 重新验证：enforceMinBarDistance可能改变相邻H点，需确保L点仍是区间最低

	// Step 2.5: 识别x2同级别走势（带K线极值验证，仅用于前端可视化和形态分析）
	x2Trends := c.identifySameLevelTrendsWithKlines(x2Points, 2, klines)

	// Step 3: 从x2推导x4级别（使用已验证的x2Points），检测超阈值段，验证极值点，强制最小间距
	x4PointsRaw := c.deriveNextLevel(x2Points, baseMinRetraceBars*4, 4, klines)
	x4PointsRaw = c.insertMissingThresholdPoints(x4PointsRaw, x2Points, baseMinRetraceBars*4, 4, klines)
	x4Points := c.validateAndCorrectExtremePoints(x4PointsRaw, klines)
	x4Points = c.enforceMinBarDistance(x4Points, baseMinRetraceBars*4)
	x4Points = c.validateAndCorrectExtremePoints(x4Points, klines) // 重新验证：enforceMinBarDistance可能改变相邻H点，需确保L点仍是区间最低

	// Step 3.5: 识别x4同级别走势（带K线极值验证，仅用于前端可视化和形态分析）
	x4Trends := c.identifySameLevelTrendsWithKlines(x4Points, 4, klines)

	// Step 4: 从x4推导x8级别（使用已验证的x4Points），检测超阈值段，验证极值点，强制最小间距
	x8PointsRaw := c.deriveNextLevel(x4Points, baseMinRetraceBars*8, 8, klines)
	x8PointsRaw = c.insertMissingThresholdPoints(x8PointsRaw, x4Points, baseMinRetraceBars*8, 8, klines)
	x8Points := c.validateAndCorrectExtremePoints(x8PointsRaw, klines)
	x8Points = c.enforceMinBarDistance(x8Points, baseMinRetraceBars*8)
	x8Points = c.validateAndCorrectExtremePoints(x8Points, klines) // 重新验证：enforceMinBarDistance可能改变相邻H点，需确保L点仍是区间最低

	// 追加尾部追踪点：确保各级别标注覆盖到最新K线附近
	// 当管道处理（尤其是enforceMinBarDistance）移除了尾部点时，
	// 需要从K线数据中重新扫描并追加未确认的极值点，以保证图表右侧有标注
	x1Points = c.appendTrailingPoints(x1Points, klines, baseMinRetraceBars*1, 1)
	x2Points = c.appendTrailingPoints(x2Points, klines, baseMinRetraceBars*2, 2)
	x4Points = c.appendTrailingPoints(x4Points, klines, baseMinRetraceBars*4, 4)
	x8Points = c.appendTrailingPoints(x8Points, klines, baseMinRetraceBars*8, 8)

	// 收集所有需要显示的标注点
	allMarkPoints := make([]MarkPoint, 0)
	activeLevels := []int{}

	if showLevelSubX1 && len(subX1Points) > 0 {
		allMarkPoints = append(allMarkPoints, subX1Points...)
		activeLevels = append(activeLevels, 0)
	}
	if showLevel1x && len(x1Points) > 0 {
		allMarkPoints = append(allMarkPoints, x1Points...)
		activeLevels = append(activeLevels, 1)
	}
	if showLevel2x && len(x2Points) > 0 {
		allMarkPoints = append(allMarkPoints, x2Points...)
		activeLevels = append(activeLevels, 2)
	}
	if showLevel4x && len(x4Points) > 0 {
		allMarkPoints = append(allMarkPoints, x4Points...)
		activeLevels = append(activeLevels, 4)
	}
	if showLevel8x && len(x8Points) > 0 {
		allMarkPoints = append(allMarkPoints, x8Points...)
		activeLevels = append(activeLevels, 8)
	}

	// 合并所有级别的走势
	allTrends := make([]SameLevelTrend, 0)
	allTrends = append(allTrends, x1Trends...)
	allTrends = append(allTrends, x2Trends...)
	allTrends = append(allTrends, x4Trends...)

	// 构建结果
	result := &models.IndicatorResult{
		Type: models.IndicatorTypeMoshiChanlun,
		Name: "莫氏缠论",
		Extra: map[string]interface{}{
			"mark_points":       allMarkPoints,
			"kline_type":        klineType,
			"levels":            activeLevels,
			"same_level_trends": allTrends, // 所有级别的走势
		},
	}

	// 生成分型标记用于图表显示
	fractalMarkers := make([]models.FractalMarker, 0, len(allMarkPoints))
	for _, mp := range allMarkPoints {
		markerType := "bottom"
		if mp.Type == PointH {
			markerType = "top"
		}
		fractalMarkers = append(fractalMarkers, models.FractalMarker{
			Index:     mp.Index,
			Timestamp: mp.Timestamp,
			Type:      markerType,
			Price:     mp.Price,
		})
	}
	result.FractalMarkers = fractalMarkers

	// 生成笔标记（连接相邻的L-H点）
	biMarkers := c.generateBiMarkers(allMarkPoints, klines)
	result.BiMarkers = biMarkers

	return result, nil
}

// calculateSubLevelPoints 计算sub-x1级别标注点（基准=2根K线）
func (c *MoshiChanlunCalculator) calculateSubLevelPoints(klines []models.KLine, klineType int) []MarkPoint {
	if len(klines) < 2 {
		return nil
	}

	// sub-x1级别固定使用2根K线作为阈值
	minRetraceBars := 2

	points := make([]MarkPoint, 0)

	// 第一个L点：第一根K线的最低点
	firstPoint := MarkPoint{
		Type:       PointL,
		Index:      0,
		Timestamp:  klines[0].Timestamp,
		Price:      klines[0].Low,
		Level:      "sub-x1",
		Multiplier: 0,
	}
	points = append(points, firstPoint)

	lastPointType := PointL
	lastLIndex := 0
	lastHIndex := -1

	tempHighIndex := 0
	tempLowIndex := 0

	for i := 1; i < len(klines); i++ {
		stateChanges := 0

		for {
			// Phase 1: 更新临时极值索引（仅首次处理该bar时执行，避免重评估覆盖已确认极值）
			if stateChanges == 0 {
				if lastPointType == PointL {
					if tempHighIndex == 0 || klines[i].High > klines[tempHighIndex].High {
						tempHighIndex = i
					}
				} else {
					if tempLowIndex == 0 || klines[i].Low < klines[tempLowIndex].Low {
						tempLowIndex = i
					}
				}
			}

			confirmed := false

			if lastPointType == PointL {
				// Phase 1.5: 双向突破检测
				// 当前bar既是最高点(tempHighIndex==i)又跌破了前一L点 → 同时注册H和L
				if tempHighIndex == i && klines[i].Low < klines[lastLIndex].Low && i > lastLIndex {
					// 按趋势顺序：先注册H（上升趋势顶部），再注册L（反转底部）
					points = append(points, MarkPoint{
						Type:       PointH,
						Index:      i,
						Timestamp:  klines[i].Timestamp,
						Price:      klines[i].High,
						Level:      "sub-x1",
						Multiplier: 0,
					})
					points = append(points, MarkPoint{
						Type:       PointL,
						Index:      i,
						Timestamp:  klines[i].Timestamp,
						Price:      klines[i].Low,
						Level:      "sub-x1",
						Multiplier: 0,
					})
					lastPointType = PointL
					lastLIndex = i
					if i+1 < len(klines) {
						tempHighIndex = i + 1
					} else {
						tempHighIndex = i
					}
					break // 双向突破已完整处理
				}

				// Phase 2: 常规H点确认
				conditionMet := false

				// 条件1：价格跌破前一L点
				if klines[i].Low < klines[lastLIndex].Low {
					conditionMet = true
				}

				// 条件2：回调K线根数超过2根
				if !conditionMet {
					barCount := i - tempHighIndex
					if barCount >= minRetraceBars {
						conditionMet = true
					}
				}

				if conditionMet && tempHighIndex > lastLIndex {
					points = append(points, MarkPoint{
						Type:       PointH,
						Index:      tempHighIndex,
						Timestamp:  klines[tempHighIndex].Timestamp,
						Price:      klines[tempHighIndex].High,
						Level:      "sub-x1",
						Multiplier: 0,
					})

					lastPointType = PointH
					lastHIndex = tempHighIndex
					tempLowIndex = tempHighIndex + 1
					if tempLowIndex >= len(klines) {
						tempLowIndex = len(klines) - 1
					}
					for j := tempLowIndex + 1; j <= i; j++ {
						if klines[j].Low < klines[tempLowIndex].Low {
							tempLowIndex = j
						}
					}
					confirmed = true
				}
			} else {
				// Phase 1.5: 双向突破检测（镜像）
				// 当前bar既是最低点(tempLowIndex==i)又突破了前一H点 → 同时注册L和H
				if tempLowIndex == i && klines[i].High > klines[lastHIndex].High && i > lastHIndex {
					// 按趋势顺序：先注册L（下降趋势底部），再注册H（反转顶部）
					points = append(points, MarkPoint{
						Type:       PointL,
						Index:      i,
						Timestamp:  klines[i].Timestamp,
						Price:      klines[i].Low,
						Level:      "sub-x1",
						Multiplier: 0,
					})
					points = append(points, MarkPoint{
						Type:       PointH,
						Index:      i,
						Timestamp:  klines[i].Timestamp,
						Price:      klines[i].High,
						Level:      "sub-x1",
						Multiplier: 0,
					})
					lastPointType = PointH
					lastHIndex = i
					if i+1 < len(klines) {
						tempLowIndex = i + 1
					} else {
						tempLowIndex = i
					}
					break // 双向突破已完整处理
				}

				// Phase 2: 常规L点确认
				conditionMet := false

				// 条件3：价格突破前一H点
				if klines[i].High > klines[lastHIndex].High {
					conditionMet = true
				}

				// 条件4：反弹K线根数超过2根
				if !conditionMet {
					barCount := i - tempLowIndex
					if barCount >= minRetraceBars {
						conditionMet = true
					}
				}

				if conditionMet && tempLowIndex > lastHIndex {
					points = append(points, MarkPoint{
						Type:       PointL,
						Index:      tempLowIndex,
						Timestamp:  klines[tempLowIndex].Timestamp,
						Price:      klines[tempLowIndex].Low,
						Level:      "sub-x1",
						Multiplier: 0,
					})

					lastPointType = PointL
					lastLIndex = tempLowIndex
					tempHighIndex = tempLowIndex + 1
					if tempHighIndex >= len(klines) {
						tempHighIndex = len(klines) - 1
					}
					for j := tempHighIndex + 1; j <= i; j++ {
						if klines[j].High > klines[tempHighIndex].High {
							tempHighIndex = j
						}
					}
					confirmed = true
				}
			}

			if confirmed {
				stateChanges++
				if stateChanges >= 2 {
					break // 每根bar最多产生2个转折点（1H+1L）
				}
				continue // 重评估当前bar在新状态下是否触发第二个转折
			}
			break // 无确认发生，结束当前bar处理
		}
	}

	// 追加尾部未确认候选点，确保画线覆盖到最新K线附近
	if lastPointType == PointL && tempHighIndex > lastLIndex && tempHighIndex < len(klines) {
		points = append(points, MarkPoint{
			Type:       PointH,
			Index:      tempHighIndex,
			Timestamp:  klines[tempHighIndex].Timestamp,
			Price:      klines[tempHighIndex].High,
			Level:      "sub-x1",
			Multiplier: 0,
		})
	} else if lastPointType == PointH && tempLowIndex > lastHIndex && tempLowIndex < len(klines) {
		points = append(points, MarkPoint{
			Type:       PointL,
			Index:      tempLowIndex,
			Timestamp:  klines[tempLowIndex].Timestamp,
			Price:      klines[tempLowIndex].Low,
			Level:      "sub-x1",
			Multiplier: 0,
		})
	}

	return points
}

// deriveNextLevel 从前一级别的H/L点推导出下一级别的H/L点
// prevPoints: 前一级别的已确认H/L点序列（必须H/L严格交替）
// minRetraceBars: 当前级别的基准回调K线根数
// multiplier: 当前级别倍数 (1/2/4/8)
// klines: 原始K线数据（用于参考）
//
// 级别识别规则：
//  1. barCount < minRetraceBars       → 同级别，合并到前一段
//  2. barCount >= minRetraceBars      → 不同级别，确认跳级
//
// 算法核心：
//   lastConfirmedType 记录最后确认的点类型。
//   candidate 追踪反方向（与 lastConfirmedType 相反）的极值候选点。
//   当遇到与 lastConfirmedType 同向的点时，测量从 candidate 到该点的K线根数。
//   若K线根数满足跳级条件，则确认 candidate 为当前级别转折点，
//   并将该同向点作为下一段的候选起点（因为它相对于新的 lastConfirmedType 是反方向）。
func (c *MoshiChanlunCalculator) deriveNextLevel(prevPoints []MarkPoint, minRetraceBars int, multiplier int, klines []models.KLine) []MarkPoint {
	if len(prevPoints) < 3 {
		return nil
	}

	level := getLevelNameByMultiplier(multiplier)
	result := make([]MarkPoint, 0)

	// 添加第一个点作为起点
	first := prevPoints[0]
	result = append(result, MarkPoint{
		Type:       first.Type,
		Index:      first.Index,
		Timestamp:  first.Timestamp,
		Price:      first.Price,
		Level:      level,
		Multiplier: multiplier,
	})

	lastConfirmedType := first.Type
	lastConfirmedIndex := first.Index // 追踪最后确认点的K线索引，用于回溯检测
	lastConfirmedPrice := first.Price // 追踪最后确认点的价格，用于推动段规则
	impulseAllowed := true            // 推动段/回调段交替：首个候选为推动段

	// candidate: 追踪反方向（与 lastConfirmedType 相反）的最佳极值候选
	// 当 lastConfirmedType=L 时，candidate 追踪最高的 H 点
	// 当 lastConfirmedType=H 时，candidate 追踪最低的 L 点
	type candidateInfo struct {
		point MarkPoint
		valid bool
	}
	candidate := candidateInfo{valid: false}

	for i := 1; i < len(prevPoints); i++ {
		pt := prevPoints[i]

		if pt.Type != lastConfirmedType {
			// 反方向点：更新候选极值
			if !candidate.valid {
				candidate = candidateInfo{point: pt, valid: true}
			} else {
				// 候选点即将被更极端的点替换，触发回溯检测
				isMoreExtreme := (pt.Type == PointH && pt.Price > candidate.point.Price) ||
					(pt.Type == PointL && pt.Price < candidate.point.Price)
				if isMoreExtreme {
					// 动态回溯检测：当趋势距离 >= minRetraceBars 时
					// 在lastConfirmed到当前点之间扫描是否存在被遗漏的回调段
					trendDistance := pt.Index - lastConfirmedIndex
					if trendDistance >= minRetraceBars {
						retroH, retroL := c.scanRetroactiveRetrace(
							prevPoints, lastConfirmedIndex, pt.Index,
							lastConfirmedType, minRetraceBars)

						// sub-level未找到时用K线数据后备扫描（结果对齐到prevPoints维护层级关系）
						if retroH == nil && retroL == nil {
							retroH, retroL = c.scanKLineRetrace(
								klines, lastConfirmedIndex, pt.Index,
								lastConfirmedType, minRetraceBars, prevPoints)
						}

						if retroH != nil && retroL != nil {
							// 找到合格回调，确认回调对
							if lastConfirmedType == PointL {
								// 上涨趋势：先确认H，再确认L
								result = append(result, MarkPoint{
									Type: PointH, Index: retroH.Index,
									Timestamp: retroH.Timestamp, Price: retroH.Price,
									Level: level, Multiplier: multiplier,
								})
								result = append(result, MarkPoint{
									Type: PointL, Index: retroL.Index,
									Timestamp: retroL.Timestamp, Price: retroL.Price,
									Level: level, Multiplier: multiplier,
								})
								lastConfirmedType = PointL
								lastConfirmedIndex = retroL.Index
								lastConfirmedPrice = retroL.Price
								impulseAllowed = true // 回溯对确认后重置为推动段
							} else {
								// 下跌趋势：先确认L，再确认H
								result = append(result, MarkPoint{
									Type: PointL, Index: retroL.Index,
									Timestamp: retroL.Timestamp, Price: retroL.Price,
									Level: level, Multiplier: multiplier,
								})
								result = append(result, MarkPoint{
									Type: PointH, Index: retroH.Index,
									Timestamp: retroH.Timestamp, Price: retroH.Price,
									Level: level, Multiplier: multiplier,
								})
								lastConfirmedType = PointH
								lastConfirmedIndex = retroH.Index
								lastConfirmedPrice = retroH.Price
								impulseAllowed = true // 回溯对确认后重置为推动段
							}
							// 当前点作为新候选
							candidate = candidateInfo{point: pt, valid: true}
							continue
						} else {
							// 未找到中间回调段，但趋势距离已满足阈值
							// 直接确认当前更极端的反向点为新的转折点
							// 这处理了长期趋势被分成多个小段的情况
							result = append(result, MarkPoint{
								Type:       pt.Type,
								Index:      pt.Index,
								Timestamp:  pt.Timestamp,
								Price:      pt.Price,
								Level:      level,
								Multiplier: multiplier,
							})
							lastConfirmedType = pt.Type
							lastConfirmedIndex = pt.Index
							lastConfirmedPrice = pt.Price
							impulseAllowed = false // 直接极值确认为推动段，下次为回调段
							candidate = candidateInfo{valid: false}
							continue
						}
					}
					// 未找到合格回调，正常替换候选
					candidate = candidateInfo{point: pt, valid: true}
				}
				// 不是更极端的点：保留现有候选
			}
		} else {
			// 同方向点（与 lastConfirmedType 相同）：代表从 candidate 的回调/反弹
			if !candidate.valid {
				continue
			}

			// 计算从候选极值到当前点的K线根数
			barCount := pt.Index - candidate.point.Index

			shouldConfirm := false

			// 阈值检测：回调段（barCount）需要满足最小要求
			// 规则：仅推动段可短于阈值，回调段必须严格满足阈值
			// barCount 检测的是 candidate → pt 的距离（回调/反弹距离）
			threshold2x := minRetraceBars * 2
			if barCount >= threshold2x {
				// 规则2: >=2x 明确跳级（进入更高级别范围）
				shouldConfirm = true
			} else if barCount >= minRetraceBars {
				// 规则1: >=1x 达到当前级别最低要求，确认
				shouldConfirm = true
			}
			// < 1x: 同级别波动，不确认（即使是突破也需要等待更显著的回调）

			// 推动段规则：推动/回调交替确认
			// 推动段（从已确认转折点出发的第一段）可以在短barCount下确认，
			// 只要回调不破已确认点的价格。回调段仍需正常barCount阈值。
			if !shouldConfirm && impulseAllowed {
				if (lastConfirmedType == PointL && pt.Price >= lastConfirmedPrice) ||
					(lastConfirmedType == PointH && pt.Price <= lastConfirmedPrice) {
					shouldConfirm = true
				}
			}

			if shouldConfirm {
				// 确认候选点为当前级别的转折点
				result = append(result, MarkPoint{
					Type:       candidate.point.Type,
					Index:      candidate.point.Index,
					Timestamp:  candidate.point.Timestamp,
					Price:      candidate.point.Price,
					Level:      level,
					Multiplier: multiplier,
				})
				lastConfirmedType = candidate.point.Type
				lastConfirmedIndex = candidate.point.Index
				lastConfirmedPrice = candidate.point.Price
				impulseAllowed = !impulseAllowed // 推动/回调交替切换
				// 当前同向点（相对于旧 lastConfirmedType）现在是反方向
				// （因为 lastConfirmedType 刚改变了），作为下一段的候选起点
				candidate = candidateInfo{point: pt, valid: true}
			}
			// 不确认时：保留现有 candidate，继续扫描
		}
	}

	// 追加尾部候选点（确保结果序列末尾有反方向点）
	if candidate.valid && len(result) > 0 && candidate.point.Type != result[len(result)-1].Type {
		result = append(result, MarkPoint{
			Type:       candidate.point.Type,
			Index:      candidate.point.Index,
			Timestamp:  candidate.point.Timestamp,
			Price:      candidate.point.Price,
			Level:      level,
			Multiplier: multiplier,
		})
	}

	return result
}

// insertMissingThresholdPoints 检测并插入缺失的超阈值转折点
// 在已确认的同级别点之间，如果存在超过该级别阈值的回调/上涨段，需要将其端点插入为同级别点
// 这确保了每个级别的高低点之间不会存在超过该级别阈值的未标记段
func (c *MoshiChanlunCalculator) insertMissingThresholdPoints(
	points []MarkPoint,
	prevPoints []MarkPoint,
	minRetraceBars int,
	multiplier int,
	klines []models.KLine,
) []MarkPoint {
	if len(points) < 2 || len(prevPoints) < 2 {
		return points
	}

	level := getLevelNameByMultiplier(multiplier)

	// 迭代处理直到没有新点被插入（最多10次避免无限循环）
	for iteration := 0; iteration < 10; iteration++ {
		result := make([]MarkPoint, 0, len(points)*2)
		changed := false

		for i := 0; i < len(points); i++ {
			currentPoint := points[i]
			result = append(result, currentPoint)

			// 检查当前点和下一个点之间是否有超阈值段
			if i < len(points)-1 {
				nextPoint := points[i+1]

				// 筛选出当前点和下一个点之间的子级别点
				var betweenPoints []MarkPoint
				for _, pt := range prevPoints {
					if pt.Index > currentPoint.Index && pt.Index < nextPoint.Index {
						betweenPoints = append(betweenPoints, pt)
					}
				}

				// 扫描 betweenPoints 中是否存在超阈值的段
				insertPoints := c.findOverThresholdSegments(
					betweenPoints, currentPoint, nextPoint,
					minRetraceBars, level, multiplier, klines)

				if len(insertPoints) > 0 {
					result = append(result, insertPoints...)
					changed = true
				}
			}
		}

		if !changed {
			return result
		}

		// 按索引排序结果
		sort.Slice(result, func(i, j int) bool {
			return result[i].Index < result[j].Index
		})

		// 去重：移除相同索引的重复点
		deduped := make([]MarkPoint, 0, len(result))
		for i, pt := range result {
			if i == 0 || pt.Index != result[i-1].Index {
				deduped = append(deduped, pt)
			}
		}

		points = deduped
	}

	return points
}

// findOverThresholdSegments 在子级别点序列中查找超过阈值的段
// 返回需要插入的转折点列表
func (c *MoshiChanlunCalculator) findOverThresholdSegments(
	betweenPoints []MarkPoint,
	startPoint, endPoint MarkPoint,
	minRetraceBars int,
	level string,
	multiplier int,
	klines []models.KLine,
) []MarkPoint {
	if len(betweenPoints) < 2 {
		return nil
	}

	var insertPoints []MarkPoint

	// 扫描相邻的H/L对，找超阈值段
	for i := 0; i < len(betweenPoints)-1; i++ {
		p1 := betweenPoints[i]
		p2 := betweenPoints[i+1]

		// 只检查H->L或L->H的相邻对
		if p1.Type == p2.Type {
			continue
		}

		segmentBars := p2.Index - p1.Index
		if segmentBars >= minRetraceBars {
			// 找到超阈值段，需要将两个端点都插入
			// 但只插入与父级别点类型不同的点（避免重复）

			// p1 点
			if p1.Index != startPoint.Index && p1.Index != endPoint.Index {
				insertPoints = append(insertPoints, MarkPoint{
					Type:       p1.Type,
					Index:      p1.Index,
					Timestamp:  p1.Timestamp,
					Price:      p1.Price,
					Level:      level,
					Multiplier: multiplier,
				})
			}

			// p2 点
			if p2.Index != startPoint.Index && p2.Index != endPoint.Index {
				insertPoints = append(insertPoints, MarkPoint{
					Type:       p2.Type,
					Index:      p2.Index,
					Timestamp:  p2.Timestamp,
					Price:      p2.Price,
					Level:      level,
					Multiplier: multiplier,
				})
			}
		}
	}

	return insertPoints
}

// calculateRetraceTime 计算回调时间（从高点开始下跌的时间）
func (c *MoshiChanlunCalculator) calculateRetraceTime(klines []models.KLine, highIndex, currentIndex int) float64 {
	if currentIndex <= highIndex || highIndex < 0 {
		return 0
	}

	// 检查是否在回调中（当前价格低于高点）
	if klines[currentIndex].Close >= klines[highIndex].High {
		return 0
	}

	// 计算时间差（分钟）
	return c.getTimeDiffMinutes(klines[highIndex].Timestamp, klines[currentIndex].Timestamp)
}

// calculateReboundTime 计算反弹时间（从低点开始上涨的时间）
func (c *MoshiChanlunCalculator) calculateReboundTime(klines []models.KLine, lowIndex, currentIndex int) float64 {
	if currentIndex <= lowIndex || lowIndex < 0 {
		return 0
	}

	// 检查是否在反弹中（当前价格高于低点）
	if klines[currentIndex].Close <= klines[lowIndex].Low {
		return 0
	}

	// 计算时间差（分钟）
	return c.getTimeDiffMinutes(klines[lowIndex].Timestamp, klines[currentIndex].Timestamp)
}

// getTimeDiffMinutes 计算两个时间戳之间的分钟差
func (c *MoshiChanlunCalculator) getTimeDiffMinutes(startTime, endTime string) float64 {
	layouts := []string{
		"2006-1-2 15:4:5",
		"2006-01-02 15:04:05",
		"2006-1-2 15:4",
		"2006-01-02 15:04",
		"2006-1-2",
		"2006-01-02",
		"2006/1/2 15:4:5",
		"2006/01/02 15:04:05",
		"2006/1/2",
		"2006/01/02",
	}

	// 使用本地时区解析，避免混合时区导致时差错误
	loc := time.Local

	var start, end time.Time
	var err error

	for _, layout := range layouts {
		start, err = time.ParseInLocation(layout, startTime, loc)
		if err == nil {
			break
		}
	}
	if err != nil {
		return 0
	}

	for _, layout := range layouts {
		end, err = time.ParseInLocation(layout, endTime, loc)
		if err == nil {
			break
		}
	}
	if err != nil {
		return 0
	}

	return end.Sub(start).Minutes()
}

// generateBiMarkers 生成笔标记（连接L-H点）
func (c *MoshiChanlunCalculator) generateBiMarkers(points []MarkPoint, klines []models.KLine) []models.BiMarker {
	if len(points) < 2 {
		return nil
	}

	// 按multiplier分组
	groupedPoints := make(map[int][]MarkPoint)
	for _, p := range points {
		groupedPoints[p.Multiplier] = append(groupedPoints[p.Multiplier], p)
	}

	biMarkers := make([]models.BiMarker, 0)

	for mult, pts := range groupedPoints {
		for i := 0; i < len(pts)-1; i++ {
			startPoint := pts[i]
			endPoint := pts[i+1]

			// 跳过双向突破产生的零长度段（同一根K线上的H和L）
			if startPoint.Index == endPoint.Index {
				continue
			}

			direction := "up"
			if startPoint.Type == PointH {
				direction = "down"
			}

			// 统计线段内的上涨K线和下跌K线数量
			upCount, downCount := c.countKLines(klines, startPoint.Index, endPoint.Index)

			// 计算实际回调/反弹时间
			actualRetraceTime := c.getTimeDiffMinutes(startPoint.Timestamp, endPoint.Timestamp)

			bi := models.BiMarker{
				StartIndex:        startPoint.Index,
				EndIndex:          endPoint.Index,
				StartTimestamp:    startPoint.Timestamp,
				EndTimestamp:      endPoint.Timestamp,
				StartPrice:        startPoint.Price,
				EndPrice:          endPoint.Price,
				Direction:         direction,
				Length:            endPoint.Index - startPoint.Index,
				UpCount:           upCount,
				DownCount:         downCount,
				Multiplier:        mult,
				ActualRetraceTime: actualRetraceTime,
			}

			biMarkers = append(biMarkers, bi)
		}
	}

	return biMarkers
}

// countKLines 统计线段内的上涨K线和下跌K线数量
func (c *MoshiChanlunCalculator) countKLines(klines []models.KLine, startIdx, endIdx int) (upCount, downCount int) {
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= len(klines) {
		endIdx = len(klines) - 1
	}
	if startIdx >= endIdx {
		return 0, 0
	}

	for i := startIdx; i <= endIdx; i++ {
		k := klines[i]
		if k.Close > k.Open {
			upCount++
		} else if k.Close < k.Open {
			downCount++
		}
		// 十字星(Close == Open)不计入
	}

	return upCount, downCount
}

// 辅助函数
func getIntParam(params map[string]interface{}, key string, defaultVal int) int {
	if v, ok := params[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		case int64:
			return int(val)
		}
	}
	return defaultVal
}

func getBoolParam(params map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := params[key]; ok {
		switch val := v.(type) {
		case bool:
			return val
		case string:
			return val == "true" || val == "1"
		}
	}
	return defaultVal
}

// identifySameLevelTrends 识别同级别走势（适用于x1/x2/x4/x8各级别）
// 走势类型优先级：
// 1. 收敛型中枢盘整（6点以上，高点先下移再上移，低点上移）
// 2. 扩张型中枢盘整（6点以上，高点上移，低点先上移再下移）
// 3. 趋势型走势（4点以上，高点上移且低点上移/高点下移且低点下移）
// multiplier: 当前级别倍数 (1/2/4/8)
// klines: 原始K线数据，用于区间极值验证
func (c *MoshiChanlunCalculator) identifySameLevelTrends(points []MarkPoint, multiplier int) []SameLevelTrend {
	// 允许只有2个点（1个H和1个L）的情况，作为最简单的走势
	if len(points) < 2 {
		return nil
	}

	trends := make([]SameLevelTrend, 0)
	i := 0

	for i < len(points) {
		// 优先尝试从当前位置构建上涨走势（起点必须是L）
		if points[i].Type == PointL {
			// 1. 首先尝试识别收敛型中枢（优先级最高，需要6点）
			trend, endIdx := c.tryBuildUpConvergentPivotGeneric(points, i, multiplier)
			if trend != nil {
				trends = append(trends, *trend)
				i = endIdx
				continue
			}

			// 2. 尝试识别扩张型中枢
			trend, endIdx = c.tryBuildUpDivergentPivotGeneric(points, i, multiplier)
			if trend != nil {
				trends = append(trends, *trend)
				i = endIdx
				continue
			}

			// 3. 尝试识别普通上涨趋势（需要4点，但允许2点的简单走势）
			trend, endIdx = c.tryBuildUpTrendGeneric(points, i, multiplier)
			if trend != nil && len(trend.Points) >= 2 {
				trends = append(trends, *trend)
				i = endIdx
				continue
			}
		}

		// 尝试从当前位置构建下跌走势（起点必须是H）
		if points[i].Type == PointH {
			trend, endIdx := c.tryBuildDownTrendGeneric(points, i, multiplier)
			if trend != nil && len(trend.Points) >= 2 {
				trends = append(trends, *trend)
				i = endIdx
				continue
			}
		}

		i++
	}

	return trends
}

// identifySameLevelTrendsWithKlines 识别同级别走势（带K线极值验证）
// 对于上涨走势，验证：
// - H点必须是前一个L点到下一个L点之间区间的最高价
// - L点必须是前一个H点到下一个H点之间区间的最低价
// 如果验证失败，重新标注真正的极值点
func (c *MoshiChanlunCalculator) identifySameLevelTrendsWithKlines(points []MarkPoint, multiplier int, klines []models.KLine) []SameLevelTrend {
	if len(points) < 2 || len(klines) == 0 {
		return nil
	}

	// 首先进行区间极值验证和修正
	validatedPoints := c.validateAndCorrectExtremePoints(points, klines)

	// 使用修正后的点序列进行走势识别
	return c.identifySameLevelTrends(validatedPoints, multiplier)
}

// validateAndCorrectExtremePoints 验证并修正点序列中的极值点
// 对于上涨走势中的 L1 → H1 → L2 → H2 序列：
// - H1必须是L1和L2之间所有K线中的最高点
// - L2必须是H1和H2之间所有K线中的最低点
// 如果验证失败，重新标注真正的极值点
func (c *MoshiChanlunCalculator) validateAndCorrectExtremePoints(points []MarkPoint, klines []models.KLine) []MarkPoint {
	if len(points) < 2 || len(klines) == 0 {
		return points
	}

	result := make([]MarkPoint, len(points))
	copy(result, points)

	// 多次迭代直到所有点都满足极值条件（最多迭代10次避免无限循环）
	for iteration := 0; iteration < 10; iteration++ {
		changed := false

		// 遍历点序列，验证每个H点和L点
		for i := 0; i < len(result); i++ {
			pt := result[i]

			if pt.Type == PointH {
				// 验证H点：必须是前一个L点到下一个L点之间区间的最高价
				var prevL, nextL *MarkPoint

				// 找前一个L点
				for j := i - 1; j >= 0; j-- {
					if result[j].Type == PointL {
						prevL = &result[j]
						break
					}
				}
				// 找下一个L点
				for j := i + 1; j < len(result); j++ {
					if result[j].Type == PointL {
						nextL = &result[j]
						break
					}
				}

				// 如果存在前一个L和下一个L，验证H点
				if prevL != nil && nextL != nil {
					correctedH := c.findHighestInRange(klines, prevL.Index, nextL.Index)
					if correctedH != nil && correctedH.Index != pt.Index {
						// 检查是否与相邻L点索引冲突
						if correctedH.Index == prevL.Index || correctedH.Index == nextL.Index {
							// 尝试找次优高点（排除冲突索引）
							correctedH = c.findHighestInRangeExcluding(klines, prevL.Index, nextL.Index,
								[]int{prevL.Index, nextL.Index})
							if correctedH == nil || correctedH.Index == pt.Index {
								continue // 无合适替代，保持原点
							}
						}
						// H点不是区间最高，需要修正
						result[i] = MarkPoint{
							Type:       PointH,
							Index:      correctedH.Index,
							Timestamp:  klines[correctedH.Index].Timestamp,
							Price:      correctedH.Price,
							Level:      pt.Level,
							Multiplier: pt.Multiplier,
						}
						changed = true
					}
				} else if prevL != nil {
					// 只有前一个L（末尾的H点），验证从prevL到当前H的区间
					correctedH := c.findHighestInRange(klines, prevL.Index, pt.Index)
					if correctedH != nil && correctedH.Index != pt.Index {
						result[i] = MarkPoint{
							Type:       PointH,
							Index:      correctedH.Index,
							Timestamp:  klines[correctedH.Index].Timestamp,
							Price:      correctedH.Price,
							Level:      pt.Level,
							Multiplier: pt.Multiplier,
						}
						changed = true
					}
				}
			} else { // PointL
				// 验证L点：必须是前一个H点到下一个H点之间区间的最低价
				var prevH, nextH *MarkPoint

				// 找前一个H点
				for j := i - 1; j >= 0; j-- {
					if result[j].Type == PointH {
						prevH = &result[j]
						break
					}
				}
				// 找下一个H点
				for j := i + 1; j < len(result); j++ {
					if result[j].Type == PointH {
						nextH = &result[j]
						break
					}
				}

				// 如果存在前一个H和下一个H，验证L点
				if prevH != nil && nextH != nil {
					correctedL := c.findLowestInRange(klines, prevH.Index, nextH.Index)
					if correctedL != nil && correctedL.Index != pt.Index {
						// 检查是否与相邻H点索引冲突
						if correctedL.Index == prevH.Index || correctedL.Index == nextH.Index {
							// 尝试找次优低点（排除冲突索引）
							correctedL = c.findLowestInRangeExcluding(klines, prevH.Index, nextH.Index,
								[]int{prevH.Index, nextH.Index})
							if correctedL == nil || correctedL.Index == pt.Index {
								continue // 无合适替代，保持原点
							}
						}
						// L点不是区间最低，需要修正
						result[i] = MarkPoint{
							Type:       PointL,
							Index:      correctedL.Index,
							Timestamp:  klines[correctedL.Index].Timestamp,
							Price:      correctedL.Price,
							Level:      pt.Level,
							Multiplier: pt.Multiplier,
						}
						changed = true
					}
				} else if prevH != nil {
					// 只有前一个H（末尾的L点），验证从prevH到当前L的区间
					correctedL := c.findLowestInRange(klines, prevH.Index, pt.Index)
					if correctedL != nil && correctedL.Index != pt.Index {
						result[i] = MarkPoint{
							Type:       PointL,
							Index:      correctedL.Index,
							Timestamp:  klines[correctedL.Index].Timestamp,
							Price:      correctedL.Price,
							Level:      pt.Level,
							Multiplier: pt.Multiplier,
						}
						changed = true
					}
				}
			}
		}

		if !changed {
			break // 所有点都已满足条件
		}
	}

	// 最后清理：移除任何仍然存在的同索引H/L对
	result = c.removeSameIndexPairs(result)

	return result
}

// enforceMinBarDistance 确保回调段间距不小于最小K线根数阈值（灵活距离规则）
// 仅检查回调段（retracement）：上涨趋势中的H→L、下跌趋势中的L→H
// 推动段（impulse）：上涨趋势中的L→H、下跌趋势中的H→L，不受最小间距限制
// 趋势方向判断：比较相邻同类型点的价格
//   - 连续L抬高（higher lows）→ 上涨趋势 → H→L为回调段
//   - 连续H降低（lower highs）→ 下跌趋势 → L→H为回调段
// 迭代执行直到所有回调段间距都满足要求（最多10次）
func (c *MoshiChanlunCalculator) enforceMinBarDistance(points []MarkPoint, minBars int) []MarkPoint {
	if len(points) < 3 {
		return points
	}

	for pass := 0; pass < 10; pass++ {
		changed := false
		result := []MarkPoint{points[0]}

		for i := 1; i < len(points); i++ {
			cur := points[i]
			last := result[len(result)-1]

			// 同类型合并：保留更极端的
			if cur.Type == last.Type {
				changed = true
				if (cur.Type == PointH && cur.Price > last.Price) ||
					(cur.Type == PointL && cur.Price < last.Price) {
					result[len(result)-1] = cur
				}
				continue
			}

			// 不同类型，通过比较相邻同类型点的价格判断是否为回调段
			// prev(result[-2])和cur是同类型（H/L严格交替，跳一个就同类型）
			// 上涨趋势（连续L抬高 or 连续H抬高）→ H→L是回调，L→H是推动
			// 下跌趋势（连续L降低 or 连续H降低）→ L→H是回调，H→L是推动
			isRetracement := false
			if len(result) >= 2 {
				prev := result[len(result)-2]
				if prev.Type == PointL && cur.Type == PointL {
					// 两个连续L点：后L >= 前L → 上涨趋势(抬高低点) → H→L为回调
					isRetracement = cur.Price >= prev.Price
				} else if prev.Type == PointH && cur.Type == PointH {
					// 两个连续H点：后H <= 前H → 下跌趋势(降低高点) → L→H为回调
					isRetracement = cur.Price <= prev.Price
				}
			}

			// 仅对回调段检查最小间距，推动段直接保留
			if cur.Index-last.Index >= minBars || !isRetracement {
				result = append(result, cur)
			} else {
				// 回调段间距不足：直接移除（回调段不享受突破保护）
				// 用户要求：仅推动段可短于阈值，回调段必须严格满足阈值
				changed = true
				if len(result) >= 2 {
					// 移除last（短线段的起点），让prev直接连接后续点
					prev := result[len(result)-2]
					result = result[:len(result)-1]

					// 此时prev和cur可能是同类型（因为移除了中间的last）
					if prev.Type == cur.Type {
						// 同类型合并：保留更极端的
						if (cur.Type == PointH && cur.Price > prev.Price) ||
							(cur.Type == PointL && cur.Price < prev.Price) {
							result[len(result)-1] = cur
						}
					} else {
						// 不同类型：添加cur
						result = append(result, cur)
					}
				}
				// len(result) < 2: 只有起始点，跳过cur
			}
		}

		if !changed {
			break
		}
		points = result
	}

	return points
}

// appendTrailingPoints 追加尾部追踪点
// 当管道处理后最后一个标注点距离K线末尾较远时，从K线数据中扫描并追加极值点，
// 确保图表右侧始终有标注。这些点是"未确认的候选点"，用于可视化连续性。
func (c *MoshiChanlunCalculator) appendTrailingPoints(points []MarkPoint, klines []models.KLine, minBars int, multiplier int) []MarkPoint {
	if len(points) < 2 || len(klines) < 2 {
		return points
	}

	lastPt := points[len(points)-1]
	totalKLines := len(klines)
	trailingGap := totalKLines - 1 - lastPt.Index

	// 如果最后标注点离数据末尾不远，无需追加
	if trailingGap < minBars {
		return points
	}

	level := getLevelNameByMultiplier(multiplier)
	startIdx := lastPt.Index + 1
	if startIdx >= totalKLines {
		return points
	}

	// 根据最后一个点的类型，决定扫描方向
	// 如果最后是L，需要找后续的H（上涨段的终点），然后可能还要找L（回调底部）
	// 如果最后是H，需要找后续的L（下跌段的终点），然后可能还要找H（反弹顶部）
	if lastPt.Type == PointL {
		// 扫描最高点作为H候选
		highIdx := startIdx
		for i := startIdx + 1; i < totalKLines; i++ {
			if klines[i].High > klines[highIdx].High {
				highIdx = i
			}
		}
		if highIdx > lastPt.Index {
			points = append(points, MarkPoint{
				Type:       PointH,
				Index:      highIdx,
				Timestamp:  klines[highIdx].Timestamp,
				Price:      klines[highIdx].High,
				Level:      level,
				Multiplier: multiplier,
			})
			// 如果最高点之后还有K线，扫描最低点作为L候选
			if highIdx < totalKLines-1 {
				lowIdx := highIdx + 1
				for i := highIdx + 2; i < totalKLines; i++ {
					if klines[i].Low < klines[lowIdx].Low {
						lowIdx = i
					}
				}
				if lowIdx > highIdx {
					points = append(points, MarkPoint{
						Type:       PointL,
						Index:      lowIdx,
						Timestamp:  klines[lowIdx].Timestamp,
						Price:      klines[lowIdx].Low,
						Level:      level,
						Multiplier: multiplier,
					})
				}
			}
		}
	} else {
		// 扫描最低点作为L候选
		lowIdx := startIdx
		for i := startIdx + 1; i < totalKLines; i++ {
			if klines[i].Low < klines[lowIdx].Low {
				lowIdx = i
			}
		}
		if lowIdx > lastPt.Index {
			points = append(points, MarkPoint{
				Type:       PointL,
				Index:      lowIdx,
				Timestamp:  klines[lowIdx].Timestamp,
				Price:      klines[lowIdx].Low,
				Level:      level,
				Multiplier: multiplier,
			})
			// 如果最低点之后还有K线，扫描最高点作为H候选
			if lowIdx < totalKLines-1 {
				highIdx := lowIdx + 1
				for i := lowIdx + 2; i < totalKLines; i++ {
					if klines[i].High > klines[highIdx].High {
						highIdx = i
					}
				}
				if highIdx > lowIdx {
					points = append(points, MarkPoint{
						Type:       PointH,
						Index:      highIdx,
						Timestamp:  klines[highIdx].Timestamp,
						Price:      klines[highIdx].High,
						Level:      level,
						Multiplier: multiplier,
					})
				}
			}
		}
	}

	return points
}

// scanRetroactiveRetrace 在prevPoints中扫描回溯回调对
// 在指定K线索引范围(startIdx, endIdx)内（不含两端），寻找满足minRetraceBars的回调段
// lastType表示趋势方向：PointL=上涨趋势（找H→L回调），PointH=下跌趋势（找L→H反弹）
// 返回第一个满足条件的(H点, L点)对，未找到返回(nil, nil)
func (c *MoshiChanlunCalculator) scanRetroactiveRetrace(
	prevPoints []MarkPoint,
	startIdx, endIdx int,
	lastType PointType,
	minRetraceBars int,
) (*MarkPoint, *MarkPoint) {
	// 筛选范围内的点（不含两端）
	var filtered []MarkPoint
	for _, pt := range prevPoints {
		if pt.Index > startIdx && pt.Index < endIdx {
			filtered = append(filtered, pt)
		}
	}

	if len(filtered) < 2 {
		return nil, nil
	}

	// 扫描相邻的H/L对
	for i := 0; i < len(filtered)-1; i++ {
		p1 := filtered[i]
		p2 := filtered[i+1]

		if lastType == PointL {
			// 上涨趋势：寻找 H→L 回调段
			if p1.Type == PointH && p2.Type == PointL && p2.Index-p1.Index >= minRetraceBars {
				return &p1, &p2
			}
		} else {
			// 下跌趋势：寻找 L→H 反弹段
			if p1.Type == PointL && p2.Type == PointH && p2.Index-p1.Index >= minRetraceBars {
				return &p2, &p1 // 返回顺序仍为(H点, L点)
			}
		}
	}

	return nil, nil
}

// scanKLineRetrace 在原始K线数据中扫描回调（sub-level未找到时的后备方案）
// 在上涨趋势中：先找区间内最高点，再找其后minRetraceBars以外的最低点
// 在下跌趋势中：先找区间内最低点，再找其后minRetraceBars以外的最高点
// 找到K线级别的回调后，将结果对齐到最近的prevPoints以维护层级包含关系
// 返回(H点, L点)对，未找到返回(nil, nil)
func (c *MoshiChanlunCalculator) scanKLineRetrace(
	klines []models.KLine,
	startIdx, endIdx int,
	lastType PointType,
	minRetraceBars int,
	prevPoints []MarkPoint,
) (*MarkPoint, *MarkPoint) {
	// 确保范围有效（排除两端）
	scanStart := startIdx + 1
	scanEnd := endIdx - 1
	if scanStart > scanEnd || scanStart < 0 || scanEnd >= len(klines) {
		return nil, nil
	}

	// snapToNearestPrevPoint 将K线索引对齐到最近的prevPoints中同类型的点
	snapToNearestPrevPoint := func(targetIdx int, targetType PointType) *MarkPoint {
		var best *MarkPoint
		bestDist := int(^uint(0) >> 1) // MaxInt
		for j := range prevPoints {
			pt := &prevPoints[j]
			if pt.Type != targetType || pt.Index <= startIdx || pt.Index >= endIdx {
				continue
			}
			dist := targetIdx - pt.Index
			if dist < 0 {
				dist = -dist
			}
			if dist < bestDist {
				bestDist = dist
				best = pt
			}
		}
		return best
	}

	if lastType == PointL {
		// 上涨趋势：找最高点→然后找其后的最低点
		peakIdx := scanStart
		for i := scanStart + 1; i <= scanEnd; i++ {
			if klines[i].High > klines[peakIdx].High {
				peakIdx = i
			}
		}

		// 在peakIdx + minRetraceBars之后找最低点
		troughStart := peakIdx + minRetraceBars
		if troughStart > scanEnd {
			return nil, nil
		}
		troughIdx := troughStart
		for i := troughStart + 1; i <= scanEnd; i++ {
			if klines[i].Low < klines[troughIdx].Low {
				troughIdx = i
			}
		}

		if troughIdx-peakIdx >= minRetraceBars {
			// 对齐到prevPoints维护层级关系
			snappedH := snapToNearestPrevPoint(peakIdx, PointH)
			snappedL := snapToNearestPrevPoint(troughIdx, PointL)
			if snappedH != nil && snappedL != nil && snappedL.Index-snappedH.Index >= minRetraceBars {
				return snappedH, snappedL
			}
			// 对齐后不满足条件，放弃
			return nil, nil
		}
	} else {
		// 下跌趋势：找最低点→然后找其后的最高点
		troughIdx := scanStart
		for i := scanStart + 1; i <= scanEnd; i++ {
			if klines[i].Low < klines[troughIdx].Low {
				troughIdx = i
			}
		}

		// 在troughIdx + minRetraceBars之后找最高点
		peakStart := troughIdx + minRetraceBars
		if peakStart > scanEnd {
			return nil, nil
		}
		peakIdx := peakStart
		for i := peakStart + 1; i <= scanEnd; i++ {
			if klines[i].High > klines[peakIdx].High {
				peakIdx = i
			}
		}

		if peakIdx-troughIdx >= minRetraceBars {
			// 对齐到prevPoints维护层级关系
			snappedH := snapToNearestPrevPoint(peakIdx, PointH)
			snappedL := snapToNearestPrevPoint(troughIdx, PointL)
			if snappedH != nil && snappedL != nil && snappedH.Index-snappedL.Index >= minRetraceBars {
				return snappedH, snappedL
			}
			return nil, nil
		}
	}

	return nil, nil
}

// findHighestInRange 在K线区间内找到最高价点
// 返回区间内最高价的K线索引和价格
func (c *MoshiChanlunCalculator) findHighestInRange(klines []models.KLine, startIdx, endIdx int) *struct {
	Index int
	Price float64
} {
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= len(klines) {
		endIdx = len(klines) - 1
	}
	if startIdx > endIdx {
		return nil
	}

	highestIdx := startIdx
	highestPrice := klines[startIdx].High

	for i := startIdx + 1; i <= endIdx; i++ {
		if klines[i].High > highestPrice {
			highestPrice = klines[i].High
			highestIdx = i
		}
	}

	return &struct {
		Index int
		Price float64
	}{Index: highestIdx, Price: highestPrice}
}

// findLowestInRange 在K线区间内找到最低价点
// 返回区间内最低价的K线索引和价格
func (c *MoshiChanlunCalculator) findLowestInRange(klines []models.KLine, startIdx, endIdx int) *struct {
	Index int
	Price float64
} {
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= len(klines) {
		endIdx = len(klines) - 1
	}
	if startIdx > endIdx {
		return nil
	}

	lowestIdx := startIdx
	lowestPrice := klines[startIdx].Low

	for i := startIdx + 1; i <= endIdx; i++ {
		if klines[i].Low < lowestPrice {
			lowestPrice = klines[i].Low
			lowestIdx = i
		}
	}

	return &struct {
		Index int
		Price float64
	}{Index: lowestIdx, Price: lowestPrice}
}

// findHighestInRangeExcluding 在K线区间内找到最高价点，排除指定索引
// 用于防止validateAndCorrectExtremePoints将H修正到与相邻L相同的索引
func (c *MoshiChanlunCalculator) findHighestInRangeExcluding(klines []models.KLine, startIdx, endIdx int, excludeIndices []int) *struct {
	Index int
	Price float64
} {
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= len(klines) {
		endIdx = len(klines) - 1
	}
	if startIdx > endIdx {
		return nil
	}

	// 构建排除索引的map以便快速查找
	excludeMap := make(map[int]bool)
	for _, idx := range excludeIndices {
		excludeMap[idx] = true
	}

	highestIdx := -1
	highestPrice := 0.0

	for i := startIdx; i <= endIdx; i++ {
		if excludeMap[i] {
			continue
		}
		if highestIdx == -1 || klines[i].High > highestPrice {
			highestPrice = klines[i].High
			highestIdx = i
		}
	}

	if highestIdx == -1 {
		return nil
	}

	return &struct {
		Index int
		Price float64
	}{Index: highestIdx, Price: highestPrice}
}

// findLowestInRangeExcluding 在K线区间内找到最低价点，排除指定索引
// 用于防止validateAndCorrectExtremePoints将L修正到与相邻H相同的索引
func (c *MoshiChanlunCalculator) findLowestInRangeExcluding(klines []models.KLine, startIdx, endIdx int, excludeIndices []int) *struct {
	Index int
	Price float64
} {
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= len(klines) {
		endIdx = len(klines) - 1
	}
	if startIdx > endIdx {
		return nil
	}

	// 构建排除索引的map以便快速查找
	excludeMap := make(map[int]bool)
	for _, idx := range excludeIndices {
		excludeMap[idx] = true
	}

	lowestIdx := -1
	lowestPrice := 0.0

	for i := startIdx; i <= endIdx; i++ {
		if excludeMap[i] {
			continue
		}
		if lowestIdx == -1 || klines[i].Low < lowestPrice {
			lowestPrice = klines[i].Low
			lowestIdx = i
		}
	}

	if lowestIdx == -1 {
		return nil
	}

	return &struct {
		Index int
		Price float64
	}{Index: lowestIdx, Price: lowestPrice}
}

// removeSameIndexPairs 移除同索引的H/L点对
// 当validateAndCorrectExtremePoints无法避免同索引时，通过后处理移除
func (c *MoshiChanlunCalculator) removeSameIndexPairs(points []MarkPoint) []MarkPoint {
	if len(points) < 2 {
		return points
	}

	result := make([]MarkPoint, 0, len(points))
	i := 0
	for i < len(points) {
		if i+1 < len(points) && points[i].Index == points[i+1].Index {
			// 同索引的H/L对：跳过两个点，保留更极端的那个（根据前后上下文决定）
			// 简单策略：如果前面有点，保留与前面类型不同的那个；否则跳过两个
			if len(result) > 0 {
				lastType := result[len(result)-1].Type
				if points[i].Type != lastType {
					result = append(result, points[i])
				} else if points[i+1].Type != lastType {
					result = append(result, points[i+1])
				}
				// 如果两个都与lastType相同，都跳过
			}
			i += 2
		} else {
			result = append(result, points[i])
			i++
		}
	}

	return result
}

// identifyX1SameLevelTrends 识别x1同级别走势（保留向后兼容）
func (c *MoshiChanlunCalculator) identifyX1SameLevelTrends(x1Points []MarkPoint) []SameLevelTrend {
	return c.identifySameLevelTrends(x1Points, 1)
}

// tryBuildUpTrend 尝试从指定位置构建上涨走势（保留向后兼容）
func (c *MoshiChanlunCalculator) tryBuildUpTrend(x1Points []MarkPoint, startIdx int) (*SameLevelTrend, int) {
	return c.tryBuildUpTrendGeneric(x1Points, startIdx, 1)
}

// tryBuildUpTrendGeneric 尝试从指定位置构建上涨走势（通用版本）
// 上涨走势规则：
// - 第一个低点是起始点，也是整个走势的最低点
// - 第二个低点必须高于第一个低点
// - 第一个高点必须高于第二个低点（保证有上涨空间）
// - 第二个高点必须高于第一个高点
// - 允许只有L-H两个点的简单走势
// 返回：走势对象和结束点在points中的索引
func (c *MoshiChanlunCalculator) tryBuildUpTrendGeneric(points []MarkPoint, startIdx int, multiplier int) (*SameLevelTrend, int) {
	if startIdx >= len(points) || points[startIdx].Type != PointL {
		return nil, startIdx
	}

	trendPoints := make([]MarkPoint, 0)
	trendPoints = append(trendPoints, points[startIdx])

	var lastL *MarkPoint
	lastL = &points[startIdx]
	lowestL := lastL // 追踪最低L点（必须是第一个L）
	var highestH *MarkPoint

	endIdx := startIdx

	for j := startIdx + 1; j < len(points); j++ {
		pt := points[j]

		if pt.Type == PointH {
			// 检查高点规则
			if highestH != nil && pt.Price <= highestH.Price {
				// 高点没有上移，走势结束
				break
			}
			// 高点必须高于最近的低点（保证上涨结构）
			if lastL != nil && pt.Price <= lastL.Price {
				break
			}

			trendPoints = append(trendPoints, pt)
			if highestH == nil || pt.Price > highestH.Price {
				highestH = &points[j]
			}
			endIdx = j
		} else { // PointL
			// 检查低点规则
			if pt.Price <= lowestL.Price {
				// 低点没有上移（跌破起始低点），走势结束
				break
			}
			if lastL != nil && pt.Price <= lastL.Price {
				// 低点没有上移，走势结束
				break
			}

			trendPoints = append(trendPoints, pt)
			lastL = &points[j]
			endIdx = j
		}
	}

	// 检查是否满足最小2点要求（允许简单的L-H走势）
	if len(trendPoints) < 2 || highestH == nil {
		return nil, startIdx
	}

	// 验证点序列中至少有1个L和1个H
	lCount, hCount := 0, 0
	for _, p := range trendPoints {
		if p.Type == PointL {
			lCount++
		} else {
			hCount++
		}
	}
	if lCount < 1 || hCount < 1 {
		return nil, startIdx
	}

	// 构建走势对象
	trend := &SameLevelTrend{
		Type:           "up",
		Pattern:        "trend", // 趋势型走势
		Multiplier:     multiplier,
		StartIndex:     trendPoints[0].Index,
		EndIndex:       trendPoints[len(trendPoints)-1].Index,
		StartTimestamp: trendPoints[0].Timestamp,
		EndTimestamp:   trendPoints[len(trendPoints)-1].Timestamp,
		LowPoint:       *lowestL,   // 上涨走势的最低点=第一个L
		HighPoint:      *highestH,  // 上涨走势的最高点=最后一个H
		Points:         trendPoints,
	}

	return trend, endIdx
}

// tryBuildDownTrend 尝试从指定位置构建下跌走势（保留向后兼容）
func (c *MoshiChanlunCalculator) tryBuildDownTrend(x1Points []MarkPoint, startIdx int) (*SameLevelTrend, int) {
	return c.tryBuildDownTrendGeneric(x1Points, startIdx, 1)
}

// tryBuildDownTrendGeneric 尝试从指定位置构建下跌走势（通用版本）
// 下跌走势规则：
// - 第一个高点是起始点，也是整个走势的最高点
// - 第二个高点必须低于第一个高点
// - 第一个低点必须低于第二个高点（保证有下跌空间）
// - 第二个低点必须低于第一个低点
// - 允许只有H-L两个点的简单走势
// 返回：走势对象和结束点在points中的索引
func (c *MoshiChanlunCalculator) tryBuildDownTrendGeneric(points []MarkPoint, startIdx int, multiplier int) (*SameLevelTrend, int) {
	if startIdx >= len(points) || points[startIdx].Type != PointH {
		return nil, startIdx
	}

	trendPoints := make([]MarkPoint, 0)
	trendPoints = append(trendPoints, points[startIdx])

	var lastH *MarkPoint
	lastH = &points[startIdx]
	highestH := lastH // 追踪最高H点（必须是第一个H）
	var lowestL *MarkPoint

	endIdx := startIdx

	for j := startIdx + 1; j < len(points); j++ {
		pt := points[j]

		if pt.Type == PointL {
			// 检查低点规则
			if lowestL != nil && pt.Price >= lowestL.Price {
				// 低点没有下移，走势结束
				break
			}
			// 低点必须低于最近的高点（保证下跌结构）
			if lastH != nil && pt.Price >= lastH.Price {
				break
			}

			trendPoints = append(trendPoints, pt)
			if lowestL == nil || pt.Price < lowestL.Price {
				lowestL = &points[j]
			}
			endIdx = j
		} else { // PointH
			// 检查高点规则
			if pt.Price >= highestH.Price {
				// 高点没有下移（突破起始高点），走势结束
				break
			}
			if lastH != nil && pt.Price >= lastH.Price {
				// 高点没有下移，走势结束
				break
			}

			trendPoints = append(trendPoints, pt)
			lastH = &points[j]
			endIdx = j
		}
	}

	// 检查是否满足最小2点要求（允许简单的H-L走势）
	if len(trendPoints) < 2 || lowestL == nil {
		return nil, startIdx
	}

	// 验证点序列中至少有1个L和1个H
	lCount, hCount := 0, 0
	for _, p := range trendPoints {
		if p.Type == PointL {
			lCount++
		} else {
			hCount++
		}
	}
	if lCount < 1 || hCount < 1 {
		return nil, startIdx
	}

	// 构建走势对象
	trend := &SameLevelTrend{
		Type:           "down",
		Pattern:        "trend", // 趋势型走势
		Multiplier:     multiplier,
		StartIndex:     trendPoints[0].Index,
		EndIndex:       trendPoints[len(trendPoints)-1].Index,
		StartTimestamp: trendPoints[0].Timestamp,
		EndTimestamp:   trendPoints[len(trendPoints)-1].Timestamp,
		HighPoint:      *highestH, // 下跌走势的最高点=第一个H
		LowPoint:       *lowestL,  // 下跌走势的最低点=最后一个L
		Points:         trendPoints,
	}

	return trend, endIdx
}

// tryBuildUpConvergentPivot 尝试从指定位置构建上涨收敛型中枢盘整走势（保留向后兼容）
func (c *MoshiChanlunCalculator) tryBuildUpConvergentPivot(x1Points []MarkPoint, startIdx int) (*SameLevelTrend, int) {
	return c.tryBuildUpConvergentPivotGeneric(x1Points, startIdx, 1)
}

// tryBuildUpConvergentPivotGeneric 尝试从指定位置构建上涨收敛型中枢盘整走势（通用版本）
// 收敛型规则（至少6个点：L1-H1-L2-H2-L3-H3）：
// - L1 < L2 < L3 (低点上移)
// - H1 > H2 < H3 (高点先下移再上移，H3突破H1)
// 级别升级规则：当H2到L3的时间间隔 > (H1到L2的时间间隔 × 2) 时触发升级
// 返回：走势对象和结束点在points中的索引
func (c *MoshiChanlunCalculator) tryBuildUpConvergentPivotGeneric(points []MarkPoint, startIdx int, multiplier int) (*SameLevelTrend, int) {
	if startIdx >= len(points) || points[startIdx].Type != PointL {
		return nil, startIdx
	}

	// 收集从startIdx开始的连续点
	trendPoints := make([]MarkPoint, 0)
	lows := make([]MarkPoint, 0)  // 收集所有L点
	highs := make([]MarkPoint, 0) // 收集所有H点

	trendPoints = append(trendPoints, points[startIdx])
	lows = append(lows, points[startIdx])

	endIdx := startIdx

	// 收集点直到不满足基本结构或达到足够点数
	for j := startIdx + 1; j < len(points); j++ {
		pt := points[j]

		// 基本约束：低点不能跌破第一个低点
		if pt.Type == PointL && pt.Price <= lows[0].Price {
			break
		}

		trendPoints = append(trendPoints, pt)
		if pt.Type == PointL {
			lows = append(lows, pt)
		} else {
			highs = append(highs, pt)
		}
		endIdx = j

		// 检查是否满足收敛型条件
		if len(lows) >= 3 && len(highs) >= 3 {
			// 验证收敛型规则：L1 < L2 < L3 且 H1 > H2 < H3 且 H3 > H1
			l1, l2, l3 := lows[0], lows[1], lows[2]
			h1, h2, h3 := highs[0], highs[1], highs[2]

			lowsAscending := l1.Price < l2.Price && l2.Price < l3.Price
			highsConvergent := h1.Price > h2.Price && h2.Price < h3.Price && h3.Price > h1.Price

			if lowsAscending && highsConvergent {
				// 找到收敛型中枢
				trend := &SameLevelTrend{
					Type:           "up",
					Pattern:        "convergent", // 收敛型中枢
					Multiplier:     multiplier,
					StartIndex:     trendPoints[0].Index,
					EndIndex:       trendPoints[len(trendPoints)-1].Index,
					StartTimestamp: trendPoints[0].Timestamp,
					EndTimestamp:   trendPoints[len(trendPoints)-1].Timestamp,
					LowPoint:       l1,                       // 最低点=第一个L
					HighPoint:      highs[len(highs)-1],      // 最高点=最后一个H
					Points:         trendPoints,
					Upgraded:       false,
				}

				// 检查是否满足级别升级条件
				// 第一次回调K线根数：H1 到 L2
				firstBarCount := l2.Index - h1.Index
				// 第二次回调K线根数：H2 到 L3
				secondBarCount := l3.Index - h2.Index

				// 当第二次回调K线根数 > 第一次回调K线根数 × 2 时，触发级别升级
				if secondBarCount > firstBarCount*2 {
					trend.Upgraded = true
					// 父级别点映射：
					// 父级别L1 ← 子级别L1（第一个低点）
					// 父级别H1 ← 子级别H2（第二个高点）
					// 父级别L2 ← 子级别L3（第三个低点）
					// 父级别H2 ← 子级别H3（第三个高点）
					trend.ParentPoints = []MarkPoint{
						l1, // 父级别L1
						h2, // 父级别H1
						l3, // 父级别L2
						h3, // 父级别H2
					}
				}

				return trend, endIdx
			}
		}
	}

	return nil, startIdx
}

// tryBuildUpDivergentPivot 尝试从指定位置构建上涨扩张型中枢盘整走势（保留向后兼容）
func (c *MoshiChanlunCalculator) tryBuildUpDivergentPivot(x1Points []MarkPoint, startIdx int) (*SameLevelTrend, int) {
	return c.tryBuildUpDivergentPivotGeneric(x1Points, startIdx, 1)
}

// tryBuildUpDivergentPivotGeneric 尝试从指定位置构建上涨扩张型中枢盘整走势（通用版本）
// 扩张型规则（至少6个点：L1-H1-L2-H2-L3-H3）：
// - L1 < L2 > L3 且 L3 > L1 (低点先上移再下移，但L3仍高于L1)
// - H1 < H2 < H3 (高点持续上移)
// 返回：走势对象和结束点在points中的索引
func (c *MoshiChanlunCalculator) tryBuildUpDivergentPivotGeneric(points []MarkPoint, startIdx int, multiplier int) (*SameLevelTrend, int) {
	if startIdx >= len(points) || points[startIdx].Type != PointL {
		return nil, startIdx
	}

	// 收集从startIdx开始的连续点
	trendPoints := make([]MarkPoint, 0)
	lows := make([]MarkPoint, 0)  // 收集所有L点
	highs := make([]MarkPoint, 0) // 收集所有H点

	trendPoints = append(trendPoints, points[startIdx])
	lows = append(lows, points[startIdx])

	endIdx := startIdx

	// 收集点直到不满足基本结构或达到足够点数
	for j := startIdx + 1; j < len(points); j++ {
		pt := points[j]

		// 基本约束：低点不能跌破第一个低点
		if pt.Type == PointL && pt.Price <= lows[0].Price {
			break
		}

		trendPoints = append(trendPoints, pt)
		if pt.Type == PointL {
			lows = append(lows, pt)
		} else {
			highs = append(highs, pt)
		}
		endIdx = j

		// 检查是否满足扩张型条件
		if len(lows) >= 3 && len(highs) >= 3 {
			// 验证扩张型规则：L1 < L2 > L3 且 L3 > L1 且 H1 < H2 < H3
			l1, l2, l3 := lows[0], lows[1], lows[2]
			h1, h2, h3 := highs[0], highs[1], highs[2]

			lowsDivergent := l1.Price < l2.Price && l2.Price > l3.Price && l3.Price > l1.Price
			highsAscending := h1.Price < h2.Price && h2.Price < h3.Price

			if lowsDivergent && highsAscending {
				// 找到扩张型中枢
				trend := &SameLevelTrend{
					Type:           "up",
					Pattern:        "divergent", // 扩张型中枢
					Multiplier:     multiplier,
					StartIndex:     trendPoints[0].Index,
					EndIndex:       trendPoints[len(trendPoints)-1].Index,
					StartTimestamp: trendPoints[0].Timestamp,
					EndTimestamp:   trendPoints[len(trendPoints)-1].Timestamp,
					LowPoint:       l1,                       // 最低点=第一个L
					HighPoint:      highs[len(highs)-1],      // 最高点=最后一个H
					Points:         trendPoints,
				}
				return trend, endIdx
			}
		}
	}

	return nil, startIdx
}

// deriveX2FromTrends 从x1同级别走势的端点推导x2级别点
// @Deprecated: 此函数已废弃，不再在主流程中使用。
// Calculate函数现在统一使用deriveNextLevel进行级别推导。
// 保留此函数仅用于：1) 向后兼容 2) 测试验证函数本身的逻辑
func (c *MoshiChanlunCalculator) deriveX2FromTrends(x1Points []MarkPoint, sameLevelTrends []SameLevelTrend, minRetraceBars int) []MarkPoint {
	return c.deriveHigherLevelFromTrends(x1Points, sameLevelTrends, minRetraceBars, 2)
}

// deriveHigherLevelFromTrends 从同级别走势的端点推导更高级别点（通用版本）
// @Deprecated: 此函数已废弃，不再在主流程中使用。
// Calculate函数现在统一使用deriveNextLevel进行级别推导，走势识别仅用于前端可视化。
// 保留此函数仅用于：1) 向后兼容 2) 测试验证函数本身的逻辑
//
// 历史用途（父子级别关系规则）：
// - 上涨走势：父级别低点=走势第一个L，父级别高点=走势最后H
// - 下跌走势：父级别高点=走势第一个H，父级别低点=走势最后L
// - 级别升级走势：使用ParentPoints中的映射点
// targetMultiplier: 目标级别倍数 (2/4/8)
func (c *MoshiChanlunCalculator) deriveHigherLevelFromTrends(childPoints []MarkPoint, sameLevelTrends []SameLevelTrend, minRetraceBars int, targetMultiplier int) []MarkPoint {
	if len(sameLevelTrends) == 0 {
		return nil
	}

	level := getLevelNameByMultiplier(targetMultiplier)
	result := make([]MarkPoint, 0)

	// 用于去重的辅助函数
	isDuplicate := func(points []MarkPoint, newPoint MarkPoint) bool {
		if len(points) == 0 {
			return false
		}
		last := points[len(points)-1]
		return last.Index == newPoint.Index && last.Type == newPoint.Type
	}

	// 转换为目标级别点
	convertToTargetLevel := func(mp MarkPoint) MarkPoint {
		return MarkPoint{
			Type:       mp.Type,
			Index:      mp.Index,
			Timestamp:  mp.Timestamp,
			Price:      mp.Price,
			Level:      level,
			Multiplier: targetMultiplier,
		}
	}

	for _, trend := range sameLevelTrends {
		// 检查是否为升级的收敛型中枢走势
		if trend.Upgraded && len(trend.ParentPoints) >= 4 {
			// 使用升级后的父级别点序列
			// ParentPoints顺序：L1(父), H1(父), L2(父), H2(父)
			for _, pt := range trend.ParentPoints {
				targetPt := convertToTargetLevel(pt)
				if !isDuplicate(result, targetPt) {
					// 确保L-H交替
					if len(result) == 0 || result[len(result)-1].Type != targetPt.Type {
						result = append(result, targetPt)
					}
				}
			}
		} else if trend.Type == "up" {
			// 普通上涨走势：先添加低点L，再添加高点H
			lowPt := convertToTargetLevel(trend.LowPoint)
			if !isDuplicate(result, lowPt) {
				// 确保L-H交替：如果前一个也是L，跳过
				if len(result) == 0 || result[len(result)-1].Type != lowPt.Type {
					result = append(result, lowPt)
				}
			}

			highPt := convertToTargetLevel(trend.HighPoint)
			if !isDuplicate(result, highPt) {
				// 确保L-H交替
				if len(result) == 0 || result[len(result)-1].Type != highPt.Type {
					result = append(result, highPt)
				}
			}
		} else { // down
			// 下跌走势：先添加高点H，再添加低点L
			highPt := convertToTargetLevel(trend.HighPoint)
			if !isDuplicate(result, highPt) {
				if len(result) == 0 || result[len(result)-1].Type != highPt.Type {
					result = append(result, highPt)
				}
			}

			lowPt := convertToTargetLevel(trend.LowPoint)
			if !isDuplicate(result, lowPt) {
				if len(result) == 0 || result[len(result)-1].Type != lowPt.Type {
					result = append(result, lowPt)
				}
			}
		}
	}

	return result
}
