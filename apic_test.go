package apic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// 测试新的SetData接口
func TestSetDataInterface(t *testing.T) {
	api := &Apic{}

	// 测试JSON数据
	jsonData := map[string]any{
		"name": "test",
		"age":  25,
	}
	err := api.SetData(NewJSONData(jsonData))
	if err != nil {
		t.Errorf("SetData with JSON failed: %v", err)
	}

	// 测试Form数据
	formData := map[string]string{
		"username": "testuser",
		"password": "testpass",
	}
	err = api.SetData(NewFormData(formData))
	if err != nil {
		t.Errorf("SetData with Form failed: %v", err)
	}

	// 测试WWWForm数据
	wwwFormData := url.Values{}
	wwwFormData.Set("key1", "value1")
	wwwFormData.Set("key2", "value2")
	err = api.SetData(NewWWWFormData(wwwFormData))
	if err != nil {
		t.Errorf("SetData with WWWForm failed: %v", err)
	}

	// 测试Query数据
	queryData := url.Values{}
	queryData.Set("page", "1")
	queryData.Set("size", "10")
	err = api.SetData(NewQueryData(queryData))
	if err != nil {
		t.Errorf("SetData with Query failed: %v", err)
	}

	// 测试Header数据
	headerData := map[string]string{
		"Authorization": "Bearer token123",
		"Content-Type":  "application/json",
	}
	err = api.SetData(NewHeaderData(headerData))
	if err != nil {
		t.Errorf("SetData with Header failed: %v", err)
	}
}

// 测试向后兼容的方法
func TestBackwardCompatibility(t *testing.T) {
	api := &Apic{}

	// 测试SetJSON
	jsonData := map[string]any{
		"name": "test",
		"age":  25,
	}
	result := api.SetJSON(jsonData)
	if result == nil {
		t.Error("SetJSON should return Api interface")
	}

	// 测试SetForm
	formData := map[string]string{
		"username": "testuser",
		"password": "testpass",
	}
	result = api.SetForm(formData)
	if result == nil {
		t.Error("SetForm should return Api interface")
	}

	// 测试SetWWWForm
	wwwFormData := url.Values{}
	wwwFormData.Set("key1", "value1")
	result = api.SetWWWForm(wwwFormData)
	if result == nil {
		t.Error("SetWWWForm should return Api interface")
	}

	// 测试SetQuery
	queryData := url.Values{}
	queryData.Set("page", "1")
	result = api.SetQuery(queryData)
	if result == nil {
		t.Error("SetQuery should return Api interface")
	}

	// 测试SetHeader
	headerData := map[string]string{
		"Authorization": "Bearer token123",
	}
	result = api.SetHeader(headerData)
	if result == nil {
		t.Error("SetHeader should return Api interface")
	}

	// 测试链式调用
	result = api.SetJSON(jsonData).SetHeader(headerData).SetQuery(queryData)
	if result == nil {
		t.Error("Chained calls should work")
	}
}

// 测试代理设置
func TestProxySettings(t *testing.T) {
	client := Init().WithProxy("http://proxy.example.com:8080")
	if client.proxy != "http://proxy.example.com:8080" {
		t.Error("Proxy should be set correctly")
	}
}

// 测试上下文
func TestContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := Init().WithContext(ctx)
	if client.ctx != ctx {
		t.Error("Context should be set correctly")
	}
}

// 测试数据类型转换函数
func TestDataConversion(t *testing.T) {
	// 测试convertToStringMap
	testData := map[string]any{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}
	result := convertToStringMap(testData)
	if result["key1"] != "value1" {
		t.Error("String conversion failed")
	}
	if result["key2"] != "123" {
		t.Error("Integer conversion failed")
	}
	if result["key3"] != "true" {
		t.Error("Boolean conversion failed")
	}

	// 测试convertToURLValues
	urlResult := convertToURLValues(testData)
	if urlResult.Get("key1") != "value1" {
		t.Error("URL Values conversion failed")
	}
}

// 测试HTTP客户端基本功能
func TestHttpClientBasic(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success", "method": "` + r.Method + `"}`))
	}))
	defer server.Close()

	// 测试标准库客户端
	client := NewStdlibClient()
	if client == nil {
		t.Error("StdlibClient should be created successfully")
	}

	// 测试设置代理
	client.SetProxy("http://proxy.example.com:8080")

	// 测试设置调试模式
	client.SetDebug(true)

	// 测试设置请求头
	headers := map[string]string{
		"User-Agent": "test-client",
	}
	client.SetHeaders(headers)
}

// 基准测试
func BenchmarkSetDataJSON(b *testing.B) {
	api := &Apic{}
	jsonData := map[string]any{
		"name": "test",
		"age":  25,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		api.SetData(NewJSONData(jsonData))
	}
}

func BenchmarkSetJSONBackwardCompatible(b *testing.B) {
	api := &Apic{}
	jsonData := map[string]any{
		"name": "test",
		"age":  25,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		api.SetJSON(jsonData)
	}
}

// 测试数据类型枚举
func TestDataTypes(t *testing.T) {
	// 验证数据类型常量
	if DataTypeJSON != 0 {
		t.Error("DataTypeJSON should be 0")
	}
	if DataTypeForm != 1 {
		t.Error("DataTypeForm should be 1")
	}
	if DataTypeWWWForm != 2 {
		t.Error("DataTypeWWWForm should be 2")
	}
	if DataTypeQuery != 3 {
		t.Error("DataTypeQuery should be 3")
	}
	if DataTypeHeader != 4 {
		t.Error("DataTypeHeader should be 4")
	}
}

// 测试SetDataRequest结构体
func TestSetDataRequest(t *testing.T) {
	// 测试基本创建
	req := NewSetDataRequest(DataTypeJSON, map[string]string{"key": "value"})
	if req == nil {
		t.Error("SetDataRequest should be created")
	}
	if req.Type != DataTypeJSON {
		t.Error("Type should be DataTypeJSON")
	}

	// 测试链式调用
	req = req.WithHeaders(map[string]string{"Authorization": "Bearer token"})
	req = req.WithContentType("application/json")
	req = req.WithEncoding("utf-8")

	if req.ContentType != "application/json" {
		t.Error("ContentType should be set")
	}
	if req.Encoding != "utf-8" {
		t.Error("Encoding should be set")
	}
	if req.Headers["Authorization"] != "Bearer token" {
		t.Error("Headers should be set")
	}
}
