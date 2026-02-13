package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"kline-indicator-service/internal/cache"
	"kline-indicator-service/internal/client"
	"kline-indicator-service/internal/models"
)

// KLineService K线服务
type KLineService struct {
	externalClient *client.ExternalAPIClient
	memoryCache    *cache.MemoryCache
	redisCache     *cache.RedisCache
	useRedis       bool
}

// NewKLineService 创建K线服务
func NewKLineService(
	externalClient *client.ExternalAPIClient,
	memoryCache *cache.MemoryCache,
	redisCache *cache.RedisCache,
) *KLineService {
	return &KLineService{
		externalClient: externalClient,
		memoryCache:    memoryCache,
		redisCache:     redisCache,
		useRedis:       redisCache != nil,
	}
}

// GetKLineData 获取K线数据
func (s *KLineService) GetKLineData(ctx context.Context, req *models.KLineRequest) (*models.KLineData, bool, error) {
	cacheKey := s.buildCacheKey(req)
	
	// 1. 尝试从内存缓存获取
	if data, ok := s.memoryCache.Get(cacheKey); ok {
		if klineData, ok := data.(*models.KLineData); ok {
			return klineData, true, nil
		}
	}
	
	// 2. 尝试从Redis缓存获取
	if s.useRedis {
		var klineData models.KLineData
		if err := s.redisCache.Get(ctx, cacheKey, &klineData); err == nil {
			// 回写到内存缓存
			s.memoryCache.Set(cacheKey, &klineData)
			return &klineData, true, nil
		}
	}
	
	// 3. 从外部API获取
	klineData, err := s.externalClient.FetchKLine(ctx, req)
	if err != nil {
		return nil, false, fmt.Errorf("获取K线数据失败: %w", err)
	}
	
	// 4. 写入缓存
	s.memoryCache.Set(cacheKey, klineData)
	if s.useRedis {
		if err := s.redisCache.Set(ctx, cacheKey, klineData); err != nil {
			log.Printf("写入Redis缓存失败: %v", err)
		}
	}
	
	return klineData, false, nil
}

// GetKLineDataBatch 批量获取K线数据
func (s *KLineService) GetKLineDataBatch(ctx context.Context, reqs []*models.KLineRequest) ([]*models.KLineData, error) {
	results := make([]*models.KLineData, len(reqs))
	
	for i, req := range reqs {
		data, _, err := s.GetKLineData(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("获取第%d个K线数据失败: %w", i, err)
		}
		results[i] = data
	}
	
	return results, nil
}

// buildCacheKey 构建缓存key
func (s *KLineService) buildCacheKey(req *models.KLineRequest) string {
	return cache.BuildKLineKey(
		req.Market,
		req.Code,
		int(req.KLineType),
		int(req.Weight),
		req.Count,
	)
}

// InvalidateCache 使缓存失效
func (s *KLineService) InvalidateCache(ctx context.Context, req *models.KLineRequest) error {
	cacheKey := s.buildCacheKey(req)
	
	s.memoryCache.Delete(cacheKey)
	
	if s.useRedis {
		if err := s.redisCache.Delete(ctx, cacheKey); err != nil {
			return fmt.Errorf("删除Redis缓存失败: %w", err)
		}
	}
	
	return nil
}

// GetCacheStats 获取缓存统计
func (s *KLineService) GetCacheStats() (memoryHits, memoryMisses int64, memorySize int, hitRate float64) {
	return s.memoryCache.Stats()
}

// CacheService 缓存服务
type CacheService struct {
	memoryCache *cache.MemoryCache
	redisCache  *cache.RedisCache
	useRedis    bool
}

// NewCacheService 创建缓存服务
func NewCacheService(memoryCache *cache.MemoryCache, redisCache *cache.RedisCache) *CacheService {
	return &CacheService{
		memoryCache: memoryCache,
		redisCache:  redisCache,
		useRedis:    redisCache != nil,
	}
}

// GetIndicatorCache 获取指标缓存
func (s *CacheService) GetIndicatorCache(ctx context.Context, market int, code string, indicatorType string, params map[string]interface{}) (*models.IndicatorResult, bool) {
	paramsHash := s.hashParams(params)
	cacheKey := cache.BuildIndicatorKey(market, code, indicatorType, paramsHash)
	
	// 从内存缓存获取
	if data, ok := s.memoryCache.Get(cacheKey); ok {
		if result, ok := data.(*models.IndicatorResult); ok {
			return result, true
		}
	}
	
	// 从Redis获取
	if s.useRedis {
		var result models.IndicatorResult
		if err := s.redisCache.Get(ctx, cacheKey, &result); err == nil {
			s.memoryCache.Set(cacheKey, &result)
			return &result, true
		}
	}
	
	return nil, false
}

// SetIndicatorCache 设置指标缓存
func (s *CacheService) SetIndicatorCache(ctx context.Context, market int, code string, indicatorType string, params map[string]interface{}, result *models.IndicatorResult) {
	paramsHash := s.hashParams(params)
	cacheKey := cache.BuildIndicatorKey(market, code, indicatorType, paramsHash)
	
	s.memoryCache.Set(cacheKey, result)
	
	if s.useRedis {
		if err := s.redisCache.Set(ctx, cacheKey, result); err != nil {
			log.Printf("写入指标Redis缓存失败: %v", err)
		}
	}
}

// hashParams 计算参数哈希
func (s *CacheService) hashParams(params map[string]interface{}) string {
	if params == nil {
		return "default"
	}
	data, _ := json.Marshal(params)
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:8])
}

// Health 健康检查
func (s *CacheService) Health(ctx context.Context) map[string]string {
	status := make(map[string]string)
	status["memory_cache"] = "up"
	
	if s.useRedis {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := s.redisCache.Health(ctx); err != nil {
			status["redis_cache"] = "down"
		} else {
			status["redis_cache"] = "up"
		}
	}
	
	return status
}
