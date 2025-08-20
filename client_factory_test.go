package apic

import (
	"sync"
	"testing"
	"time"
)

func TestClientFactory_AcquireRelease(t *testing.T) {
	factory := NewClientFactory(5)
	
	// 测试获取客户端
	client := factory.AcquireClient()
	if client == nil {
		t.Fatal("AcquireClient returned nil")
	}
	
	// 验证当前使用数量
	if factory.GetCurrentSize() != 1 {
		t.Errorf("Expected current size 1, got %d", factory.GetCurrentSize())
	}
	
	// 测试释放客户端
	factory.ReleaseClient(client)
	if factory.GetCurrentSize() != 0 {
		t.Errorf("Expected current size 0 after release, got %d", factory.GetCurrentSize())
	}
}

func TestClientFactory_Reset(t *testing.T) {
	factory := NewClientFactory(5)
	client := factory.AcquireClient()
	
	// 配置客户端
	client.SetProxy("http://proxy.example.com:8080")
	client.SetDebug(true)
	client.SetHeader("X-Test", "value")
	client.AddMiddleware(NewDebugMiddleware())
	
	// 释放并重新获取
	factory.ReleaseClient(client)
	client2 := factory.AcquireClient()
	
	// 验证状态已重置
	if client2.proxy != "" {
		t.Error("Proxy should be reset")
	}
	if client2.debug {
		t.Error("Debug should be reset")
	}
	if len(client2.headers) != 0 {
		t.Error("Headers should be reset")
	}
	if len(client2.middlewares) != 0 {
		t.Error("Middlewares should be reset")
	}
	
	factory.ReleaseClient(client2)
}

func TestClientFactory_Concurrent(t *testing.T) {
	factory := NewClientFactory(10)
	var wg sync.WaitGroup
	concurrency := 50
	
	// 并发获取和释放客户端
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			client := factory.AcquireClient()
			if client == nil {
				t.Error("AcquireClient returned nil")
				return
			}
			
			// 模拟一些工作
			client.SetHeader("X-Worker", "test")
			time.Sleep(10 * time.Millisecond)
			
			factory.ReleaseClient(client)
		}()
	}
	
	wg.Wait()
	
	// 验证所有客户端都已释放
	if factory.GetCurrentSize() != 0 {
		t.Errorf("Expected current size 0 after all releases, got %d", factory.GetCurrentSize())
	}
}

func TestClientFactory_Stats(t *testing.T) {
	factory := NewClientFactory(15)
	
	stats := factory.GetStats()
	if stats.MaxPoolSize != 15 {
		t.Errorf("Expected max pool size 15, got %d", stats.MaxPoolSize)
	}
	if stats.CurrentSize != 0 {
		t.Errorf("Expected current size 0, got %d", stats.CurrentSize)
	}
	
	// 获取一些客户端
	clients := make([]*StdlibClient, 3)
	for i := 0; i < 3; i++ {
		clients[i] = factory.AcquireClient()
	}
	
	stats = factory.GetStats()
	if stats.CurrentSize != 3 {
		t.Errorf("Expected current size 3, got %d", stats.CurrentSize)
	}
	
	// 释放客户端
	for _, client := range clients {
		factory.ReleaseClient(client)
	}
	
	stats = factory.GetStats()
	if stats.CurrentSize != 0 {
		t.Errorf("Expected current size 0 after release, got %d", stats.CurrentSize)
	}
}

func TestClientFactory_ReleaseNil(t *testing.T) {
	factory := NewClientFactory(5)
	
	// 测试释放 nil 客户端不会崩溃
	factory.ReleaseClient(nil)
	
	if factory.GetCurrentSize() != 0 {
		t.Errorf("Expected current size 0, got %d", factory.GetCurrentSize())
	}
}

func BenchmarkClientFactory_AcquireRelease(b *testing.B) {
	factory := NewClientFactory(10)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client := factory.AcquireClient()
			client.SetHeader("X-Bench", "test")
			factory.ReleaseClient(client)
		}
	})
}

func BenchmarkNewStdlibClient(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client := NewStdlibClient()
			client.SetHeader("X-Bench", "test")
			// 模拟使用后的清理
			_ = client
		}
	})
}