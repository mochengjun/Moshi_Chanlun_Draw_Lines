package chanlun

import (
	"fmt"
	"time"

	"kline-indicator-service/internal/calculator"
	"kline-indicator-service/internal/models"
)

// ProcessedKLine 处理后的K线（无包含关系）
type ProcessedKLine struct {
	Index           int
	High            float64
	Low             float64
	Timestamp       string
	OriginalIndices []int
	HighOrigIdx     int
	LowOrigIdx      int
}

// FractalResult 包含分型结果的返回值
type FractalResult struct {
	ProcessedKLines []ProcessedKLine
	Fractals        []models.FractalMarker
}

// FractalIndicator 缠论分型指标
type FractalIndicator struct{}

// NewFractalIndicator 创建分型指标
func NewFractalIndicator() calculator.Indicator {
	return &FractalIndicator{}
}

// Metadata 获取元信息
func (f *FractalIndicator) Metadata() models.IndicatorMetadata {
	return models.IndicatorMetadata{
		Type:        models.IndicatorTypeChanlunFractal,
		Name:        "缠论分型",
		Category:    models.CategoryChanlun,
		Description: "识别K线的顶分型和底分型，基于缠论分型定义",
		ParamsDef: []models.ParameterDef{
			{
				Name:         "strict",
				Type:         "bool",
				Required:     false,
				DefaultValue: true,
				Description:  "是否严格模式（处理包含关系）",
			},
		},
	}
}

// Validate 验证参数
func (f *FractalIndicator) Validate(params map[string]interface{}) error {
	return nil
}

// Calculate 计算分型
func (f *FractalIndicator) Calculate(klines []models.KLine, params map[string]interface{}) (*models.IndicatorResult, error) {
	startTime := time.Now()

	strict := getBoolParam(params, "strict", true)

	if len(klines) < 3 {
		return nil, fmt.Errorf("K线数据不足，需要至少3根K线")
	}

	var fractals []models.FractalMarker
	var processedKLines []ProcessedKLine

	if strict {
		// 同步处理包含关系和分型识别
		result := processInclusionSync(klines)
		processedKLines = result.ProcessedKLines
		fractals = result.Fractals
	} else {
		// 不处理包含关系，直接使用原始K线
		processedKLines = make([]ProcessedKLine, len(klines))
		for i, k := range klines {
			processedKLines[i] = ProcessedKLine{
				Index:           i,
				High:            k.High,
				Low:             k.Low,
				Timestamp:       k.Timestamp,
				OriginalIndices: []int{i},
				HighOrigIdx:     i,
				LowOrigIdx:      i,
			}
		}
		// 识别分型
		fractals = detectFractalsDirect(processedKLines, klines)
	}

	return &models.IndicatorResult{
		Type:            models.IndicatorTypeChanlunFractal,
		Name:            "缠论分型",
		FractalMarkers:  fractals,
		ComputationTime: time.Since(startTime).Milliseconds(),
	}, nil
}

