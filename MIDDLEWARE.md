# APIC 中间件系统

APIC v2 引入了灵活的中间件系统，参考了 Gin 框架的设计理念，使用 Context 模式允许用户在请求发送前和响应接收后执行自定义逻辑。

## 中间件类型

```go
// Context 中间件上下文，类似于Gin的Context
type Context struct {
    Request  *http.Request
    Response *http.Response
    // ... 其他字段
}

// MiddlewareFunc 函数式中间件类型，使用Context模式
type MiddlewareFunc func(ctx *Context)
```

中间件使用 Context 模式：
- `ctx.Request` 包含请求信息，可以在请求前修改
- `ctx.Response` 包含响应信息，在 `ctx.Next()` 调用后可用
- 调用 `ctx.Next()` 继续执行下一个中间件
- 调用 `ctx.Abort()` 可以中止中间件链的执行

## 内置中间件

### DebugMiddleware

内置的调试中间件，提供详细的请求和响应日志：

```go
client := apic.NewStdlibClient()
debugMw := apic.NewDebugMiddleware(true) // 创建调试中间件
client.AddMiddleware(debugMw) // 添加到客户端
```

调试输出格式：
```
< GET /api/data HTTP/1.1
< Host: api.example.com
< Authorization: Bearer token
< 
< HTTP/1.1 200 OK
< Content-Type: application/json
< 
> {"data": "response"}
```

## 自定义中间件

### 1. 日志中间件

```go
func NewLogMiddleware(logger *log.Logger) apic.MiddlewareFunc {
    return func(ctx *apic.Context) {
        // 请求前
        logger.Printf("发送请求: %s %s", ctx.Request.Method, ctx.Request.URL.String())
        
        // 调用下一个中间件
        ctx.Next()
        
        // 响应后
        if ctx.Response != nil {
            logger.Printf("收到响应: %d", ctx.Response.StatusCode)
        }
    }
}
```

### 2. 认证中间件

```go
func NewAuthMiddleware(token string) apic.MiddlewareFunc {
    return func(ctx *apic.Context) {
        // 请求前添加认证头
        if token != "" {
            ctx.Request.Header.Set("Authorization", "Bearer "+token)
        }
        
        // 调用下一个中间件
        ctx.Next()
        
        // 响应后检查认证状态
        if ctx.Response != nil && ctx.Response.StatusCode == 401 {
            log.Println("认证失败，请检查token")
        }
    }
}
```

### 3. 计时中间件

```go
func NewTimingMiddleware() apic.MiddlewareFunc {
    return func(ctx *apic.Context) {
        // 请求前记录开始时间
        startTime := time.Now()
        
        // 调用下一个中间件
        ctx.Next()
        
        // 响应后计算耗时
        duration := time.Since(startTime)
        log.Printf("请求耗时: %v", duration)
    }
}
```

### 4. 重试中间件

```go
func NewRetryMiddleware(maxRetries int) apic.MiddlewareFunc {
    return func(ctx *apic.Context) {
        var retryCount int
        
        for retryCount <= maxRetries {
            // 调用下一个中间件
            ctx.Next()
            
            // 检查是否需要重试
            if ctx.Response == nil || ctx.Response.StatusCode < 500 {
                break // 成功或非服务器错误，不需要重试
            }
            
            if retryCount < maxRetries {
                retryCount++
                log.Printf("服务器错误，准备第 %d 次重试", retryCount)
                // 重置中间件链索引以便重试
                // 注意：实际重试逻辑需要在更高层实现
            }
        }
    }
}
```

## 使用示例

### 基本使用

```go
package main

import (
    "context"
    "log"
    "github.com/wonli/apic/v2"
)

func main() {
    // 创建客户端
    client := apic.NewStdlibClient()
    
    // 添加多个中间件
    client.AddMiddleware(NewLogMiddleware(log.Default()))
    client.AddMiddleware(NewAuthMiddleware("your-api-token"))
    client.AddMiddleware(NewTimingMiddleware())
    
    // 添加内置调试中间件
    client.AddMiddleware(apic.NewDebugMiddleware(true))
    
    // 发送请求
    ctx := context.Background()
    resp, err := client.GET(ctx, "https://api.example.com/data", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()
    
    // 清空所有中间件（如果需要）
    // client.ClearMiddlewares()
}
```

### 内联中间件

```go
package main

import (
    "context"
    "log"
    "github.com/wonli/apic/v2"
)

func main() {
    client := apic.NewStdlibClient()
    
    // 使用内联中间件
    client.AddMiddleware(func(ctx *apic.Context) {
        // 请求前
        log.Printf("Sending %s request to %s", ctx.Request.Method, ctx.Request.URL)
        
        // 调用下一个中间件
        ctx.Next()
        
        // 响应后
        if ctx.Response != nil {
            log.Printf("Received response with status %d", ctx.Response.StatusCode)
        }
    })
    
    // CORS 中间件示例（类似 Gin 的 CORS）
    client.AddMiddleware(func(ctx *apic.Context) {
        // 请求前设置 CORS 头
        ctx.Request.Header.Set("Access-Control-Request-Method", ctx.Request.Method)
        ctx.Request.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
        
        // 调用下一个中间件
        ctx.Next()
    })
    
    ctx := context.Background()
    resp, err := client.GET(ctx, "https://api.example.com/data", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()
}
```

## 中间件管理方法

```go
// 添加中间件
client.AddMiddleware(middleware)

// 获取所有中间件
middlewares := client.GetMiddlewares()

// 清空所有中间件
client.ClearMiddlewares()
```

## 执行顺序

1. **中间件1 请求前逻辑**: 按添加顺序执行
2. **中间件2 请求前逻辑**: 按添加顺序执行
3. **发送HTTP请求**
4. **中间件2 响应后逻辑**: 按相反顺序执行
5. **中间件1 响应后逻辑**: 按相反顺序执行

每个中间件通过调用 `ctx.Next()` 来继续执行链中的下一个中间件。

## Context 方法

- `ctx.Next()`: 继续执行下一个中间件
- `ctx.Abort()`: 中止中间件链的执行
- `ctx.IsAborted()`: 检查是否已中止

## 最佳实践

1. **轻量级**: 中间件应该尽可能轻量，避免阻塞请求
2. **调用 Next()**: 记得调用 `ctx.Next()` 继续执行链
3. **错误处理**: 中间件内部应该处理好错误，避免影响主流程
4. **资源清理**: 如果中间件创建了资源，确保在适当时机清理
5. **并发安全**: 中间件函数应该是并发安全的

## 注意事项

- 中间件在每个请求中都会被调用
- 修改请求头等操作应该在调用 `ctx.Next()` 之前进行
- 响应体的读取会影响后续处理，如需读取请重新设置 `ctx.Response.Body`
- 中间件的执行顺序很重要，认证中间件通常应该在日志中间件之前
- 如果不调用 `ctx.Next()`，后续中间件将不会执行

通过中间件系统，APIC 提供了强大的扩展能力，让用户可以根据需要定制HTTP客户端的行为。