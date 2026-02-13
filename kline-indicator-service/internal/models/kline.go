package models

// KLineType K线类型
type KLineType int

const (
	KLineType1Min   KLineType = 1  // 1分钟
	KLineType3Min   KLineType = 4  // 3分钟
	KLineType5Min   KLineType = 2  // 5分钟
	KLineType15Min  KLineType = 5  // 15分钟
	KLineType30Min  KLineType = 6  // 30分钟
	KLineType60Min  KLineType = 3  // 60分钟
	KLineType120Min KLineType = 8  // 120分钟
	KLineTypeHalfDay KLineType = 7 // 半日线
	KLineTypeDay    KLineType = 10 // 日K
	KLineTypeWeek   KLineType = 11 // 周K
	KLineTypeMonth  KLineType = 20 // 月K
	KLineTypeSeason KLineType = 21 // 季K
	KLineTypeYear   KLineType = 30 // 年K
)

// WeightType 复权类型
type WeightType int

const (
	WeightNone    WeightType = 0 // 不复权
	WeightForward WeightType = 1 // 前复权
	WeightBack    WeightType = 2 // 后复权
)

// KLine K线数据结构
type KLine struct {
	Timestamp string  `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	Amount    float64 `json:"amount,omitempty"`
}

// IsYang 是否为阳线
func (k *KLine) IsYang() bool {
	return k.Close > k.Open
}

// IsYin 是否为阴线
func (k *KLine) IsYin() bool {
	return k.Close < k.Open
}

// BodySize 实体大小
func (k *KLine) BodySize() float64 {
	if k.Close > k.Open {
		return k.Close - k.Open
	}
	return k.Open - k.Close
}

// UpperShadow 上影线长度
func (k *KLine) UpperShadow() float64 {
	if k.Close > k.Open {
		return k.High - k.Close
	}
	return k.High - k.Open
}

// LowerShadow 下影线长度
func (k *KLine) LowerShadow() float64 {
	if k.Close > k.Open {
		return k.Open - k.Low
	}
	return k.Close - k.Low
}

// IsLimitUp 是否涨停
func (k *KLine) IsLimitUp(prevClose float64, limitPct float64) bool {
	if limitPct == 0 {
		limitPct = 0.1
	}
	limitPrice := prevClose * (1 + limitPct)
	return k.Close >= limitPrice*0.99 // 允许0.01误差
}

// KLineRequest K线请求参数
type KLineRequest struct {
	Market    int       `form:"market" json:"market"`       // 市场代码
	Code      string    `form:"code" json:"code"`           // 股票代码
	KLineType KLineType `form:"klinetype" json:"klinetype"` // K线类型
	Weight    WeightType `form:"weight" json:"weight"`       // 复权方式
	Count     int       `form:"count" json:"count"`         // 数据条数
	EndTime   string    `form:"endtime" json:"endtime"`     // 结束时间
}

// KLineData K线数据响应
type KLineData struct {
	Market    int       `json:"market"`
	Code      string    `json:"code"`
	Name      string    `json:"name,omitempty"`
	KLineType KLineType `json:"klinetype"`
	Weight    WeightType `json:"weight"`
	KLines    []KLine   `json:"klines"`
	Count     int       `json:"count"`
}

// ExternalAPIRequest 外部API请求格式
type ExternalAPIRequest struct {
	ReqType int                    `json:"reqtype"`
	ReqID   int                    `json:"reqid"`
	Session string                 `json:"session"`
	Data    ExternalAPIRequestData `json:"data"`
}

// ExternalAPIRequestData 外部API请求数据
type ExternalAPIRequestData struct {
	Market    int    `json:"market"`
	Code      string `json:"code"`
	KLineType int    `json:"klinetype"`
	Weight    int    `json:"weight"`
	TimeType  int    `json:"timetype"`
	Time0     string `json:"time0"`
	Time1     string `json:"time1,omitempty"`
	Count     int    `json:"count"`
}

// ExternalAPIResponse 外部API响应格式
type ExternalAPIResponse struct {
	Status     int                      `json:"status"`
	Msg        string                   `json:"msg"`
	ServerTime string                   `json:"servertime"`
	ReqType    int                      `json:"reqtype"`
	ReqID      int                      `json:"reqid"`
	Data       ExternalAPIResponseData  `json:"data"`
}

// ExternalAPIResponseData 外部API响应数据
type ExternalAPIResponseData struct {
	Market    int             `json:"market"`
	Code      string          `json:"code"`
	KLineType int             `json:"klinetype"`
	Weight    int             `json:"weight"`
	LastUDate string          `json:"lastoldtime"`
	KList     [][]interface{} `json:"kline"`
}
