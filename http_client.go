package apic

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var Apis *ApiClients
var once sync.Once

// ApiClients api clients list
type ApiClients struct {
	ctx           context.Context
	proxy         string
	named         map[string]*ApiId
	middlewares   []MiddlewareFunc
	clientFactory *ClientFactory
	mu            sync.RWMutex
}

// Init 返回全局单例实例，适用于:
// - 对外提供API服务
// - 全局配置相对固定的场景
// - 需要在整个应用中共享API配置
// 注意: 全局单例的配置修改会影响所有使用者
func Init() *ApiClients {
	once.Do(func() {
		Apis = &ApiClients{
			ctx:           context.Background(),
			named:         map[string]*ApiId{},
			clientFactory: NewClientFactory(20),
		}
	})

	return Apis
}

// NewApiClient 创建新的独立实例，适用于:
// - 多租户场景
// - 测试环境
// - 不同业务模块需要不同配置
// - 需要配置隔离的场景
func NewApiClient() *ApiClients {
	return &ApiClients{
		ctx:           context.Background(),
		named:         map[string]*ApiId{},
		clientFactory: NewClientFactory(20),
	}
}

type ClientConfig struct {
	MaxPoolSize    int
	DefaultContext context.Context
	DefaultProxy   string
}

func NewApiClientWithConfig(config *ClientConfig) *ApiClients {
	if config == nil {
		config = &ClientConfig{MaxPoolSize: 20}
	}
	return &ApiClients{
		ctx:           config.DefaultContext,
		proxy:         config.DefaultProxy,
		named:         map[string]*ApiId{},
		clientFactory: NewClientFactory(config.MaxPoolSize),
	}
}

func (a *ApiClients) Named(api *ApiId) {
	a.mu.Lock()
	defer a.mu.Unlock()

	_, ok := a.named[api.Name]
	if ok {
		panic("ApiId registered multiple times")
	}

	a.named[api.Name] = api
}

func (a *ApiClients) WithContext(ctx context.Context) *ApiClients {
	a.ctx = ctx
	return a
}

func (a *ApiClients) WithProxy(proxy string) *ApiClients {
	a.proxy = proxy
	return a
}

// AddMiddleware 添加中间件
func (a *ApiClients) Use(middlewares ...MiddlewareFunc) *ApiClients {
	a.middlewares = append(a.middlewares, middlewares...)
	return a
}

// ClearMiddlewares 清空所有中间件
func (a *ApiClients) ClearMiddlewares() *ApiClients {
	a.middlewares = nil
	return a
}

// GetMiddlewares 获取所有中间件
func (a *ApiClients) GetMiddlewares() []MiddlewareFunc {
	return a.middlewares
}

// GetClientFactoryStats 获取客户端工厂统计信息
func (a *ApiClients) GetClientFactoryStats() *FactoryStats {
	if a.clientFactory == nil {
		return nil
	}

	return a.clientFactory.GetStats()
}

// SetClientFactoryMaxSize 设置客户端工厂最大池大小
func (a *ApiClients) SetClientFactoryMaxSize(maxSize int) *ApiClients {
	a.clientFactory = NewClientFactory(maxSize)
	return a
}

func (a *ApiClients) Call(id *ApiId, op *Options) error {
	_, err := a.getApiData(id, op)
	if err != nil {
		return err
	}

	return nil
}

func (a *ApiClients) CallApi(id *ApiId, op *Options) (*ResponseData, error) {
	return a.getApiData(id, op)
}

func (a *ApiClients) CallNamed(name string, op *Options) (*ResponseData, error) {
	a.mu.RLock()
	id, ok := a.named[name]
	a.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("named api not registered")
	}

	return a.getApiData(id, op)
}

func (a *ApiClients) CallBindJson(id *ApiId, resp any, op *Options) error {
	_, err := a.getApiData(id, op)
	if err != nil {
		return err
	}

	return id.Response.BindJson(resp)
}

func (a *ApiClients) CallFunc(id *ApiId, op *Options, callback func(a *Api, data []byte) error) error {
	apiData, err := a.getApiData(id, op)
	if err != nil {
		return err
	}

	return callback(&id.Client, apiData.Data)
}

