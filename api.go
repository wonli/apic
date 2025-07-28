package apic

import (
	"context"
	"github.com/guonaihong/gout/dataflow"
	"net/url"
)

// Api api client interface.
type Api interface {
	Url() string                                   // Request API full URL.
	Path() string                                  // Request path.
	Query() url.Values                             // URL query parameters.
	Headers() Params                               // Headers required for the request.
	PostBody() any                                 // Request parameters.
	FormData() any                                 // Form data as map[string]string.
	WWWFormData() any                              // Form data as map[string]string.
	SetData(c *dataflow.DataFlow) error            // SetData
	Setup() (Api, error)                           // Setup for the API.
	HttpMethod() HttpMethod                        // HTTP method of the request.
	Debug() bool                                   // Whether to run in debug mode.
	UseContext(ctx context.Context) error          // Use context.
	OnRequest() error                              // Handle request data.
	OnHttpStatusError(code int, resp []byte) error // Handle HTTP status errors.
	OnResponse(resp []byte) (*ResponseData, error) // Process response data.
	OnEvent(callback func(data string))            // 注册事件回调函数
	ReceiveEvent(data string)                      // 注册收到事件时的处理方法
}
