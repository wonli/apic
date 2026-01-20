package apic

import (
	"net/http"
)

// Context 中间件上下文，类似于Gin的Context
type Context struct {
	Request  *http.Request
	Response *http.Response
	aborted  bool
	index    int
	handlers []MiddlewareFunc

	Id         *ApiId
	HttpClient *StdlibClient

	// 新增：用于流式响应日志处理
	StreamLogChan chan string   // 用于异步传递流式数据
	StreamLogDone chan struct{} // 用于等待异步日志完成
}

// Next 调用链中的下一个中间件
func (c *Context) Next() {
	c.index++
	for c.index < len(c.handlers) && !c.aborted {
		c.handlers[c.index](c)
		c.index++
	}
}

// Abort 中止中间件链的执行
func (c *Context) Abort() {
	c.aborted = true
}

// IsAborted 检查是否已中止
func (c *Context) IsAborted() bool {
	return c.aborted
}

// MiddlewareFunc 函数式中间件类型，使用Context模式
type MiddlewareFunc func(ctx *Context)
