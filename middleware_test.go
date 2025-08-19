package apic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMiddleware 测试中间件状态
type TestMiddleware struct {
	beforeRequestCalled bool
	afterResponseCalled bool
	request             *http.Request
	response            *http.Response
}

// NewTestMiddleware 创建测试中间件
func NewTestMiddleware() (*TestMiddleware, MiddlewareFunc) {
	tm := &TestMiddleware{}
	mw := func(ctx *Context) {
		// 请求前调用
		tm.beforeRequestCalled = true
		tm.request = ctx.Request

		// 调用下一个中间件
		ctx.Next()

		// 响应后调用
		tm.afterResponseCalled = true
		tm.response = ctx.Response
	}
	return tm, mw
}

func TestMiddlewareInterface(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "test"}`))
	}))
	defer server.Close()

	// 创建客户端
	client := NewStdlibClient()

	// 添加测试中间件
	testMw, mwFunc := NewTestMiddleware()
	client.AddMiddleware(mwFunc)

	// 发送请求
	ctx := context.Background()
	resp, err := client.GET(ctx, server.URL, nil)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证中间件被调用
	if !testMw.beforeRequestCalled {
		t.Error("BeforeRequest 方法未被调用")
	}

	if !testMw.afterResponseCalled {
		t.Error("AfterResponse 方法未被调用")
	}

	// 验证请求和响应对象
	if testMw.request == nil {
		t.Error("请求对象为空")
	}

	if testMw.response == nil {
		t.Error("响应对象为空")
	}

	if testMw.response.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, testMw.response.StatusCode)
	}
}

func TestDebugMiddleware(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello World"))
	}))
	defer server.Close()

	// 创建客户端并添加调试中间件
	client := NewStdlibClient()
	debugMw := NewDebugMiddleware()
	client.AddMiddleware(debugMw)

	// 发送请求
	ctx := context.Background()
	resp, err := client.GET(ctx, server.URL, nil)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证中间件已添加
	middlewares := client.GetMiddlewares()
	if len(middlewares) == 0 {
		t.Error("没有找到中间件")
	}

	// 验证响应状态
	if resp.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, resp.StatusCode)
	}
}

func TestMultipleMiddlewares(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// 创建客户端
	client := NewStdlibClient()

	// 添加多个中间件
	mw1, mwFunc1 := NewTestMiddleware()
	mw2, mwFunc2 := NewTestMiddleware()
	mw3, mwFunc3 := NewTestMiddleware()

	client.AddMiddleware(mwFunc1)
	client.AddMiddleware(mwFunc2)
	client.AddMiddleware(mwFunc3)

	// 发送请求
	ctx := context.Background()
	resp, err := client.GET(ctx, server.URL, nil)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证所有中间件都被调用
	middlewares := []*TestMiddleware{mw1, mw2, mw3}
	for i, mw := range middlewares {
		if !mw.beforeRequestCalled {
			t.Errorf("中间件 %d 的 BeforeRequest 未被调用", i+1)
		}
		if !mw.afterResponseCalled {
			t.Errorf("中间件 %d 的 AfterResponse 未被调用", i+1)
		}
	}
}

func TestClearMiddlewares(t *testing.T) {
	// 创建客户端
	client := NewStdlibClient()

	// 添加中间件
	_, mwFunc1 := NewTestMiddleware()
	_, mwFunc2 := NewTestMiddleware()
	client.AddMiddleware(mwFunc1)
	client.AddMiddleware(mwFunc2)

	// 验证中间件数量
	if len(client.GetMiddlewares()) < 2 {
		t.Error("中间件添加失败")
	}

	// 清空中间件
	client.ClearMiddlewares()

	// 验证中间件已清空
	if len(client.GetMiddlewares()) != 0 {
		t.Error("中间件清空失败")
	}
}

// NewAuthTestMiddleware 创建认证测试中间件
func NewAuthTestMiddleware(token string) MiddlewareFunc {
	return func(ctx *Context) {
		if token != "" {
			// 请求前添加认证头
			ctx.Request.Header.Set("Authorization", "Bearer "+token)
		}
		// 调用下一个中间件
		ctx.Next()
	}
}

func TestAuthMiddleware(t *testing.T) {
	// 创建测试服务器，验证认证头
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Authenticated"))
	}))
	defer server.Close()

	// 创建客户端
	client := NewStdlibClient()

	// 添加认证中间件
	authMw := NewAuthTestMiddleware("test-token")
	client.AddMiddleware(authMw)

	// 发送请求
	ctx := context.Background()
	resp, err := client.GET(ctx, server.URL, nil)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证认证成功
	if resp.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, resp.StatusCode)
	}
}