// processInclusionSync 同步处理K线包含关系与分型识别
// 在处理包含关系的过程中同时识别分型，确定趋势方向
// - 形成顶分型时确定方向向下，后续包含关系按下降趋势处理
// - 形成底分型时确定方向向上，后续包含关系按上升趋势处理
// - 顶底分型之间可以共用同一根K线
func processInclusionSync(klines []models.KLine) FractalResult {
	result := FractalResult{
		ProcessedKLines: make([]ProcessedKLine, 0, len(klines)),
		Fractals:        make([]models.FractalMarker, 0),
	}

	if len(klines) < 2 {
		result.ProcessedKLines = make([]ProcessedKLine, len(klines))
		for i, k := range klines {
			result.ProcessedKLines[i] = ProcessedKLine{
				Index:           i,
				High:            k.High,
				Low:             k.Low,
				Timestamp:       k.Timestamp,
				OriginalIndices: []int{i},
				HighOrigIdx:     i,
				LowOrigIdx:      i,
			}
		}
		return result
	}

	// 当前趋势方向：初始为空，确定分型后变为UP或DOWN
	currentTrend := "UNDEFINED"

	// 第一根K线直接加入
	result.ProcessedKLines = append(result.ProcessedKLines, ProcessedKLine{
		Index:           0,
		High:            klines[0].High,
		Low:             klines[0].Low,
		Timestamp:       klines[0].Timestamp,
		OriginalIndices: []int{0},
		HighOrigIdx:     0,
		LowOrigIdx:      0,
	})

	for i := 1; i < len(klines); i++ {
		current := klines[i]
		prev := result.ProcessedKLines[len(result.ProcessedKLines)-1]

		// 检查包含关系
		if hasInclusion(prev.High, prev.Low, current.High, current.Low) {
			// 确定处理方向
			direction := determineDirectionSync(result.ProcessedKLines, currentTrend)

			// 合并K线
			var newHigh, newLow float64
			var newHighOrigIdx, newLowOrigIdx int

			if direction == "UP" {
				// 向上处理：取高高、高低
				if current.High > prev.High {
					newHigh = current.High
					newHighOrigIdx = i
				} else {
					newHigh = prev.High
					newHighOrigIdx = prev.HighOrigIdx
				}
				if current.Low > prev.Low {
					newLow = current.Low
					newLowOrigIdx = i
				} else {
					newLow = prev.Low
					newLowOrigIdx = prev.LowOrigIdx
				}
			} else {
				// 向下处理：取低高、低低
				if current.High < prev.High {
					newHigh = current.High
					newHighOrigIdx = i
				} else {
					newHigh = prev.High
					newHighOrigIdx = prev.HighOrigIdx
				}
				if current.Low < prev.Low {
					newLow = current.Low
					newLowOrigIdx = i
				} else {
					newLow = prev.Low
					newLowOrigIdx = prev.LowOrigIdx
				}
			}

			// 更新最后一个处理后的K线
			lastIdx := len(result.ProcessedKLines) - 1
			result.ProcessedKLines[lastIdx].High = newHigh
			result.ProcessedKLines[lastIdx].Low = newLow
			result.ProcessedKLines[lastIdx].HighOrigIdx = newHighOrigIdx
			result.ProcessedKLines[lastIdx].LowOrigIdx = newLowOrigIdx
			result.ProcessedKLines[lastIdx].OriginalIndices = append(result.ProcessedKLines[lastIdx].OriginalIndices, i)

			// 处理完包含关系后，检查是否形成分型并更新趋势
			if len(result.ProcessedKLines) >= 3 {
				checkAndUpdateTrendSync(result.ProcessedKLines, &currentTrend, result.Fractals, klines)
			}
		} else {
			// 无包含关系，直接添加
			result.ProcessedKLines = append(result.ProcessedKLines, ProcessedKLine{
				Index:           i,
				High:            current.High,
				Low:             current.Low,
				Timestamp:       current.Timestamp,
				OriginalIndices: []int{i},
				HighOrigIdx:     i,
				LowOrigIdx:      i,
			})

			// 检查是否形成分型并更新趋势
			if len(result.ProcessedKLines) >= 3 {
				checkAndUpdateTrendSync(result.ProcessedKLines, &currentTrend, result.Fractals, klines)
			}
		}
	}

	return result
}

// determineDirectionSync 确定处理方向
func determineDirectionSync(processed []ProcessedKLine, currentTrend string) string {
	// 如果已有明确趋势，使用当前趋势方向
	if currentTrend == "UP" || currentTrend == "DOWN" {
		return currentTrend
	}

	// 初始状态：根据前后K线变化确定方向
	if len(processed) < 2 {
		return "UP"
	}

	prev := processed[len(processed)-2]
	curr := processed[len(processed)-1]

	// 无包含关系时
	if !hasInclusion(prev.High, prev.Low, curr.High, curr.Low) {
		if curr.High > prev.High && curr.Low > prev.Low {
			return "UP"
		}
		if curr.High < prev.High && curr.Low < prev.Low {
			return "DOWN"
		}
	}

	// 有包含关系时：比较最高点
	if curr.High > prev.High {
		return "UP"
	}
	return "DOWN"
}

// checkAndUpdateTrendSync 检查是否形成分型并更新趋势
func checkAndUpdateTrendSync(processed []ProcessedKLine, currentTrend *string, fractals *[]models.FractalMarker, klines []models.KLine) {
	if len(processed) < 3 {
		return
	}

	n := len(processed)
	left := processed[n-3]
	mid := processed[n-2]
	right := processed[n-1]

	// 识别顶分型：中间K线高点最高
	hasTopFractal := mid.High > left.High && mid.High > right.High
	if hasTopFractal {
		origIdx := mid.HighOrigIdx
		if origIdx < len(klines) {
			*fractals = append(*fractals, models.FractalMarker{
				Timestamp: klines[origIdx].Timestamp,
				Type:      "top",
				Price:     mid.High,
				Index:     origIdx,
				Zone:      [2]float64{mid.Low, mid.High},
			})
		}
		// 顶分型形成，趋势转为向下
		*currentTrend = "DOWN"
	}

	// 识别底分型：中间K线低点最低（与顶分型可共用同一根K线）
	hasBottomFractal := mid.Low < left.Low && mid.Low < right.Low
	if hasBottomFractal {
		origIdx := mid.LowOrigIdx
		if origIdx < len(klines) {
			*fractals = append(*fractals, models.FractalMarker{
				Timestamp: klines[origIdx].Timestamp,
				Type:      "bottom",
				Price:     mid.Low,
				Index:     origIdx,
				Zone:      [2]float64{mid.Low, mid.High},
			})
		}
		// 底分型形成，趋势转为向上
		*currentTrend = "UP"
	}
}

