package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"kline-indicator-service/internal/models"
)

// RateLimiter 限流器
type RateLimiter struct {
	rate       float64        // 每秒允许的请求数
	burst      int            // 突发容量
	tokens     float64        // 当前令牌数
	lastUpdate time.Time      // 上次更新时间
	mu         sync.Mutex
}

// NewRateLimiter 创建限流器
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst),
		lastUpdate: time.Now(),
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(rl.lastUpdate).Seconds()
	rl.lastUpdate = now
	
	// 添加令牌
	rl.tokens += elapsed * rl.rate
	if rl.tokens > float64(rl.burst) {
		rl.tokens = float64(rl.burst)
	}
	
	// 消耗令牌
	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}
	
	return false
}

// Remaining 返回剩余令牌数
func (rl *RateLimiter) Remaining() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return int(rl.tokens)
}

// globalLimiter 全局限流器
var globalLimiter *RateLimiter

// InitRateLimiter 初始化全局限流器
func InitRateLimiter(rate float64, burst int) {
	globalLimiter = NewRateLimiter(rate, burst)
}

// RateLimit 限流中间件
func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalLimiter == nil {
			c.Next()
			return
		}
		
		if !globalLimiter.Allow() {
			c.Header("X-Rate-Limit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, models.NewErrorResponse(
				models.ErrCodeTooManyRequests,
				models.ErrMsgTooManyRequests,
			))
			c.Abort()
			return
		}
		
		// 设置剩余配额
		c.Header("X-Rate-Limit-Remaining", string(rune(globalLimiter.Remaining())))
		
		c.Next()
	}
}

// IPRateLimiter IP级别限流器
type IPRateLimiter struct {
	limiters map[string]*RateLimiter
	rate     float64
	burst    int
	mu       sync.RWMutex
}

// NewIPRateLimiter 创建IP限流器
func NewIPRateLimiter(rate float64, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*RateLimiter),
		rate:     rate,
		burst:    burst,
	}
}

// GetLimiter 获取IP对应的限流器
func (ipl *IPRateLimiter) GetLimiter(ip string) *RateLimiter {
	ipl.mu.RLock()
	limiter, exists := ipl.limiters[ip]
	ipl.mu.RUnlock()
	
	if exists {
		return limiter
	}
	
	ipl.mu.Lock()
	defer ipl.mu.Unlock()
	
	// 双重检查
	if limiter, exists = ipl.limiters[ip]; exists {
		return limiter
	}
	
	limiter = NewRateLimiter(ipl.rate, ipl.burst)
	ipl.limiters[ip] = limiter
	
	return limiter
}

// IPRateLimit IP级别限流中间件
func IPRateLimit(rate float64, burst int) gin.HandlerFunc {
	ipLimiter := NewIPRateLimiter(rate, burst)
	
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := ipLimiter.GetLimiter(ip)
		
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, models.NewErrorResponse(
				models.ErrCodeTooManyRequests,
				models.ErrMsgTooManyRequests,
			))
			c.Abort()
			return
		}
		
		c.Next()
	}
}
