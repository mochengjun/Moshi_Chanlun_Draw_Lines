package calculator

import (
	"fmt"
	"sync"

	"kline-indicator-service/internal/models"
)

// Indicator 指标计算接口
type Indicator interface {
	// Metadata 获取指标元信息
	Metadata() models.IndicatorMetadata
	
	// Validate 验证参数
	Validate(params map[string]interface{}) error
	
	// Calculate 计算指标
	Calculate(klines []models.KLine, params map[string]interface{}) (*models.IndicatorResult, error)
}

// IndicatorFactory 指标工厂函数
type IndicatorFactory func() Indicator

// Registry 指标注册中心
type Registry struct {
	mu         sync.RWMutex
	indicators map[models.IndicatorType]IndicatorFactory
	metadata   map[models.IndicatorType]models.IndicatorMetadata
}

// globalRegistry 全局注册中心
var globalRegistry = &Registry{
	indicators: make(map[models.IndicatorType]IndicatorFactory),
	metadata:   make(map[models.IndicatorType]models.IndicatorMetadata),
}

// GetRegistry 获取全局注册中心
func GetRegistry() *Registry {
	return globalRegistry
}

// Register 注册指标
func (r *Registry) Register(indicatorType models.IndicatorType, factory IndicatorFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.indicators[indicatorType] = factory
	
	// 缓存元信息
	indicator := factory()
	r.metadata[indicatorType] = indicator.Metadata()
}

// Create 创建指标实例
func (r *Registry) Create(indicatorType models.IndicatorType) (Indicator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	factory, ok := r.indicators[indicatorType]
	if !ok {
		return nil, fmt.Errorf("未注册的指标类型: %s", indicatorType)
	}
	
	return factory(), nil
}

// List 列出所有指标
func (r *Registry) List() []models.IndicatorMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make([]models.IndicatorMetadata, 0, len(r.metadata))
	for _, meta := range r.metadata {
		result = append(result, meta)
	}
	
	return result
}

// GetMetadata 获取指标元信息
func (r *Registry) GetMetadata(indicatorType models.IndicatorType) (models.IndicatorMetadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	meta, ok := r.metadata[indicatorType]
	return meta, ok
}

// Has 检查指标是否已注册
func (r *Registry) Has(indicatorType models.IndicatorType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	_, ok := r.indicators[indicatorType]
	return ok
}

// Register 全局注册函数
func Register(indicatorType models.IndicatorType, factory IndicatorFactory) {
	globalRegistry.Register(indicatorType, factory)
}