func (a *ApiClients) getApiData(id *ApiId, op *Options) (*ResponseData, error) {
	api, err := id.Client.Setup()
	if err != nil {
		return nil, err
	}

	if api == nil {
		api = id.Client
	}

	if id.Request == nil {
		id.Request = &RequestData{}
	}

	if op == nil {
		op = &Options{}
	}

	id.Request.ApiId = id.Name
	id.Request.InitFromApiClient(id.Client)

	ctx := context.Background()
	if a.ctx != nil {
		ctx = a.ctx
		err = api.UseContext(a.ctx)
		if err != nil {
			return nil, err
		}
	}

	// Merge query parameters
	if op.Query != nil {
		if id.Request.Query == nil {
			id.Request.Query = op.Query
		} else {
			for key, valData := range op.Query {
				if len(valData) == 1 {
					id.Request.Query.Add(key, valData[0])
				} else {
					for _, val := range valData {
						id.Request.Query.Add(key, val)
					}
				}
			}
		}
	}

	//set postBody
	if op.PostBody != nil {
		id.Request.PostBody = op.PostBody
	}

	//set header
	if op.Headers != nil {
		if id.Request.Header == nil {
			id.Request.Header = op.Headers
		} else {
			for key, val := range op.Headers {
				id.Request.Header[key] = val
			}
		}
	}

	err = api.OnRequest()
	if err != nil {
		return nil, err
	}

	// 从客户端池获取客户端
	client := a.clientFactory.AcquireClient()
	defer a.clientFactory.ReleaseClient(client)

	if client == nil {
		return nil, fmt.Errorf("acquired client is nil")
	}

	// 配置客户端
	if a.proxy != "" {
		client.SetProxy(a.proxy)
	}

	// 设置超时时间
	if op.Timeout > 0 {
		client.SetTimeout(op.Timeout)
	} else if id.Stream {
		client.SetTimeout(time.Minute * 15)
	}

	// 设置中间件上下文信息
	client.SetContextInfo(id, client)

	// 添加中间件
	for _, middleware := range a.middlewares {
		client.AddMiddleware(middleware)
	}

	// 检查是否需要启用调试模式
	if id.Request.Debug || op.Debug || api.Debug() {
		client.SetDebug(true)
		// 自动添加调试中间件
		client.AddMiddleware(NewDebugMiddleware())
	}

	// 设置请求头
	if id.Request.Header != nil {
		headers := make(map[string]string)
		for k, v := range id.Request.Header {
			headers[k] = fmt.Sprintf("%v", v)
		}
		client.SetHeaders(headers)
	}

	// 准备请求数据
	var requestData *SetDataRequest
	if id.Request.Form != nil {
		requestData = NewFormData(convertToStringMap(id.Request.Form))
	} else if id.Request.WWWForm != nil {
		if wwwForm, ok := id.Request.WWWForm.(url.Values); ok {
			requestData = NewWWWFormData(wwwForm)
		} else {
			requestData = NewWWWFormData(convertToURLValues(id.Request.WWWForm))
		}
	} else if id.Request.PostBody != nil {
		requestData = NewJSONData(id.Request.PostBody)
	}

	// 调用SetData接口
	if requestData != nil {
		err = id.Client.SetData(requestData)
		if err != nil {
			return nil, err
		}
	}

	// 构建完整URL
	apiAddress := id.Request.Url + id.Request.Path

	// 执行HTTP请求
	var response *http.Response
	switch id.Request.HttpMethod {
	case POST:
		response, err = client.POST(ctx, apiAddress, requestData, id.Request.Query)
	case DELETE:
		response, err = client.DELETE(ctx, apiAddress, id.Request.Query)
	case HEAD:
		response, err = client.DoRequest(ctx, "HEAD", apiAddress, nil, id.Request.Query)
	case OPTIONS:
		response, err = client.DoRequest(ctx, "OPTIONS", apiAddress, nil, id.Request.Query)
	case PATCH:
		response, err = client.PATCH(ctx, apiAddress, requestData, id.Request.Query)
	case PUT:
		response, err = client.PUT(ctx, apiAddress, requestData, id.Request.Query)
	default:
		response, err = client.GET(ctx, apiAddress, id.Request.Query)
	}

	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if response == nil {
		return nil, fmt.Errorf("HTTP response is nil")
	}

	defer response.Body.Close()

	// 初始化响应数据
	id.Response = &ResponseData{
		HttpStatus: response.StatusCode,
		Header:     make(http.Header),
	}

	// 复制响应头
	for k, v := range response.Header {
		id.Response.Header[k] = v
	}

	// 处理流式响应
	if id.Stream {
		// 处理 text/event-stream
		contentType := response.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/event-stream") {
			bodyData, err := io.ReadAll(response.Body)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("%s", string(bodyData))
		}

		buf := bufio.NewReader(response.Body)
		for {
			line, err := buf.ReadString('\n')
			if err != nil && err != io.EOF {
				return nil, err
			}

			api.ReceiveEvent(strings.TrimSpace(line))
			if err == io.EOF {
				break
			}
		}

		return id.Response, nil
	}

	// 读取响应体
	bodyData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	// 处理HTTP状态错误
	id.Response.Data = bodyData
	if id.Response.HttpStatus != http.StatusOK {
		err = api.OnHttpStatusError(id.Response.HttpStatus, id.Response.Data)
		if err != nil {
			return id.Response, err
		}
	}

	// 处理响应数据
	responseData, err := api.OnResponse(id.Response.Data)
	if err != nil {
		return nil, err
	}

	// 确保返回的ResponseData包含正确的HttpStatus
	if responseData != nil {
		responseData.HttpStatus = id.Response.HttpStatus
		responseData.Header = id.Response.Header
	}

	return responseData, nil
}

// 辅助函数：转换为字符串映射
func convertToStringMap(data any) map[string]string {
	result := make(map[string]string)
	switch v := data.(type) {
	case map[string]string:
		return v
	case map[string]any:
		for k, val := range v {
			result[k] = fmt.Sprintf("%v", val)
		}
	case Params:
		for k, val := range v {
			result[k] = fmt.Sprintf("%v", val)
		}
	}
	return result
}

// 辅助函数：转换为URL Values
func convertToURLValues(data any) url.Values {
	result := make(url.Values)
	switch v := data.(type) {
	case url.Values:
		return v
	case map[string]string:
		for k, val := range v {
			result.Set(k, val)
		}
	case map[string]any:
		for k, val := range v {
			result.Set(k, fmt.Sprintf("%v", val))
		}
	case Params:
		for k, val := range v {
			result.Set(k, fmt.Sprintf("%v", val))
		}
	}
	return result
}
