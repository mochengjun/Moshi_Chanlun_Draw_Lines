package basic

import (
	"fmt"
	"time"

	"kline-indicator-service/internal/calculator"
	"kline-indicator-service/internal/models"
)

// MAIndicator 移动平均线指标
type MAIndicator struct{}

// NewMAIndicator 创建MA指标
func NewMAIndicator() calculator.Indicator {
	return &MAIndicator{}
}

// Metadata 获取元信息
func (m *MAIndicator) Metadata() models.IndicatorMetadata {
	return models.IndicatorMetadata{
		Type:        models.IndicatorTypeMA,
		Name:        "移动平均线",
		Category:    models.CategoryBasic,
		Description: "简单移动平均线(SMA)，计算指定周期内收盘价的算术平均值",
		ParamsDef: []models.ParameterDef{
			{
				Name:         "period",
				Type:         "int",
				Required:     true,
				DefaultValue: 5,
				Min:          1,
				Max:          250,
				Description:  "计算周期",
			},
		},
	}
}

// Validate 验证参数
func (m *MAIndicator) Validate(params map[string]interface{}) error {
	period := getIntParam(params, "period", 5)
	if period < 1 || period > 250 {
		return fmt.Errorf("period必须在1到250之间")
	}
	return nil
}

// Calculate 计算MA
func (m *MAIndicator) Calculate(klines []models.KLine, params map[string]interface{}) (*models.IndicatorResult, error) {
	startTime := time.Now()
	
	period := getIntParam(params, "period", 5)
	if err := m.Validate(params); err != nil {
		return nil, err
	}
	
	if len(klines) < period {
		return nil, fmt.Errorf("K线数据不足，需要至少%d根K线", period)
	}
	
	values := make([]models.IndicatorValue, len(klines))
	
	// 计算移动平均
	for i := 0; i < len(klines); i++ {
		if i < period-1 {
			values[i] = models.IndicatorValue{
				Timestamp: klines[i].Timestamp,
				Value:     0, // 数据不足时为0
			}
			continue
		}
		
		sum := 0.0
		for j := 0; j < period; j++ {
			sum += klines[i-j].Close
		}
		
		values[i] = models.IndicatorValue{
			Timestamp: klines[i].Timestamp,
			Value:     sum / float64(period),
		}
	}
	
	return &models.IndicatorResult{
		Type: models.IndicatorTypeMA,
		Name: fmt.Sprintf("MA(%d)", period),
		Series: []models.IndicatorSeries{
			{
				Name:   fmt.Sprintf("MA%d", period),
				Values: values,
			},
		},
		ComputationTime: time.Since(startTime).Milliseconds(),
	}, nil
}

// EMAIndicator 指数移动平均线指标
type EMAIndicator struct{}

// NewEMAIndicator 创建EMA指标
func NewEMAIndicator() calculator.Indicator {
	return &EMAIndicator{}
}

// Metadata 获取元信息
func (e *EMAIndicator) Metadata() models.IndicatorMetadata {
	return models.IndicatorMetadata{
		Type:        models.IndicatorTypeEMA,
		Name:        "指数移动平均线",
		Category:    models.CategoryBasic,
		Description: "指数移动平均线(EMA)，对近期价格赋予更高权重",
		ParamsDef: []models.ParameterDef{
			{
				Name:         "period",
				Type:         "int",
				Required:     true,
				DefaultValue: 12,
				Min:          1,
				Max:          250,
				Description:  "计算周期",
			},
		},
	}
}

// Validate 验证参数
func (e *EMAIndicator) Validate(params map[string]interface{}) error {
	period := getIntParam(params, "period", 12)
	if period < 1 || period > 250 {
		return fmt.Errorf("period必须在1到250之间")
	}
	return nil
}

// Calculate 计算EMA
func (e *EMAIndicator) Calculate(klines []models.KLine, params map[string]interface{}) (*models.IndicatorResult, error) {
	startTime := time.Now()
	
	period := getIntParam(params, "period", 12)
	if err := e.Validate(params); err != nil {
		return nil, err
	}
	
	if len(klines) < period {
		return nil, fmt.Errorf("K线数据不足，需要至少%d根K线", period)
	}
	
	values := make([]models.IndicatorValue, len(klines))
	
	// EMA平滑系数
	multiplier := 2.0 / float64(period+1)
	
	// 第一个EMA使用SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += klines[i].Close
		values[i] = models.IndicatorValue{
			Timestamp: klines[i].Timestamp,
			Value:     0,
		}
	}
	ema := sum / float64(period)
	values[period-1].Value = ema
	
	// 计算后续EMA
	for i := period; i < len(klines); i++ {
		ema = (klines[i].Close-ema)*multiplier + ema
		values[i] = models.IndicatorValue{
			Timestamp: klines[i].Timestamp,
			Value:     ema,
		}
	}
	
	return &models.IndicatorResult{
		Type: models.IndicatorTypeEMA,
		Name: fmt.Sprintf("EMA(%d)", period),
		Series: []models.IndicatorSeries{
			{
				Name:   fmt.Sprintf("EMA%d", period),
				Values: values,
			},
		},
		ComputationTime: time.Since(startTime).Milliseconds(),
	}, nil
}

