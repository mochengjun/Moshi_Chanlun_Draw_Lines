package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache Redis缓存
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

// NewRedisCache 创建Redis缓存
func NewRedisCache(cfg RedisConfig, ttl time.Duration) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
	
	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接Redis失败: %w", err)
	}
	
	return &RedisCache{
		client: client,
		ttl:    ttl,
	}, nil
}

// Get 获取缓存
func (rc *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := rc.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, dest)
}

// Set 设置缓存
func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}) error {
	return rc.SetWithTTL(ctx, key, value, rc.ttl)
}

// SetWithTTL 设置带TTL的缓存
func (rc *RedisCache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %w", err)
	}
	
	return rc.client.Set(ctx, key, data, ttl).Err()
}

// Delete 删除缓存
func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	return rc.client.Del(ctx, key).Err()
}

// Exists 检查key是否存在
func (rc *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := rc.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Health 检查Redis健康状态
func (rc *RedisCache) Health(ctx context.Context) error {
	return rc.client.Ping(ctx).Err()
}

// Close 关闭连接
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

// GetClient 获取Redis客户端
func (rc *RedisCache) GetClient() *redis.Client {
	return rc.client
}

// BuildKLineKey 构建K线缓存key
func BuildKLineKey(market int, code string, klineType, weight, count int) string {
	return fmt.Sprintf("kline:%d:%s:%d:%d:%d", market, code, klineType, weight, count)
}

// BuildIndicatorKey 构建指标缓存key
func BuildIndicatorKey(market int, code string, indicatorType string, paramsHash string) string {
	return fmt.Sprintf("indicator:%d:%s:%s:%s", market, code, indicatorType, paramsHash)
}
