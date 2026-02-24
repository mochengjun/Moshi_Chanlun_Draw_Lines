package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		
		// 处理请求
		c.Next()
		
		// 计算延迟
		latency := time.Since(startTime)
		
		// 获取状态码
		statusCode := c.Writer.Status()
		
		// 获取请求IP
		clientIP := c.ClientIP()
		
		// 获取请求方法
		method := c.Request.Method
		
		// 如果有查询参数，拼接到path
		if raw != "" {
			path = path + "?" + raw
		}
		
		// 记录日志
		log.Printf("[GIN] %v | %3d | %13v | %15s | %-7s %s",
			startTime.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)
	}
}

// RequestID 请求ID中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取或生成请求ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		
		// 设置到上下文
		c.Set("RequestID", requestID)
		
		// 设置到响应头
		c.Header("X-Request-ID", requestID)
		
		c.Next()
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + randomString(8)
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
