package calculator

import (
	"fmt"
)

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

// getBoolParam 获取布尔参数
func getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if params == nil {
		return defaultValue
	}
	
	v, ok := params[key]
	if !ok {
		return defaultValue
	}
	
	switch val := v.(type) {
	case bool:
		return val
	default:
		return defaultValue
	}
}

// validateRange 验证范围
func validateRange(value, min, max int, name string) error {
	if value < min || value > max {
		return fmt.Errorf("%s必须在%d到%d之间，当前值: %d", name, min, max, value)
	}
	return nil
}

// max 返回两个数的最大值
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// min 返回两个数的最小值
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// abs 返回绝对值
func abs(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}
