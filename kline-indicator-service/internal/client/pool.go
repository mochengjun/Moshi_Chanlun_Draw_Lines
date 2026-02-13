package client

import (
	"net/http"
	"sync"
	"time"
)

// ConnectionPool HTTP连接池管理
type ConnectionPool struct {
	transport *http.Transport
	mu        sync.RWMutex
	stats     PoolStats
}

// PoolStats 连接池统计
type PoolStats struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	ActiveConns     int
}

// NewConnectionPool 创建连接池
func NewConnectionPool(maxIdleConns, maxConnsPerHost int) *ConnectionPool {
	return &ConnectionPool{
		transport: &http.Transport{
			MaxIdleConns:        maxIdleConns,
			MaxIdleConnsPerHost: maxConnsPerHost,
			MaxConnsPerHost:     maxConnsPerHost,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
			DisableKeepAlives:   false,
		},
	}
}

// GetTransport 获取Transport
func (p *ConnectionPool) GetTransport() *http.Transport {
	return p.transport
}

// IncrementSuccess 增加成功计数
func (p *ConnectionPool) IncrementSuccess() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.TotalRequests++
	p.stats.SuccessRequests++
}

// IncrementFailed 增加失败计数
func (p *ConnectionPool) IncrementFailed() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.TotalRequests++
	p.stats.FailedRequests++
}

// GetStats 获取统计信息
func (p *ConnectionPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// Close 关闭连接池
func (p *ConnectionPool) Close() {
	p.transport.CloseIdleConnections()
}
