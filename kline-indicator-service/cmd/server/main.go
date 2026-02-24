package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"kline-indicator-service/internal/api"
	"kline-indicator-service/internal/api/handlers"
	"kline-indicator-service/internal/api/middleware"
	"kline-indicator-service/internal/cache"
	"kline-indicator-service/internal/client"
	"kline-indicator-service/internal/service"
)

// Config 应用配置
type Config struct {
	ServerPort        string
	ServerMode        string
	ExternalAPIURL    string
	ExternalAPITimeout time.Duration
	MaxRetries        int
	MemoryCacheSize   int
	MemoryCacheTTL    time.Duration
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	RedisCacheTTL     time.Duration
	RateLimit         float64
	RateBurst         int
	JWTSecret         string
	EnableAuth        bool
}

func main() {
	// 加载配置
	cfg := loadConfig()
	
	// 设置Gin模式
	if cfg.ServerMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	
	// 初始化限流器
	middleware.InitRateLimiter(cfg.RateLimit, cfg.RateBurst)
	
	// 创建外部API客户端
	externalClient := client.NewExternalAPIClient(
		cfg.ExternalAPIURL,
		cfg.ExternalAPITimeout,
		cfg.MaxRetries,
	)
	
	// 创建内存缓存
	memoryCache := cache.NewMemoryCache(cfg.MemoryCacheSize, cfg.MemoryCacheTTL)
	
	// 创建Redis缓存（可选）
	var redisCache *cache.RedisCache
	if cfg.RedisAddr != "" {
		var err error
		redisCache, err = cache.NewRedisCache(cache.RedisConfig{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
			PoolSize: 100,
		}, cfg.RedisCacheTTL)
		if err != nil {
			log.Printf("警告: Redis连接失败，将仅使用内存缓存: %v", err)
			redisCache = nil
		} else {
			log.Printf("Redis连接成功: %s", cfg.RedisAddr)
		}
	}
	
	// 创建服务
	klineService := service.NewKLineService(externalClient, memoryCache, redisCache)
	cacheService := service.NewCacheService(memoryCache, redisCache)
	indicatorService := service.NewIndicatorService(klineService, cacheService)
	
	// 创建处理器
	klineHandler := handlers.NewKLineHandler(klineService)
	indicatorHandler := handlers.NewIndicatorHandler(indicatorService)
	healthHandler := handlers.NewHealthHandler(cacheService, klineService)
	wsHandler := handlers.NewWSHandler(klineService, indicatorService)
	
	// 设置路由
	routerConfig := &api.RouterConfig{
		KLineHandler:     klineHandler,
		IndicatorHandler: indicatorHandler,
		HealthHandler:    healthHandler,
		WSHandler:        wsHandler,
	}
	
	var router *gin.Engine
	if cfg.EnableAuth {
		router = api.SetupRouterWithAuth(routerConfig, cfg.JWTSecret)
	} else {
		router = api.SetupRouter(routerConfig)
	}
	
	// 创建HTTP服务器
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}
	
	// 启动服务器
	go func() {
		log.Printf("服务器启动在端口 %s", cfg.ServerPort)
		log.Printf("健康检查: http://localhost:%s/api/v1/health", cfg.ServerPort)
		log.Printf("K线接口: http://localhost:%s/api/v1/kline", cfg.ServerPort)
		log.Printf("指标计算: http://localhost:%s/api/v1/indicators/calculate", cfg.ServerPort)
		log.Printf("WebSocket: ws://localhost:%s/ws/kline", cfg.ServerPort)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()
	
	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("正在关闭服务器...")
	
	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("服务器关闭错误: %v", err)
	}
	
	// 关闭Redis连接
	if redisCache != nil {
		redisCache.Close()
	}
	
	// 关闭内存缓存清理协程
	memoryCache.Close()
	
	// 关闭WebSocket心跳协程
	wsHandler.Close()
	
	log.Println("服务器已关闭")
}

