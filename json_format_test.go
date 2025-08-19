package apic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestJSONFormatting 测试JSON格式化功能
func TestJSONFormatting(t *testing.T) {
	// 创建测试服务器，返回JSON响应
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"id":   123,
				"name": "Test User",
				"tags": []string{"tag1", "tag2"},
			},
			"message": "Operation completed successfully",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// 创建客户端并添加调试中间件
	client := NewStdlibClient()
	client.AddMiddleware(NewDebugMiddleware())

	// 发送JSON请求
	jsonData := map[string]interface{}{
		"username": "testuser",
		"email":    "test@example.com",
		"profile": map[string]interface{}{
			"age":     25,
			"country": "US",
		},
	}

	request := NewJSONData(jsonData)
	resp, err := client.POST(context.Background(), server.URL, request, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 验证Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

// TestNonJSONFormatting 测试非JSON内容不被格式化
func TestNonJSONFormatting(t *testing.T) {
	// 创建测试服务器，返回纯文本响应
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("This is plain text response"))
	}))
	defer server.Close()

	// 创建客户端并添加调试中间件
	client := NewStdlibClient()
	client.AddMiddleware(NewDebugMiddleware())

	// 发送请求
	resp, err := client.GET(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 验证Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", contentType)
	}
}