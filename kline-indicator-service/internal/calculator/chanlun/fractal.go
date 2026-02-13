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
	
	var processedKLines []ProcessedKLine
	if strict {
		// 处理包含关系
		processedKLines = processInclusion(klines)
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
			}
		}
	}
	
	// 识别分型
	fractals := detectFractals(processedKLines, klines)
	
	return &models.IndicatorResult{
		Type:            models.IndicatorTypeChanlunFractal,
		Name:            "缠论分型",
		FractalMarkers:  fractals,
		ComputationTime: time.Since(startTime).Milliseconds(),
	}, nil
}

// processInclusion 处理K线包含关系
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

// determineDirection 确定处理方向
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

// detectFractals 识别分型
func detectFractals(processed []ProcessedKLine, original []models.KLine) []models.FractalMarker {
	if len(processed) < 3 {
		return nil
	}
	
	fractals := make([]models.FractalMarker, 0)
	
	for i := 1; i < len(processed)-1; i++ {
		left := processed[i-1]
		mid := processed[i]
		right := processed[i+1]
		
		// 顶分型：中间K线高点最高，且低点也最高
		if mid.High > left.High && mid.High > right.High &&
			mid.Low > left.Low && mid.Low > right.Low {
			
			// 获取原始K线索引来确定时间戳
			origIdx := mid.OriginalIndices[0]
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
		
		// 底分型：中间K线低点最低，且高点也最低
		if mid.Low < left.Low && mid.Low < right.Low &&
			mid.High < left.High && mid.High < right.High {
			
			origIdx := mid.OriginalIndices[0]
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