// YAMLConfig YAML配置文件结构
type YAMLConfig struct {
	Server struct {
		Port string `yaml:"port"`
		Mode string `yaml:"mode"`
	} `yaml:"server"`
	ExternalAPI struct {
		URL        string `yaml:"url"`
		Timeout    string `yaml:"timeout"`
		MaxRetries int    `yaml:"max_retries"`
	} `yaml:"external_api"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
		PoolSize int    `yaml:"pool_size"`
	} `yaml:"redis"`
	Cache struct {
		MemorySize int    `yaml:"memory_size"`
		MemoryTTL  string `yaml:"memory_ttl"`
		RedisTTL   string `yaml:"redis_ttl"`
	} `yaml:"cache"`
	RateLimit struct {
		RequestsPerSecond float64 `yaml:"requests_per_second"`
		Burst             int     `yaml:"burst"`
	} `yaml:"rate_limit"`
	Auth struct {
		JWTSecret   string `yaml:"jwt_secret"`
		TokenExpire string `yaml:"token_expire"`
	} `yaml:"auth"`
}

// loadConfig 加载配置
func loadConfig() *Config {
	// 首先尝试从YAML文件加载配置
	yamlCfg := loadYAMLConfig()
	
	// 使用YAML配置作为默认值，环境变量可以覆盖
	defaultAPIURL := "http://localhost:9000"
	if yamlCfg != nil && yamlCfg.ExternalAPI.URL != "" {
		defaultAPIURL = yamlCfg.ExternalAPI.URL
	}
	
	defaultPort := "8080"
	if yamlCfg != nil && yamlCfg.Server.Port != "" {
		defaultPort = yamlCfg.Server.Port
	}
	
	defaultMode := "debug"
	if yamlCfg != nil && yamlCfg.Server.Mode != "" {
		defaultMode = yamlCfg.Server.Mode
	}
	
	defaultTimeout := 5 * time.Second
	if yamlCfg != nil && yamlCfg.ExternalAPI.Timeout != "" {
		if d, err := time.ParseDuration(yamlCfg.ExternalAPI.Timeout); err == nil {
			defaultTimeout = d
		}
	}
	
	defaultMaxRetries := 3
	if yamlCfg != nil && yamlCfg.ExternalAPI.MaxRetries > 0 {
		defaultMaxRetries = yamlCfg.ExternalAPI.MaxRetries
	}
	
	defaultRedisAddr := ""
	if yamlCfg != nil && yamlCfg.Redis.Addr != "" {
		defaultRedisAddr = yamlCfg.Redis.Addr
	}
	
	defaultCacheSize := 1000
	if yamlCfg != nil && yamlCfg.Cache.MemorySize > 0 {
		defaultCacheSize = yamlCfg.Cache.MemorySize
	}
	
	defaultMemoryTTL := 5 * time.Minute
	if yamlCfg != nil && yamlCfg.Cache.MemoryTTL != "" {
		if d, err := time.ParseDuration(yamlCfg.Cache.MemoryTTL); err == nil {
			defaultMemoryTTL = d
		}
	}
	
	defaultRedisTTL := 30 * time.Minute
	if yamlCfg != nil && yamlCfg.Cache.RedisTTL != "" {
		if d, err := time.ParseDuration(yamlCfg.Cache.RedisTTL); err == nil {
			defaultRedisTTL = d
		}
	}
	
	defaultRateLimit := 100.0
	if yamlCfg != nil && yamlCfg.RateLimit.RequestsPerSecond > 0 {
		defaultRateLimit = yamlCfg.RateLimit.RequestsPerSecond
	}
	
	defaultBurst := 200
	if yamlCfg != nil && yamlCfg.RateLimit.Burst > 0 {
		defaultBurst = yamlCfg.RateLimit.Burst
	}
	
	defaultJWTSecret := "your-secret-key"
	if yamlCfg != nil && yamlCfg.Auth.JWTSecret != "" {
		defaultJWTSecret = yamlCfg.Auth.JWTSecret
	}
	
	cfg := &Config{
		ServerPort:         getEnv("SERVER_PORT", defaultPort),
		ServerMode:         getEnv("SERVER_MODE", defaultMode),
		ExternalAPIURL:     getEnv("EXTERNAL_API_URL", defaultAPIURL),
		ExternalAPITimeout: getDurationEnv("EXTERNAL_API_TIMEOUT", defaultTimeout),
		MaxRetries:         getIntEnv("MAX_RETRIES", defaultMaxRetries),
		MemoryCacheSize:    getIntEnv("MEMORY_CACHE_SIZE", defaultCacheSize),
		MemoryCacheTTL:     getDurationEnv("MEMORY_CACHE_TTL", defaultMemoryTTL),
		RedisAddr:          getEnv("REDIS_ADDR", defaultRedisAddr),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getIntEnv("REDIS_DB", 0),
		RedisCacheTTL:      getDurationEnv("REDIS_CACHE_TTL", defaultRedisTTL),
		RateLimit:          getFloatEnv("RATE_LIMIT", defaultRateLimit),
		RateBurst:          getIntEnv("RATE_BURST", defaultBurst),
		JWTSecret:          getEnv("JWT_SECRET", defaultJWTSecret),
		EnableAuth:         getBoolEnv("ENABLE_AUTH", false),
	}
	
	log.Printf("配置加载完成: ExternalAPIURL=%s", cfg.ExternalAPIURL)
	return cfg
}

// loadYAMLConfig 从YAML文件加载配置
func loadYAMLConfig() *YAMLConfig {
	// 尝试多个可能的配置文件路径
	configPaths := []string{
		"config/config.yaml",
		"../config/config.yaml",
		"../../config/config.yaml",
		"config.yaml",
	}
	
	for _, path := range configPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		
		var cfg YAMLConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			log.Printf("警告: 解析配置文件 %s 失败: %v", path, err)
			continue
		}
		
		log.Printf("已从 %s 加载配置文件", path)
		return &cfg
	}
	
	log.Printf("警告: 未找到配置文件，使用默认值")
	return nil
}

// getEnv 获取环境变量
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv 获取整数环境变量
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
	}
	return defaultValue
}

// getFloatEnv 获取浮点数环境变量
func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		var f float64
		if _, err := fmt.Sscanf(value, "%f", &f); err == nil {
			return f
		}
	}
	return defaultValue
}

// getBoolEnv 获取布尔环境变量
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

// getDurationEnv 获取时间环境变量
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
