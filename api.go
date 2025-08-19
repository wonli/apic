package apic

import (
	"context"
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
	SetData(data *SetDataRequest) error           // SetData - 统一的数据设置接口
	Setup() (Api, error)                           // Setup for the API.
	HttpMethod() HttpMethod                        // HTTP method of the request.
	Debug() bool                                   // Whether to run in debug mode.
	UseContext(ctx context.Context) error          // Use context.
	OnRequest() error                              // Handle request data.
	OnHttpStatusError(code int, resp []byte) error // Handle HTTP status errors.
	OnResponse(resp []byte) (*ResponseData, error) // Process response data.
	OnEvent(callback func(data string))            // 注册事件回调函数
	ReceiveEvent(data string)                      // 注册收到事件时的处理方法
	
	// 向后兼容的方法 - 已废弃，建议使用SetData
	SetJSON(data interface{}) Api                  // Deprecated: 使用SetData(NewJSONData(data))替代
	SetForm(data interface{}) Api                  // Deprecated: 使用SetData(NewFormData(data))替代
	SetWWWForm(data interface{}) Api               // Deprecated: 使用SetData(NewWWWFormData(data))替代
	SetQuery(data interface{}) Api                 // Deprecated: 使用SetData(NewQueryData(data))替代
	SetHeader(data interface{}) Api                // Deprecated: 使用SetData(NewHeaderData(data))替代
	SetHeaderRaw(data interface{}) Api             // Deprecated: 使用SetData(NewHeaderData(data))替代
	NoAutoContentType() Api                        // Deprecated: 在SetData中设置ContentType
	SetProxy(proxy string) Api                     // Deprecated: 使用ApiClients.WithProxy()替代
}
