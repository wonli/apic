package apic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestRetryMiddleware_Success 测试成功请求不重试
func TestRetryMiddleware_Success(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	// 创建客户端并添加重试中间件
	client := NewStdlibClient()
	client.AddMiddleware(NewRetryMiddleware(DefaultRetryConfig()))

	// 执行请求
	resp, err := client.GET(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestRetryMiddleware_RetryOnServerError 测试服务器错误时重试
func TestRetryMiddleware_RetryOnServerError(t *testing.T) {
	attempts := 0

	// 创建测试服务器，前2次返回500，第3次返回200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		t.Logf("Server attempt %d", attempts)
		if attempts <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}
	}))
	defer server.Close()

	// 创建重试配置（MaxRetries=2，总共最多3次尝试）
	config := &RetryConfig{
		MaxRetries:           2,
		InitialDelay:         1 * time.Millisecond,
		MaxDelay:             10 * time.Millisecond,
		BackoffMultiplier:    2.0,
		RetryableStatusCodes: []int{500},
	}

	// 创建客户端并添加重试中间件
	client := NewStdlibClient()
	client.AddMiddleware(NewRetryMiddleware(config))

	// 执行请求
	resp, err := client.GET(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 应该总共尝试3次：初始请求 + 2次重试
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestRetryMiddleware_MaxRetriesExceeded 测试超过最大重试次数
func TestRetryMiddleware_MaxRetriesExceeded(t *testing.T) {
	attempts := 0

	// 创建测试服务器（总是返回错误）
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	// 创建重试配置
	config := &RetryConfig{
		MaxRetries:           2,
		InitialDelay:         10 * time.Millisecond,
		MaxDelay:             100 * time.Millisecond,
		BackoffMultiplier:    2.0,
		RetryableStatusCodes: []int{500},
	}

	// 创建客户端并添加重试中间件
	client := NewStdlibClient()
	client.AddMiddleware(NewRetryMiddleware(config))

	// 执行请求
	resp, err := client.GET(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	// 应该尝试 1 + 2 = 3 次（初始请求 + 2次重试）
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestRetryMiddleware_WithOptions 测试选项模式
func TestRetryMiddleware_WithOptions(t *testing.T) {
	attempts := 0
	retryCallbacks := 0

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusBadGateway)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// 创建客户端并添加重试中间件（使用选项模式）
	client := NewStdlibClient()
	client.AddMiddleware(NewRetryMiddlewareWithOptions(
		WithMaxRetries(3),
		WithInitialDelay(5*time.Millisecond),
		WithRetryableStatusCodes(502),
		WithOnRetry(func(attempt int, err error, delay time.Duration) {
			retryCallbacks++
			t.Logf("Retry attempt %d, delay: %v", attempt, delay)
		}),
	))

	// 执行请求
	resp, err := client.GET(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}

	if retryCallbacks != 1 {
		t.Errorf("Expected 1 retry callback, got %d", retryCallbacks)
	}
}

// TestRetryMiddleware_CustomRetryCondition 测试自定义重试条件
func TestRetryMiddleware_CustomRetryCondition(t *testing.T) {
	attempts := 0

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusNotFound) // 404通常不重试
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// 创建客户端并添加重试中间件（自定义重试条件）
	client := NewStdlibClient()
	client.AddMiddleware(NewRetryMiddlewareWithOptions(
		WithMaxRetries(3),
		WithInitialDelay(5*time.Millisecond),
		WithRetryCondition(func(resp *http.Response, err error) bool {
			// 自定义条件：404也重试
			return err != nil || (resp != nil && resp.StatusCode == http.StatusNotFound)
		}),
	))

	// 执行请求
	resp, err := client.GET(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestRetryMiddleware_WithPOSTData 测试POST请求重试
func TestRetryMiddleware_WithPOSTData(t *testing.T) {
	attempts := 0
	receivedBodies := []string{}

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++

		// 读取请求体
		body, _ := io.ReadAll(r.Body)
		receivedBodies = append(receivedBodies, string(body))

		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// 创建客户端并添加重试中间件
	client := NewStdlibClient()
	client.AddMiddleware(NewRetryMiddlewareWithOptions(
		WithMaxRetries(2),
		WithInitialDelay(5*time.Millisecond),
		WithRetryableStatusCodes(500),
	))

	// 创建POST数据
	data := NewJSONData(map[string]any{
		"message": "test data",
		"id":      123,
	})

	// 执行POST请求
	resp, err := client.POST(context.Background(), server.URL, data, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}

	// 验证每次重试都发送了相同的请求体
	if len(receivedBodies) != 2 {
		t.Errorf("Expected 2 received bodies, got %d", len(receivedBodies))
	}

	for i, body := range receivedBodies {
		if !strings.Contains(body, "test data") {
			t.Errorf("Attempt %d: expected body to contain 'test data', got: %s", i+1, body)
		}
	}
}

// TestCalculateDelay 测试延迟计算
func TestCalculateDelay(t *testing.T) {
	config := &RetryConfig{
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          5 * time.Second,
		BackoffMultiplier: 2.0,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{10, 5 * time.Second}, // 应该被限制在MaxDelay
	}

	for _, test := range tests {
		actual := calculateDelay(config, test.attempt)
		if actual != test.expected {
			t.Errorf("Attempt %d: expected delay %v, got %v", test.attempt, test.expected, actual)
		}
	}
}

// TestShouldRetry 测试重试条件判断
func TestShouldRetry(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 3

	tests := []struct {
		name     string
		resp     *http.Response
		err      error
		attempt  int
		expected bool
	}{
		{
			name:     "network error",
			resp:     nil,
			err:      fmt.Errorf("network error"),
			attempt:  0,
			expected: true,
		},
		{
			name:     "500 error",
			resp:     &http.Response{StatusCode: 500},
			err:      nil,
			attempt:  0,
			expected: true,
		},
		{
			name:     "200 success",
			resp:     &http.Response{StatusCode: 200},
			err:      nil,
			attempt:  0,
			expected: false,
		},
		{
			name:     "max retries exceeded",
			resp:     &http.Response{StatusCode: 500},
			err:      nil,
			attempt:  3,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := shouldRetry(test.resp, test.err, config, test.attempt)
			if actual != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, actual)
			}
		})
	}
}
