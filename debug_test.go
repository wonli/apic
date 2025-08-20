package apic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// 测试调试功能
func TestDebugFeature(t *testing.T) {
	// 简单测试调试模式的启用
	client := NewStdlibClient()

	// 测试调试模式设置
	if client.debug {
		t.Error("Debug should be false by default")
	}

	client.SetDebug(true)
	if !client.debug {
		t.Error("Debug should be true after SetDebug(true)")
	}

	client.SetDebug(false)
	if client.debug {
		t.Error("Debug should be false after SetDebug(false)")
	}
}

// 测试标准库客户端的调试功能
func TestStdlibClientDebug(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"debug": true}`))
	}))
	defer server.Close()

	// 创建标准库客户端并启用调试
	client := NewStdlibClient().SetDebug(true)

	// 设置请求头
	client.SetHeaders(map[string]string{
		"User-Agent":    "debug-test-client",
		"Authorization": "Bearer debug-token",
	})

	// 创建测试数据
	jsonData := NewJSONData(map[string]any{
		"test":      "debug",
		"timestamp": time.Now().Unix(),
	})

	// 执行POST请求
	ctx := context.Background()
	resp, err := client.POST(ctx, server.URL+"/debug", jsonData, nil)
	if err != nil {
		t.Errorf("Debug POST request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 执行GET请求
	queryParams := url.Values{}
	queryParams.Set("debug", "true")
	queryParams.Set("format", "json")

	resp2, err := client.GET(ctx, server.URL+"/debug", queryParams)
	if err != nil {
		t.Errorf("Debug GET request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp2.StatusCode)
	}
}

// 测试调试模式下的错误处理
func TestDebugErrorHandling(t *testing.T) {
	// 创建返回错误的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request", "code": 400}`))
	}))
	defer server.Close()

	// 创建标准库客户端并启用调试
	client := NewStdlibClient().SetDebug(true)

	// 执行会失败的请求
	ctx := context.Background()
	resp, err := client.GET(ctx, server.URL+"/error", nil)
	if err != nil {
		t.Errorf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// 验证错误状态码被正确记录
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

// TestDebugApi 用于测试的API实现
type TestDebugApi struct {
	*Apic
	testURL   string
	debugMode bool
}

func (t *TestDebugApi) Url() string {
	return t.testURL
}

func (t *TestDebugApi) Path() string {
	return "/debug-test"
}

func (t *TestDebugApi) Query() url.Values {
	q := url.Values{}
	q.Set("debug", "true")
	q.Set("test", "api")
	return q
}

func (t *TestDebugApi) Headers() Params {
	return Params{
		"X-Debug-Test":  "true",
		"X-API-Version": "1.0",
	}
}

func (t *TestDebugApi) PostBody() any {
	return map[string]any{
		"debug":     true,
		"test_data": "debug api test",
		"timestamp": time.Now().Unix(),
	}
}

func (t *TestDebugApi) HttpMethod() HttpMethod {
	return POST
}

func (t *TestDebugApi) Debug() bool {
	return t.debugMode
}

func (t *TestDebugApi) OnResponse(resp []byte) (*ResponseData, error) {
	return &ResponseData{Data: resp}, nil
}

func TestSimpleHTTPRequest(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	// 创建标准HTTP客户端
	client := &http.Client{}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Standard HTTP client - Status: %d", resp.StatusCode)
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 测试StdlibClient
	stdlibClient := NewStdlibClient()
	stdlibResp, err := stdlibClient.GET(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("StdlibClient request failed: %v", err)
	}
	defer stdlibResp.Body.Close()

	t.Logf("StdlibClient - Status: %d", stdlibResp.StatusCode)
	if stdlibResp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", stdlibResp.StatusCode)
	}

	// 测试Reset后的StdlibClient
	stdlibClient.Reset()
	stdlibResp2, err := stdlibClient.GET(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("Reset StdlibClient request failed: %v", err)
	}
	defer stdlibResp2.Body.Close()

	t.Logf("Reset StdlibClient - Status: %d", stdlibResp2.StatusCode)
	if stdlibResp2.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", stdlibResp2.StatusCode)
	}
}

// TestFormatJSONLimit 测试 JSON 长度限制功能
func TestFormatJSONLimit(t *testing.T) {
	// 测试短 JSON
	shortJSON := []byte(`{"name":"test","value":123}`)
	result := formatJSON(shortJSON)
	if !strings.Contains(result, "name") {
		t.Error("Short JSON should be formatted properly")
	}

	// 测试长 JSON (超过 1000 字符)
	longData := strings.Repeat("a", 1200)
	longJSON := []byte(`{"data":"` + longData + `"}`)
	result = formatJSON(longJSON)

	// 验证长JSON不被格式化，长度应该等于原始长度
	if len(result) != len(longJSON) {
		t.Errorf("Long JSON should not be formatted, expected length %d, got %d", len(longJSON), len(result))
	}

	// 验证内容就是原始JSON
	if result != string(longJSON) {
		t.Error("Long JSON should return original data without formatting")
	}

	t.Logf("Long JSON original length: %d, formatted length: %d", len(longJSON), len(result))
}
