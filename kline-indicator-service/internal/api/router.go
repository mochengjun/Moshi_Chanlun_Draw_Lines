package api

import (
	"github.com/gin-gonic/gin"

	"kline-indicator-service/internal/api/handlers"
	"kline-indicator-service/internal/api/middleware"
)

// RouterConfig 路由配置
type RouterConfig struct {
	KLineHandler     *handlers.KLineHandler
	IndicatorHandler *handlers.IndicatorHandler
	HealthHandler    *handlers.HealthHandler
	WSHandler        *handlers.WSHandler
}

// SetupRouter 设置路由
func SetupRouter(cfg *RouterConfig) *gin.Engine {
	router := gin.New()
	
	// 全局中间件
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())
	router.Use(middleware.RateLimit())
	
	// Ping路由（不需要认证）
	router.GET("/ping", cfg.HealthHandler.Ping)
	
	// API v1路由组
	v1 := router.Group("/api/v1")
	{
		// 健康检查（不需要认证）
		v1.GET("/health", cfg.HealthHandler.Health)
		
		// K线数据接口
		v1.GET("/kline", cfg.KLineHandler.GetKLine)
		
		// 指标接口
		indicators := v1.Group("/indicators")
		{
			indicators.GET("/list", cfg.IndicatorHandler.ListIndicators)
			indicators.POST("/calculate", cfg.IndicatorHandler.Calculate)
		}
	}
	
	// WebSocket路由
	if cfg.WSHandler != nil {
		router.GET("/ws/kline", cfg.WSHandler.HandleKLine)
	}
	
	return router
}

// SetupRouterWithAuth 设置带认证的路由
func SetupRouterWithAuth(cfg *RouterConfig, jwtSecret string) *gin.Engine {
	// 初始化JWT
	middleware.InitJWT(jwtSecret, []string{
		"/ping",
		"/api/v1/health",
		"/api/v1/indicators/list",
	})
	
	router := gin.New()
	
	// 全局中间件
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())
	router.Use(middleware.RateLimit())
	
	// Ping路由
	router.GET("/ping", cfg.HealthHandler.Ping)
	
	// API v1路由组
	v1 := router.Group("/api/v1")
	{
		// 健康检查
		v1.GET("/health", cfg.HealthHandler.Health)
		
		// 指标列表（不需要认证）
		v1.GET("/indicators/list", cfg.IndicatorHandler.ListIndicators)
		
		// 需要认证的路由
		auth := v1.Group("")
		auth.Use(middleware.JWTAuth())
		{
			// K线数据接口
			auth.GET("/kline", cfg.KLineHandler.GetKLine)
			
			// 指标计算接口
			auth.POST("/indicators/calculate", cfg.IndicatorHandler.Calculate)
		}
	}
	
	// WebSocket路由
	if cfg.WSHandler != nil {
		router.GET("/ws/kline", cfg.WSHandler.HandleKLine)
	}
	
	return router
}