// processInclusion 处理K线包含关系（旧版兼容）
func processInclusion(klines []models.KLine) []ProcessedKLine {
	if len(klines) < 2 {
		result := make([]ProcessedKLine, len(klines))
		for i, k := range klines {
			result[i] = ProcessedKLine{
				Index:           i,
				High:            k.High,
				Low:             k.Low,
				Timestamp:       k.Timestamp,
				OriginalIndices: []int{i},
				HighOrigIdx:     i,
				LowOrigIdx:      i,
			}
		}
		return result
	}

	result := make([]ProcessedKLine, 0, len(klines))

	for i := 0; i < len(klines); i++ {
		if i == 0 {
			result = append(result, ProcessedKLine{
				Index:           i,
				High:            klines[i].High,
				Low:             klines[i].Low,
				Timestamp:       klines[i].Timestamp,
				OriginalIndices: []int{i},
				HighOrigIdx:     i,
				LowOrigIdx:      i,
			})
			continue
		}

		current := klines[i]
		prev := result[len(result)-1]

		// 检查包含关系
		if hasInclusion(prev.High, prev.Low, current.High, current.Low) {
			// 确定处理方向
			direction := determineDirection(result)

			// 合并K线
			var newHigh, newLow float64
			if direction == "UP" {
				// 向上处理：取高高、高低
				newHigh = maxFloat(prev.High, current.High)
				newLow = maxFloat(prev.Low, current.Low)
			} else {
				// 向下处理：取低高、低低
				newHigh = minFloat(prev.High, current.High)
				newLow = minFloat(prev.Low, current.Low)
			}

			// 更新最后一个处理后的K线
			result[len(result)-1].High = newHigh
			result[len(result)-1].Low = newLow
			result[len(result)-1].OriginalIndices = append(result[len(result)-1].OriginalIndices, i)
		} else {
			// 无包含关系，直接添加
			result = append(result, ProcessedKLine{
				Index:           i,
				High:            current.High,
				Low:             current.Low,
				Timestamp:       current.Timestamp,
				OriginalIndices: []int{i},
				HighOrigIdx:     i,
				LowOrigIdx:      i,
			})
		}
	}

	return result
}

// hasInclusion 检查是否有包含关系
func hasInclusion(h1, l1, h2, l2 float64) bool {
	// K1包含K2 或 K2包含K1
	return (h1 >= h2 && l1 <= l2) || (h2 >= h1 && l2 <= l1)
}

// determineDirection 确定处理方向（旧版兼容）
func determineDirection(processed []ProcessedKLine) string {
	if len(processed) < 2 {
		return "UP"
	}

	// 根据前两根K线确定方向
	prev := processed[len(processed)-2]
	curr := processed[len(processed)-1]

	if curr.High > prev.High {
		return "UP"
	}
	return "DOWN"
}

// detectFractalsDirect 直接从处理后的K线识别分型（不处理包含关系时使用）
func detectFractalsDirect(processed []ProcessedKLine, original []models.KLine) []models.FractalMarker {
	if len(processed) < 3 {
		return nil
	}

	fractals := make([]models.FractalMarker, 0)

	for i := 1; i < len(processed)-1; i++ {
		left := processed[i-1]
		mid := processed[i]
		right := processed[i+1]

		// 顶分型：中间K线高点最高
		if mid.High > left.High && mid.High > right.High {
			origIdx := mid.HighOrigIdx
			if origIdx < len(original) {
				fractals = append(fractals, models.FractalMarker{
					Timestamp: original[origIdx].Timestamp,
					Type:      "top",
					Price:     mid.High,
					Index:     origIdx,
					Zone:      [2]float64{mid.Low, mid.High},
				})
			}
		}

		// 底分型：中间K线低点最低
		if mid.Low < left.Low && mid.Low < right.Low {
			origIdx := mid.LowOrigIdx
			if origIdx < len(original) {
				fractals = append(fractals, models.FractalMarker{
					Timestamp: original[origIdx].Timestamp,
					Type:      "bottom",
					Price:     mid.Low,
					Index:     origIdx,
					Zone:      [2]float64{mid.Low, mid.High},
				})
			}
		}
	}

	return fractals
}

// detectFractals 识别分型（旧版兼容）
func detectFractals(processed []ProcessedKLine, original []models.KLine) []models.FractalMarker {
	return detectFractalsDirect(processed, original)
}

// getBoolParam 获取布尔参数
func getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if params == nil {
		return defaultValue
	}
	v, ok := params[key]
	if !ok {
		return defaultValue
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return defaultValue
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// 初始化时注册指标
func init() {
	calculator.Register(models.IndicatorTypeChanlunFractal, NewFractalIndicator)
}
