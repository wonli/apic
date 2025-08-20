package apic

import (
	"sync"
	"sync/atomic"
)

// pooledClient 包装 StdlibClient 并管理使用状态
type pooledClient struct {
	client *StdlibClient
	inUse  bool
}

// ClientFactory 客户端工厂，实现对象池和客户端复用
type ClientFactory struct {
	pool        sync.Pool
	maxPoolSize int
	currentSize int64
}

// NewClientFactory 创建新的客户端工厂
func NewClientFactory(maxPoolSize int) *ClientFactory {
	if maxPoolSize <= 0 {
		maxPoolSize = 10 // 默认最大池大小
	}

	return &ClientFactory{
		maxPoolSize: maxPoolSize,
		pool: sync.Pool{
			New: func() any {
				return &pooledClient{
					client: NewStdlibClient(),
					inUse:  false,
				}
			},
		},
	}
}

// AcquireClient 从池中获取一个客户端
func (cf *ClientFactory) AcquireClient() *StdlibClient {
	pc := cf.pool.Get().(*pooledClient)
	pc.inUse = true

	// 重置客户端状态
	pc.client.Reset()

	// 更新当前使用数量
	atomic.AddInt64(&cf.currentSize, 1)

	return pc.client
}

// ReleaseClient 将客户端释放回池中
func (cf *ClientFactory) ReleaseClient(client *StdlibClient) {
	if client == nil {
		return
	}

	// 重置客户端状态，清理可能的污染
	client.Reset()

	// 创建 pooledClient 并放回池中
	pc := &pooledClient{
		client: client,
		inUse:  false,
	}

	// 更新当前使用数量
	atomic.AddInt64(&cf.currentSize, -1)

	cf.pool.Put(pc)
}

// GetCurrentSize 获取当前正在使用的客户端数量
func (cf *ClientFactory) GetCurrentSize() int64 {
	return atomic.LoadInt64(&cf.currentSize)
}

// GetMaxPoolSize 获取最大池大小
func (cf *ClientFactory) GetMaxPoolSize() int {
	return cf.maxPoolSize
}

// Stats 获取客户端工厂的统计信息
type FactoryStats struct {
	CurrentSize int64 `json:"current_size"`
	MaxPoolSize int   `json:"max_pool_size"`
}

// GetStats 获取客户端工厂统计信息
func (cf *ClientFactory) GetStats() *FactoryStats {
	return &FactoryStats{
		CurrentSize: cf.GetCurrentSize(),
		MaxPoolSize: cf.GetMaxPoolSize(),
	}
}
