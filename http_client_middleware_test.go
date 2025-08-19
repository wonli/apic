package apic

import (
	"testing"
)

// TestApiClientsMiddleware 测试 ApiClients 中间件功能
func TestApiClientsMiddleware(t *testing.T) {
	// 创建测试中间件
	middleware1 := MiddlewareFunc(func(ctx *Context) {
		// 简单的中间件，不做任何操作
		ctx.Next()
	})

	middleware2 := MiddlewareFunc(func(ctx *Context) {
		// 简单的中间件，不做任何操作
		ctx.Next()
	})

	// 测试添加中间件
	client := Init()
	// 先清空可能存在的中间件
	client.ClearMiddlewares()
	client.Use(middleware1, middleware2)

	// 验证中间件数量
	middlewares := client.GetMiddlewares()
	if len(middlewares) != 2 {
		t.Errorf("Expected 2 middlewares, got %d", len(middlewares))
	}

	// 测试清空中间件
	client.ClearMiddlewares()
	middlewares = client.GetMiddlewares()
	if len(middlewares) != 0 {
		t.Errorf("Expected 0 middlewares after clear, got %d", len(middlewares))
	}
}

// TestApiClientsChaining 测试 ApiClients 链式调用
func TestApiClientsChaining(t *testing.T) {
	client := Init()

	// 测试链式调用
	result := client.
		Use(MiddlewareFunc(func(ctx *Context) { ctx.Next() })).
		WithProxy("http://proxy.example.com").
		Use(MiddlewareFunc(func(ctx *Context) { ctx.Next() }))

	if result != client {
		t.Error("Chaining should return the same client instance")
	}

	if len(client.GetMiddlewares()) != 2 {
		t.Errorf("Expected 2 middlewares, got %d", len(client.GetMiddlewares()))
	}

	if client.proxy != "http://proxy.example.com" {
		t.Errorf("Expected proxy to be set, got %s", client.proxy)
	}
}
