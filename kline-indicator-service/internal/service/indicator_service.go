package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kline-indicator-service/internal/calculator"
	_ "kline-indicator-service/internal/calculator/moshi" // 注册莫氏缠论指标
	"kline-indicator-service/internal/models"
)

// IndicatorService 指标计算服务
type IndicatorService struct {
	klineService *KLineService
	cacheService *CacheService
	registry     *calculator.Registry
}

// NewIndicatorService 创建指标服务
func NewIndicatorService(klineService *KLineService, cacheService *CacheService) *IndicatorService {
	return &IndicatorService{
		klineService: klineService,
		cacheService: cacheService,
		registry:     calculator.GetRegistry(),
	}
}

// Calculate 计算指标
func (s *IndicatorService) Calculate(ctx context.Context, req *models.IndicatorCalculateRequest) (*models.IndicatorCalculateResponse, error) {
	startTime := time.Now()
	
	// 获取K线数据
	klineReq := &models.KLineRequest{
		Market:    req.Market,
		Code:      req.Code,
		KLineType: req.KLineType,
		Weight:    req.Weight,
		Count:     req.Count,
	}
	
	klineData, _, err := s.klineService.GetKLineData(ctx, klineReq)
	if err != nil {
		return nil, fmt.Errorf("获取K线数据失败: %w", err)
	}
	
	// 并发计算多个指标
	results := make([]models.IndicatorResult, len(req.Indicators))
	var wg sync.WaitGroup
	errChan := make(chan error, len(req.Indicators))
	
	for i, config := range req.Indicators {
		wg.Add(1)
		go func(idx int, cfg models.IndicatorConfig) {
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("计算指标%s时panic: %v", cfg.Type, r)
				}
				wg.Done()
			}()
			
			result, err := s.calculateSingle(ctx, req.Market, req.Code, klineData.KLines, cfg)
			if err != nil {
				errChan <- fmt.Errorf("计算指标%s失败: %w", cfg.Type, err)
				return
			}
			results[idx] = *result
		}(i, config)
	}
	
	wg.Wait()
	close(errChan)
	
	// 检查错误
	for err := range errChan {
		return nil, err
	}
	
	return &models.IndicatorCalculateResponse{
		Code:            models.ErrCodeSuccess,
		Message:         "success",
		StockCode:       req.Code,
		Indicators:      results,
		ComputationTime: time.Since(startTime).Milliseconds(),
	}, nil
}

// calculateSingle 计算单个指标
func (s *IndicatorService) calculateSingle(ctx context.Context, market int, code string, klines []models.KLine, config models.IndicatorConfig) (*models.IndicatorResult, error) {
	// 尝试从缓存获取
	if cached, ok := s.cacheService.GetIndicatorCache(ctx, market, code, string(config.Type), config.Params); ok {
		return cached, nil
	}
	
	// 创建指标实例
	indicator, err := s.registry.Create(config.Type)
	if err != nil {
		return nil, err
	}
	
	// 验证参数
	if err := indicator.Validate(config.Params); err != nil {
		return nil, err
	}
	
	// 计算指标
	result, err := indicator.Calculate(klines, config.Params)
	if err != nil {
		return nil, err
	}
	
	// 缓存结果
	s.cacheService.SetIndicatorCache(ctx, market, code, string(config.Type), config.Params, result)
	
	return result, nil
}

// ListIndicators 列出所有指标
func (s *IndicatorService) ListIndicators() []models.IndicatorMetadata {
	return s.registry.List()
}

// GetIndicatorMetadata 获取指标元信息
func (s *IndicatorService) GetIndicatorMetadata(indicatorType models.IndicatorType) (models.IndicatorMetadata, bool) {
	return s.registry.GetMetadata(indicatorType)
}
