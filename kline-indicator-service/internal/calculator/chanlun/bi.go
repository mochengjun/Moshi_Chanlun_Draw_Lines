package chanlun

import (
	"fmt"
	"time"

	"kline-indicator-service/internal/calculator"
	"kline-indicator-service/internal/models"
)

// BiIndicator 缠论笔指标
type BiIndicator struct{}

// NewBiIndicator 创建笔指标
func NewBiIndicator() calculator.Indicator {
	return &BiIndicator{}
}

// Metadata 获取元信息
func (b *BiIndicator) Metadata() models.IndicatorMetadata {
	return models.IndicatorMetadata{
		Type:        models.IndicatorTypeChanlunBi,
		Name:        "缠论笔",
		Category:    models.CategoryChanlun,
		Description: "识别K线的笔结构，笔由顶分型和底分型之间的K线组成",
		ParamsDef: []models.ParameterDef{
			{
				Name:         "min_klines",
				Type:         "int",
				Required:     false,
				DefaultValue: 5,
				Min:          4,
				Max:          10,
				Description:  "笔最少包含的K线数",
			},
			{
				Name:         "strict",
				Type:         "bool",
				Required:     false,
				DefaultValue: true,
				Description:  "是否严格模式",
			},
		},
	}
}

// Validate 验证参数
func (b *BiIndicator) Validate(params map[string]interface{}) error {
	minKLines := getIntParam(params, "min_klines", 5)
	if minKLines < 4 || minKLines > 10 {
		return fmt.Errorf("min_klines必须在4到10之间")
	}
	return nil
}

// Calculate 计算笔
func (b *BiIndicator) Calculate(klines []models.KLine, params map[string]interface{}) (*models.IndicatorResult, error) {
	startTime := time.Now()
	
	minKLines := getIntParam(params, "min_klines", 5)
	strict := getBoolParam(params, "strict", true)
	
	if err := b.Validate(params); err != nil {
		return nil, err
	}
	
	if len(klines) < minKLines {
		return nil, fmt.Errorf("K线数据不足，需要至少%d根K线", minKLines)
	}
	
	// 先识别分型
	fractalIndicator := &FractalIndicator{}
	fractalResult, err := fractalIndicator.Calculate(klines, map[string]interface{}{"strict": strict})
	if err != nil {
		return nil, fmt.Errorf("识别分型失败: %w", err)
	}
	
	// 根据分型识别笔
	biMarkers := detectBi(fractalResult.FractalMarkers, klines, minKLines)
	
	return &models.IndicatorResult{
		Type:            models.IndicatorTypeChanlunBi,
		Name:            "缠论笔",
		FractalMarkers:  fractalResult.FractalMarkers,
		BiMarkers:       biMarkers,
		ComputationTime: time.Since(startTime).Milliseconds(),
	}, nil
}

// detectBi 识别笔
func detectBi(fractals []models.FractalMarker, klines []models.KLine, minKLines int) []models.BiMarker {
	if len(fractals) < 2 {
		return nil
	}
	
	biMarkers := make([]models.BiMarker, 0)
	
	// 筛选有效的分型序列（顶底交替）
	validFractals := filterAlternatingFractals(fractals)
	
	if len(validFractals) < 2 {
		return nil
	}
	
	// 根据相邻分型构建笔
	for i := 0; i < len(validFractals)-1; i++ {
		start := validFractals[i]
		end := validFractals[i+1]
		
		// 计算笔包含的K线数
		length := end.Index - start.Index + 1
		
		// 检查笔的有效性
		if length < minKLines {
			continue
		}
		
		// 确定笔方向
		var direction string
		if start.Type == "bottom" && end.Type == "top" {
			direction = "up"
		} else if start.Type == "top" && end.Type == "bottom" {
			direction = "down"
		} else {
			continue // 无效的分型组合
		}
		
		biMarkers = append(biMarkers, models.BiMarker{
			StartIndex:     start.Index,
			EndIndex:       end.Index,
			StartTimestamp: start.Timestamp,
			EndTimestamp:   end.Timestamp,
			StartPrice:     start.Price,
			EndPrice:       end.Price,
			Direction:      direction,
			Length:         length,
		})
	}
	
	return biMarkers
}

// filterAlternatingFractals 筛选顶底交替的分型
func filterAlternatingFractals(fractals []models.FractalMarker) []models.FractalMarker {
	if len(fractals) == 0 {
		return nil
	}
	
	result := make([]models.FractalMarker, 0, len(fractals))
	result = append(result, fractals[0])
	
	for i := 1; i < len(fractals); i++ {
		last := result[len(result)-1]
		current := fractals[i]
		
		// 如果类型相同，保留更极端的那个
		if last.Type == current.Type {
			if last.Type == "top" {
				// 顶分型，保留更高的
				if current.Price > last.Price {
					result[len(result)-1] = current
				}
			} else {
				// 底分型，保留更低的
				if current.Price < last.Price {
					result[len(result)-1] = current
				}
			}
		} else {
			// 类型不同，添加新分型
			result = append(result, current)
		}
	}
	
	return result
}

// getIntParam 获取整数参数
func getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if params == nil {
		return defaultValue
	}
	v, ok := params[key]
	if !ok {
		return defaultValue
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return defaultValue
	}
}

// 初始化时注册指标
func init() {
	calculator.Register(models.IndicatorTypeChanlunBi, NewBiIndicator)
}
