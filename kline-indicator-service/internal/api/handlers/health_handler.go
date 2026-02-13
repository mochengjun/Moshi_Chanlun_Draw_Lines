package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"kline-indicator-service/internal/models"
	"kline-indicator-service/internal/service"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	cacheService *service.CacheService
	klineService *service.KLineService
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(cacheService *service.CacheService, klineService *service.KLineService) *HealthHandler {
	return &HealthHandler{
		cacheService: cacheService,
		klineService: klineService,
	}
}

// Health 健康检查
// @Summary 健康检查
// @Description 检查服务健康状态
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} models.HealthResponse
// @Router /api/v1/health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	// 获取服务状态
	services := h.cacheService.Health(c.Request.Context())
	services["indicator_engine"] = "up"
	
	// 获取缓存统计
	memoryHits, memoryMisses, memorySize, hitRate := h.klineService.GetCacheStats()
	
	status := "healthy"
	for _, v := range services {
		if v != "up" {
			status = "degraded"
			break
		}
	}
	
	c.JSON(http.StatusOK, &models.HealthResponse{
		Status:    status,
		Timestamp: time.Now().Format(time.RFC3339),
		Services:  services,
		Metrics: &models.HealthMetrics{
			TotalRequests:     memoryHits + memoryMisses,
			CacheHitRate:      hitRate,
			AvgResponseTimeMs: 0, // TODO: 实现响应时间统计
			ActiveConnections: memorySize,
		},
	})
}

// Ping 简单的存活检查
// @Summary Ping检查
// @Description 简单的存活检查
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {string} string "pong"
// @Router /ping [get]
func (h *HealthHandler) Ping(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}
