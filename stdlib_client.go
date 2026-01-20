package apic

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// StdlibClient 基于标准库的HTTP客户端
type StdlibClient struct {
	client      *http.Client
	proxy       string
	debug       bool
	headers     map[string]string
	middlewares []MiddlewareFunc
	// 中间件上下文信息
	contextId         *ApiId
	contextHttpClient *StdlibClient
	middlewareCtx     *Context // 保存中间件执行时的Context引用
}

// NewStdlibClient 创建新的标准库HTTP客户端
func NewStdlibClient() *StdlibClient {
	c := &StdlibClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// 允许最多20次重定向，避免"stopped after 10 redirects"错误
				if len(via) >= 20 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		headers:     make(map[string]string),
		middlewares: make([]MiddlewareFunc, 0),
	}
	return c
}

// SetProxy 设置代理
func (c *StdlibClient) SetProxy(proxy string) *StdlibClient {
	c.proxy = proxy
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err == nil {
			c.client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}
	return c
}

// SetDebug 设置调试模式
func (c *StdlibClient) SetDebug(debug bool) *StdlibClient {
	c.debug = debug
	return c
}

// SetTimeout 设置请求超时时间
func (c *StdlibClient) SetTimeout(timeout time.Duration) *StdlibClient {
	if c.client != nil {
		c.client.Timeout = timeout
	}
	return c
}

// SetHeader 设置请求头
func (c *StdlibClient) SetHeader(key, value string) *StdlibClient {
	c.headers[key] = value
	return c
}

// SetHeaders 批量设置请求头
func (c *StdlibClient) SetHeaders(headers map[string]string) *StdlibClient {
	for k, v := range headers {
		c.headers[k] = v
	}
	return c
}

// prepareRequestBody 准备请求体
func (c *StdlibClient) prepareRequestBody(data *SetDataRequest) (io.Reader, string, error) {
	if data == nil || data.Content == nil {
		return nil, "", nil
	}

	switch data.Type {
	case DataTypeJSON:
		jsonData, err := json.Marshal(data.Content)
		if err != nil {
			return nil, "", err
		}
		contentType := "application/json"
		if data.ContentType != "" {
			contentType = data.ContentType
		}
		return bytes.NewReader(jsonData), contentType, nil

	case DataTypeXML:
		xmlData, err := xml.Marshal(data.Content)
		if err != nil {
			return nil, "", err
		}
		contentType := "application/xml"
		if data.ContentType != "" {
			contentType = data.ContentType
		}
		return bytes.NewReader(xmlData), contentType, nil

	case DataTypeWWWForm:
		var formData url.Values
		switch v := data.Content.(type) {
		case url.Values:
			formData = v
		case map[string]string:
			formData = make(url.Values)
			for k, val := range v {
				formData.Set(k, val)
			}
		case map[string]any:
			formData = make(url.Values)
			for k, val := range v {
				formData.Set(k, fmt.Sprintf("%v", val))
			}
		default:
			return nil, "", fmt.Errorf("unsupported form data type: %T", data.Content)
		}
		contentType := "application/x-www-form-urlencoded"
		if data.ContentType != "" {
			contentType = data.ContentType
		}
		return strings.NewReader(formData.Encode()), contentType, nil

	case DataTypeText:
		text := fmt.Sprintf("%v", data.Content)
		contentType := "text/plain"
		if data.ContentType != "" {
			contentType = data.ContentType
		}
		return strings.NewReader(text), contentType, nil

	case DataTypeRaw:
		switch v := data.Content.(type) {
		case []byte:
			contentType := "application/octet-stream"
			if data.ContentType != "" {
				contentType = data.ContentType
			}
			return bytes.NewReader(v), contentType, nil
		case io.Reader:
			contentType := "application/octet-stream"
			if data.ContentType != "" {
				contentType = data.ContentType
			}
			return v, contentType, nil
		case string:
			contentType := "text/plain"
			if data.ContentType != "" {
				contentType = data.ContentType
			}
			return strings.NewReader(v), contentType, nil
		default:
			return nil, "", fmt.Errorf("unsupported raw data type: %T", data.Content)
		}

	default:
		return nil, "", fmt.Errorf("unsupported data type: %v", data.Type)
	}
}

