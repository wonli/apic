package apic

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// SimpleTestApi 简单的测试API实现
type SimpleTestApi struct {
	*Apic
	url string
}

func (s *SimpleTestApi) Url() string {
	return s.url
}

func (s *SimpleTestApi) Path() string {
	return "" // 返回空路径，因为URL已经是完整的
}

func (s *SimpleTestApi) HttpMethod() HttpMethod {
	return GET
}

func (s *SimpleTestApi) OnResponse(resp []byte) (*ResponseData, error) {
	return &ResponseData{Data: resp}, nil
}

func (s *SimpleTestApi) Setup() (Api, error) {
	return s, nil
}

func (s *SimpleTestApi) OnHttpStatusError(statusCode int, data []byte) error {
	// 记录状态码错误信息
	fmt.Printf("OnHttpStatusError called with status: %d, data: %s\n", statusCode, string(data))
	// 对于测试，我们不将非200状态码视为错误
	return nil
}

func TestClientPool_Integration(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	// 创建API定义
	api := &SimpleTestApi{
		Apic: &Apic{},
		url:  server.URL,
	}

	// 创建ApiId
	apiId := &ApiId{
		Name:   "test-api",
		Client: api,
		Request: &RequestData{
			HttpMethod: GET,
			Url:        server.URL,
			Path:       "", // 空路径，因为server.URL已经是完整URL
		},
	}

	// 初始化ApiClients
	apiClients := Init()

	// 调试：检查客户端池状态
	stats := apiClients.GetClientFactoryStats()
	t.Logf("Initial pool stats: Current=%d, Max=%d", stats.CurrentSize, stats.MaxPoolSize)

	// 执行多次调用以测试客户端复用
	for i := 0; i < 5; i++ {
		t.Logf("Making API call %d to URL: %s%s", i, apiId.Request.Url, apiId.Request.Path)
		
		// 调试：检查客户端池状态
		stats = apiClients.GetClientFactoryStats()
		t.Logf("Before call %d - pool stats: Current=%d, Max=%d", i, stats.CurrentSize, stats.MaxPoolSize)
		
		response, err := apiClients.CallApi(apiId, nil)
		if err != nil {
			t.Fatalf("API call %d failed: %v", i, err)
		}

		t.Logf("Response status: %d, data length: %d, data: %s", response.HttpStatus, len(response.Data), string(response.Data))
		if response.HttpStatus != 200 {
			t.Errorf("Expected status 200, got %d", response.HttpStatus)
		}

		if len(response.Data) == 0 {
			t.Error("Expected response data, got empty")
		}
	}

	// 验证客户端池统计信息
	stats = apiClients.GetClientFactoryStats()
	if stats == nil {
		t.Error("Expected client factory stats, got nil")
	}

	t.Logf("Client pool stats: Current=%d, Max=%d",
		stats.CurrentSize, stats.MaxPoolSize)
}

func TestClientPool_ConcurrentRequests(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟一些处理时间
		time.Sleep(10 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"message": "success"}`)
	}))
	defer server.Close()

	// 初始化API客户端，设置较小的池大小
	apiClients := Init().SetClientFactoryMaxSize(5)

	// 创建API定义
	api := &SimpleTestApi{
		Apic: &Apic{},
		url:  server.URL,
	}

	apiId := &ApiId{
		Name:   "concurrent-test-api",
		Client: api,
		Request: &RequestData{
			HttpMethod: GET,
			Url:        server.URL,
			Path:       "/",
		},
	}

	// 并发执行多个请求
	const numRequests = 20
	var wg sync.WaitGroup
	results := make(chan string, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()
			response, err := apiClients.CallApi(apiId, nil)
			if err != nil {
				results <- fmt.Sprintf("request %d failed: %v", requestID, err)
				return
			}
			results <- fmt.Sprintf("request %d got status %d", requestID, response.HttpStatus)
		}(i)
	}

	wg.Wait()
	close(results)

	// 检查结果
	successCount := 0
	for result := range results {
		t.Log(result)
		if fmt.Sprintf("request %d got status 200", successCount) == result {
			successCount++
		}
	}

	// 验证客户端池统计信息
	stats := apiClients.GetClientFactoryStats()
	t.Logf("Final pool stats: Current=%d, Max=%d",
		stats.CurrentSize, stats.MaxPoolSize)

	// 验证池大小没有超过限制
	if stats.CurrentSize > int64(stats.MaxPoolSize) {
		t.Errorf("Pool size exceeded limit: current=%d, max=%d", stats.CurrentSize, stats.MaxPoolSize)
	}
}

func TestClientPool_WithMiddleware(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	// 创建API定义
	api := &SimpleTestApi{
		Apic: &Apic{},
		url:  server.URL,
	}

	apiId := &ApiId{
		Name:   "middleware-test-api",
		Client: api,
		Request: &RequestData{
			HttpMethod: GET,
			Url:        server.URL,
			Path:       "/",
			Debug:      true,
		},
	}

	// 初始化ApiClients并添加中间件
	middlewareCalled := false
	apiClients := Init().Use(MiddlewareFunc(func(ctx *Context) {
		middlewareCalled = true
		ctx.Next()
	}))

	// 执行API调用
	response, err := apiClients.CallApi(apiId, nil)
	if err != nil {
		t.Fatalf("API call failed: %v", err)
	}

	if response.HttpStatus != 200 {
		t.Errorf("Expected status 200, got %d", response.HttpStatus)
	}

	if !middlewareCalled {
		t.Error("Expected middleware to be called")
	}

	// 验证客户端池统计信息
	stats := apiClients.GetClientFactoryStats()
	t.Logf("Middleware test pool stats: Current=%d, Max=%d",
		stats.CurrentSize, stats.MaxPoolSize)
}