// BOLLIndicator 布林带指标
type BOLLIndicator struct{}

// NewBOLLIndicator 创建BOLL指标
func NewBOLLIndicator() calculator.Indicator {
	return &BOLLIndicator{}
}

// Metadata 获取元信息
func (b *BOLLIndicator) Metadata() models.IndicatorMetadata {
	return models.IndicatorMetadata{
		Type:        models.IndicatorTypeBOLL,
		Name:        "布林带",
		Category:    models.CategoryBasic,
		Description: "布林带(Bollinger Bands)，由中轨(MA)和上下轨(标准差)组成",
		ParamsDef: []models.ParameterDef{
			{
				Name:         "period",
				Type:         "int",
				Required:     false,
				DefaultValue: 20,
				Min:          2,
				Max:          250,
				Description:  "计算周期",
			},
			{
				Name:         "std_dev",
				Type:         "float",
				Required:     false,
				DefaultValue: 2.0,
				Min:          0.5,
				Max:          5.0,
				Description:  "标准差倍数",
			},
		},
	}
}

// Validate 验证参数
func (b *BOLLIndicator) Validate(params map[string]interface{}) error {
	period := getIntParam(params, "period", 20)
	if period < 2 || period > 250 {
		return fmt.Errorf("period必须在2到250之间")
	}
	stdDev := getFloatParam(params, "std_dev", 2.0)
	if stdDev < 0.5 || stdDev > 5.0 {
		return fmt.Errorf("std_dev必须在0.5到5.0之间")
	}
	return nil
}

// Calculate 计算布林带
func (b *BOLLIndicator) Calculate(klines []models.KLine, params map[string]interface{}) (*models.IndicatorResult, error) {
	startTime := time.Now()
	
	period := getIntParam(params, "period", 20)
	stdDevMultiplier := getFloatParam(params, "std_dev", 2.0)
	
	if err := b.Validate(params); err != nil {
		return nil, err
	}
	
	if len(klines) < period {
		return nil, fmt.Errorf("K线数据不足，需要至少%d根K线", period)
	}
	
	midValues := make([]models.IndicatorValue, len(klines))
	upperValues := make([]models.IndicatorValue, len(klines))
	lowerValues := make([]models.IndicatorValue, len(klines))
	
	for i := 0; i < len(klines); i++ {
		if i < period-1 {
			midValues[i] = models.IndicatorValue{Timestamp: klines[i].Timestamp, Value: 0}
			upperValues[i] = models.IndicatorValue{Timestamp: klines[i].Timestamp, Value: 0}
			lowerValues[i] = models.IndicatorValue{Timestamp: klines[i].Timestamp, Value: 0}
			continue
		}
		
		// 计算MA
		sum := 0.0
		for j := 0; j < period; j++ {
			sum += klines[i-j].Close
		}
		ma := sum / float64(period)
		
		// 计算标准差
		variance := 0.0
		for j := 0; j < period; j++ {
			diff := klines[i-j].Close - ma
			variance += diff * diff
		}
		stdDev := sqrt(variance / float64(period))
		
		midValues[i] = models.IndicatorValue{Timestamp: klines[i].Timestamp, Value: ma}
		upperValues[i] = models.IndicatorValue{Timestamp: klines[i].Timestamp, Value: ma + stdDevMultiplier*stdDev}
		lowerValues[i] = models.IndicatorValue{Timestamp: klines[i].Timestamp, Value: ma - stdDevMultiplier*stdDev}
	}
	
	return &models.IndicatorResult{
		Type: models.IndicatorTypeBOLL,
		Name: fmt.Sprintf("BOLL(%d,%.1f)", period, stdDevMultiplier),
		Series: []models.IndicatorSeries{
			{Name: "MID", Values: midValues},
			{Name: "UPPER", Values: upperValues},
			{Name: "LOWER", Values: lowerValues},
		},
		ComputationTime: time.Since(startTime).Milliseconds(),
	}, nil
}

// sqrt 简单的平方根实现
func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x == 0 {
		return 0
	}
	
	z := x / 2
	for i := 0; i < 100; i++ {
		zNew := (z + x/z) / 2
		if abs(zNew-z) < 1e-10 {
			break
		}
		z = zNew
	}
	return z
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
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

// getFloatParam 获取浮点数参数
func getFloatParam(params map[string]interface{}, key string, defaultValue float64) float64 {
	if params == nil {
		return defaultValue
	}
	v, ok := params[key]
	if !ok {
		return defaultValue
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return defaultValue
	}
}

// 初始化时注册指标
func init() {
	calculator.Register(models.IndicatorTypeMA, NewMAIndicator)
	calculator.Register(models.IndicatorTypeEMA, NewEMAIndicator)
	calculator.Register(models.IndicatorTypeBOLL, NewBOLLIndicator)
}
