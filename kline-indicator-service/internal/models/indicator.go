package models

// IndicatorType 指标类型
type IndicatorType string

const (
	IndicatorTypeMoshiChanlun   IndicatorType = "moshi_chanlun"   // 莫氏缠论
	IndicatorTypeChanlunFractal IndicatorType = "chanlun_fractal" // 缠论分型
	IndicatorTypeChanlunBi      IndicatorType = "chanlun_bi"      // 缠论笔
	IndicatorTypeMA             IndicatorType = "ma"              // 移动平均线
	IndicatorTypeEMA            IndicatorType = "ema"             // 指数移动平均线
	IndicatorTypeBOLL           IndicatorType = "boll"            // 布林带
)

// IndicatorCategory 指标分类
type IndicatorCategory string

const (
	CategoryChanlun IndicatorCategory = "chanlun" // 缠论指标
	CategoryBasic   IndicatorCategory = "basic"   // 基础指标
)

// ParameterDef 参数定义
type ParameterDef struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // int, float, bool, string
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value"`
	Min          interface{} `json:"min,omitempty"`
	Max          interface{} `json:"max,omitempty"`
	Description  string      `json:"description"`
}

// IndicatorMetadata 指标元信息
type IndicatorMetadata struct {
	Type        IndicatorType     `json:"type"`
	Name        string            `json:"name"`
	Category    IndicatorCategory `json:"category"`
	Description string            `json:"description"`
	ParamsDef   []ParameterDef    `json:"params_def"`
}

// IndicatorConfig 指标配置
type IndicatorConfig struct {
	Type   IndicatorType          `json:"type"`
	Params map[string]interface{} `json:"params"`
}

// IndicatorValue 指标数值（用于线形指标）
type IndicatorValue struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

// IndicatorSeries 指标序列（用于多值指标如BOLL、MACD）
type IndicatorSeries struct {
	Name   string           `json:"name"`
	Values []IndicatorValue `json:"values"`
}

// FractalType 分型类型
type FractalType string

const (
	FractalTypeTop    FractalType = "TOP"
	FractalTypeBottom FractalType = "BOTTOM"
)

// FractalMarker 分型标记
type FractalMarker struct {
	Index     int         `json:"index"`
	Timestamp string      `json:"timestamp"`
	Type      string      `json:"type"`  // "top" or "bottom"
	Price     float64     `json:"price"`
	Zone      [2]float64  `json:"zone,omitempty"` // [下沿, 上沿]
}

// BiDirection 笔方向
type BiDirection string

const (
	BiDirectionUp   BiDirection = "UP"
	BiDirectionDown BiDirection = "DOWN"
)

// BiMarker 笔标记
type BiMarker struct {
	StartIndex        int     `json:"start_index"`
	EndIndex          int     `json:"end_index"`
	StartTimestamp    string  `json:"start_timestamp"`
	EndTimestamp      string  `json:"end_timestamp"`
	StartPrice        float64 `json:"start_price"`
	EndPrice          float64 `json:"end_price"`
	Direction         string  `json:"direction"`                    // "up" or "down"
	Length            int     `json:"length,omitempty"`             // 包含K线数
	UpCount           int     `json:"up_count,omitempty"`           // 上涨K线数量
	DownCount         int     `json:"down_count,omitempty"`         // 下跌K线数量
	Multiplier        int     `json:"multiplier"`                   // 级别倍数 0/1/2/4/8
	ActualRetraceTime float64 `json:"actual_retrace_time,omitempty"` // 实际回调/反弹时间(分钟)
}

// IndicatorResult 指标计算结果
type IndicatorResult struct {
	Type            IndicatorType          `json:"type"`
	Name            string                 `json:"name,omitempty"`
	Series          []IndicatorSeries      `json:"series,omitempty"`           // 线形数据
	FractalMarkers  []FractalMarker        `json:"fractal_markers,omitempty"`  // 分型标记
	BiMarkers       []BiMarker             `json:"bi_markers,omitempty"`       // 笔标记
	Extra           map[string]interface{} `json:"extra,omitempty"`            // 扩展数据
	ComputationTime int64                  `json:"computation_time_ms"`        // 计算耗时
}

// ParamDef 参数定义 (别名)
type ParamDef = ParameterDef

// IndicatorCalculateRequest 指标计算请求
type IndicatorCalculateRequest struct {
	Market     int               `json:"market"`
	Code       string            `json:"code"`
	KLineType  KLineType         `json:"klinetype"`
	Weight     WeightType        `json:"weight"`
	Count      int               `json:"count"`
	Indicators []IndicatorConfig `json:"indicators"`
}

// IndicatorCalculateResponse 指标计算响应
type IndicatorCalculateResponse struct {
	Code            int               `json:"code"`
	Message         string            `json:"message"`
	StockCode       string            `json:"stock_code"`
	Indicators      []IndicatorResult `json:"indicators"`
	ComputationTime int64             `json:"computation_time_ms"`
}
