package models

// Response 通用API响应
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}) *Response {
	return &Response{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code int, message string) *Response {
	return &Response{
		Code:    code,
		Message: message,
	}
}

// KLineResponse K线数据响应
type KLineResponse struct {
	Code     int        `json:"code"`
	Message  string     `json:"message"`
	Data     *KLineData `json:"data,omitempty"`
	CacheHit bool       `json:"cache_hit"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Metrics   *HealthMetrics    `json:"metrics,omitempty"`
}

// HealthMetrics 健康指标
type HealthMetrics struct {
	TotalRequests      int64   `json:"total_requests"`
	CacheHitRate       float64 `json:"cache_hit_rate"`
	AvgResponseTimeMs  float64 `json:"avg_response_time_ms"`
	ActiveConnections  int     `json:"active_connections"`
	ExternalAPILatency float64 `json:"external_api_latency_ms"`
}

// IndicatorListResponse 指标列表响应
type IndicatorListResponse struct {
	Code       int                 `json:"code"`
	Message    string              `json:"message"`
	Indicators []IndicatorMetadata `json:"indicators"`
}

// ErrorCode 错误码定义
const (
	ErrCodeSuccess          = 0
	ErrCodeBadRequest       = 400
	ErrCodeUnauthorized     = 401
	ErrCodeForbidden        = 403
	ErrCodeNotFound         = 404
	ErrCodeTooManyRequests  = 429
	ErrCodeInternalError    = 500
	ErrCodeExternalAPIError = 502
	ErrCodeServiceUnavailable = 503
)

// 错误消息
const (
	ErrMsgBadRequest       = "请求参数错误"
	ErrMsgUnauthorized     = "未授权访问"
	ErrMsgForbidden        = "禁止访问"
	ErrMsgNotFound         = "资源不存在"
	ErrMsgTooManyRequests  = "请求过于频繁"
	ErrMsgInternalError    = "服务器内部错误"
	ErrMsgExternalAPIError = "外部API调用失败"
	ErrMsgServiceUnavailable = "服务暂时不可用"
)