// DoRequest 执行HTTP请求
func (c *StdlibClient) DoRequest(ctx context.Context, method, url string, data *SetDataRequest, query url.Values) (*http.Response, error) {
	// 处理查询参数
	if len(query) > 0 {
		if strings.Contains(url, "?") {
			url += "&" + query.Encode()
		} else {
			url += "?" + query.Encode()
		}
	}

	// 准备请求体
	body, contentType, err := c.prepareRequestBody(data)
	if err != nil {
		return nil, err
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	// 设置Content-Type
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// 设置通用请求头
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// 设置数据特定的请求头
	if data != nil {
		for k, v := range data.Headers {
			req.Header.Set(k, v)
		}
	}

	// 执行中间件链
	resp, err := c.executeMiddlewareChain(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GET 执行GET请求
func (c *StdlibClient) GET(ctx context.Context, url string, query url.Values) (*http.Response, error) {
	return c.DoRequest(ctx, "GET", url, nil, query)
}

// POST 执行POST请求
func (c *StdlibClient) POST(ctx context.Context, url string, data *SetDataRequest, query url.Values) (*http.Response, error) {
	return c.DoRequest(ctx, "POST", url, data, query)
}

// PUT 执行PUT请求
func (c *StdlibClient) PUT(ctx context.Context, url string, data *SetDataRequest, query url.Values) (*http.Response, error) {
	return c.DoRequest(ctx, "PUT", url, data, query)
}

// DELETE 执行DELETE请求
func (c *StdlibClient) DELETE(ctx context.Context, url string, query url.Values) (*http.Response, error) {
	return c.DoRequest(ctx, "DELETE", url, nil, query)
}

// PATCH 执行PATCH请求
func (c *StdlibClient) PATCH(ctx context.Context, url string, data *SetDataRequest, query url.Values) (*http.Response, error) {
	return c.DoRequest(ctx, "PATCH", url, data, query)
}

// AddMiddleware 添加中间件
func (c *StdlibClient) AddMiddleware(middleware MiddlewareFunc) *StdlibClient {
	c.middlewares = append(c.middlewares, middleware)
	return c
}

// ClearMiddlewares 清空所有中间件
func (c *StdlibClient) ClearMiddlewares() *StdlibClient {
	c.middlewares = make([]MiddlewareFunc, 0)
	return c
}

// GetMiddlewares 获取所有中间件
func (c *StdlibClient) GetMiddlewares() []MiddlewareFunc {
	return c.middlewares
}

// SetContextInfo 设置中间件上下文信息
func (c *StdlibClient) SetContextInfo(id *ApiId, httpClient *StdlibClient) *StdlibClient {
	c.contextId = id
	c.contextHttpClient = httpClient
	return c
}

// executeMiddlewareChain 执行中间件链
func (c *StdlibClient) executeMiddlewareChain(req *http.Request) (*http.Response, error) {
	return c.executeMiddlewareChainWithContext(req, c.contextId, c.contextHttpClient)
}

// Reset 重置客户端状态，用于对象池复用
func (c *StdlibClient) Reset() *StdlibClient {
	// 重置代理设置
	c.proxy = ""
	// 重置调试模式
	c.debug = false
	// 清空自定义头部
	c.headers = make(map[string]string)
	// 清空中间件
	c.middlewares = make([]MiddlewareFunc, 0)
	// 清空上下文信息
	c.contextId = nil
	c.contextHttpClient = nil
	c.middlewareCtx = nil
	// 重置HTTP客户端的Transport（清除代理设置）
	// 注意：保持默认的Transport，不要重置为nil或新的Transport
	// 这样可以避免连接问题
	return c
}

// executeMiddlewareChainWithContext 执行中间件链，支持传入ApiId和HttpClient
func (c *StdlibClient) executeMiddlewareChainWithContext(req *http.Request, id *ApiId, httpClient *StdlibClient) (*http.Response, error) {
	var resp *http.Response
	var err error

	// 复制中间件列表
	middlewareFuncs := make([]MiddlewareFunc, len(c.middlewares))
	copy(middlewareFuncs, c.middlewares)

	// 添加实际的HTTP请求处理函数
	middlewareFuncs = append(middlewareFuncs, func(ctx *Context) {
		// 执行HTTP请求
		resp, err = c.client.Do(req)
		// 立即设置响应到上下文中
		ctx.Response = resp
	})

	// 创建中间件上下文
	ctx := &Context{
		Request:    req,
		Response:   nil,
		index:      -1,
		handlers:   middlewareFuncs,
		Id:         id,
		HttpClient: httpClient,
	}

	// 保存Context引用，供流式响应处理使用
	c.middlewareCtx = ctx

	// 开始执行中间件链
	ctx.Next()

	return resp, err
}
