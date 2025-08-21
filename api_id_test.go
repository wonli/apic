package apic

import (
	"net/url"
	"testing"
)

// TestApiId 测试用的 API 实现
type TestApiId struct {
	*Apic
}

func (t *TestApiId) Url() string {
	return "https://httpbin.org"
}

func (t *TestApiId) Path() string {
	return "/get"
}

func (t *TestApiId) Query() url.Values {
	u := url.Values{}
	u.Add("test", "value")
	return u
}

func (t *TestApiId) HttpMethod() HttpMethod {
	return GET
}

func TestApiIdCall(t *testing.T) {
	// 创建测试用的 ApiId
	apiId := &ApiId{
		Name: "test.api",
		Client: &TestApiId{
			Apic: &Apic{},
		},
	}

	// 测试 Call 方法
	err := apiId.Call(nil)
	if err != nil {
		t.Errorf("ApiId.Call() failed: %v", err)
	}
}

func TestApiIdCallApi(t *testing.T) {
	// 创建测试用的 ApiId
	apiId := &ApiId{
		Name: "test.api",
		Client: &TestApiId{
			Apic: &Apic{},
		},
	}

	// 测试 CallApi 方法
	resp, err := apiId.CallApi(nil)
	if err != nil {
		t.Errorf("ApiId.CallApi() failed: %v", err)
		return
	}

	if resp == nil {
		t.Error("ApiId.CallApi() returned nil response")
		return
	}

	if resp.HttpStatus != 200 {
		t.Errorf("Expected HTTP status 200, got %d", resp.HttpStatus)
	}

	if len(resp.Data) == 0 {
		t.Error("Response data is empty")
	}
}

func TestApiIdCallBindJson(t *testing.T) {
	// 创建测试用的 ApiId
	apiId := &ApiId{
		Name: "test.api",
		Client: &TestApiId{
			Apic: &Apic{},
		},
	}

	// 定义响应结构体
	var response map[string]interface{}

	// 测试 CallBindJson 方法
	err := apiId.CallBindJson(&response, nil)
	if err != nil {
		t.Errorf("ApiId.CallBindJson() failed: %v", err)
		return
	}

	// 验证响应数据
	if response == nil {
		t.Error("CallBindJson did not populate response")
	}
}

func TestApiIdCallFunc(t *testing.T) {
	// 创建测试用的 ApiId
	apiId := &ApiId{
		Name: "test.api",
		Client: &TestApiId{
			Apic: &Apic{},
		},
	}

	// 定义回调函数
	callbackCalled := false
	callback := func(a *Api, data []byte) error {
		callbackCalled = true
		if len(data) == 0 {
			t.Error("Callback received empty data")
		}
		return nil
	}

	// 测试 CallFunc 方法
	err := apiId.CallFunc(nil, callback)
	if err != nil {
		t.Errorf("ApiId.CallFunc() failed: %v", err)
		return
	}

	if !callbackCalled {
		t.Error("Callback function was not called")
	}
}