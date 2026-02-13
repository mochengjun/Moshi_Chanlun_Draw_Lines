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

// MarketCode 市场代码
type MarketCode = int

const (
	// 上海市场
	MarketSHIndex  MarketCode = 0    // 上证指数
	MarketSHA      MarketCode = 1    // 上证A股
	MarketSHB      MarketCode = 2    // 上证B股
	MarketSHFund   MarketCode = 3    // 上证基金
	MarketSHBond1  MarketCode = 4    // 上证国债
	MarketSHBond2  MarketCode = 5    // 上证债券
	MarketSHWarrant MarketCode = 6   // 上证权证
	MarketSTAR     MarketCode = 7    // 科创板
	MarketSHOther  MarketCode = 101  // 上证其他

	// 深圳市场
	MarketSZIndex  MarketCode = 1000 // 深证指数
	MarketSZA      MarketCode = 1001 // 深证A股
	MarketSZB      MarketCode = 1002 // 深证B股
	MarketSZFund   MarketCode = 1003 // 深证基金
	MarketSME      MarketCode = 1004 // 中小企业板
	MarketSZBond1  MarketCode = 1005 // 深证国债
	MarketSZBond2  MarketCode = 1006 // 深证债券
	MarketSZWarrant MarketCode = 1007 // 深证权证
	MarketGEM      MarketCode = 1008 // 创业板
	MarketSZOther  MarketCode = 1101 // 深证其他

	// 期货市场
	MarketSHFE     MarketCode = 3000 // 上海期货
	MarketDCE      MarketCode = 3001 // 大连商品
	MarketCZCE     MarketCode = 3002 // 郑州商品
	MarketCFFEX    MarketCode = 3003 // 中金所
	MarketSHFENight MarketCode = 3009 // 上期所夜盘
)

// ResolveMarket 根据股票代码自动识别市场代码
// 当前端传入的market为简化值(0=指数/深市,1=沪市)时，根据code前缀映射为外部API需要的精确市场代码
func ResolveMarket(market int, code string) int {
	// 如果已经是精确市场代码（>=2），直接使用
	if market >= 2 {
		return market
	}

	if len(code) < 3 {
		return market
	}

	prefix3 := code[:3]
	prefix2 := code[:2]

	switch {
	// 创业板: 300xxx, 301xxx
	case prefix3 == "300" || prefix3 == "301":
		return MarketGEM
	// 科创板: 688xxx, 689xxx
	case prefix3 == "688" || prefix3 == "689":
		return MarketSTAR
	// 深证A股: 002xxx, 003xxx (中小企业板已并入深市主板)
	case prefix3 == "002" || prefix3 == "003":
		return MarketSZA
	// 深证指数: 399xxx
	case prefix3 == "399":
		return MarketSZIndex
	// 上证B股: 900xxx
	case prefix3 == "900":
		return MarketSHB
	// 深证B股: 200xxx
	case prefix3 == "200":
		return MarketSZB
	// 沪市个股: 600xxx, 601xxx, 603xxx, 605xxx
	case prefix2 == "60":
		return MarketSHA
	// 上证指数: 000xxx (当market=0时视为指数)
	case prefix3 == "000" && market == 0:
		return MarketSHIndex
	// 深证A股: 000xxx, 001xxx (当market=1001或显式深市)
	case prefix3 == "000" || prefix3 == "001":
		if market == 1 {
			return MarketSZA
		}
		return MarketSHIndex
	default:
		return market
	}
}

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
