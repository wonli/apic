package apic

import (
	"bytes"
	"errors"
	"io"
	"math"
	"net/http"
	"time"
)

// RetryConfig 重试配置
type RetryConfig struct {
	// MaxRetries 最大重试次数（不包括初始请求）
	MaxRetries int
	// InitialDelay 初始延迟时间
	InitialDelay time.Duration
	// MaxDelay 最大延迟时间
	MaxDelay time.Duration
	// BackoffMultiplier 退避倍数
	BackoffMultiplier float64
	// RetryableStatusCodes 可重试的HTTP状态码
	RetryableStatusCodes []int
	// RetryCondition 自定义重试条件函数
	RetryCondition func(*http.Response, error) bool
	// OnRetry 重试时的回调函数
	OnRetry func(attempt int, err error, delay time.Duration)
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:           3,
		InitialDelay:         100 * time.Millisecond,
		MaxDelay:             30 * time.Second,
		BackoffMultiplier:    2.0,
		RetryableStatusCodes: []int{408, 429, 500, 502, 503, 504},
		RetryCondition:       nil,
		OnRetry:              nil,
	}
}

// NewRetryMiddleware 创建重试中间件
func NewRetryMiddleware(config *RetryConfig) MiddlewareFunc {
	if config == nil {
		config = DefaultRetryConfig()
	}
	
	return func(ctx *Context) {
		// 保存原始请求体，用于重试时重新设置
		originalBody, err := saveRequestBody(ctx.Request)
		if err != nil {
			// 如果无法保存请求体，继续执行但不重试
			ctx.Next()
			return
		}
		
		// 保存当前中间件链的剩余部分
		remainingHandlers := ctx.handlers[ctx.index+1:]
		originalIndex := ctx.index
		
		attempt := 0
		for {
			// 恢复请求体（除了第一次）
			if attempt > 0 {
				if err := restoreRequestBody(ctx.Request, originalBody); err != nil {
					// 无法恢复请求体，停止重试
					break
				}
				// 重置上下文状态
				ctx.Response = nil
				ctx.index = originalIndex
			}
			
			// 执行剩余的中间件链
			for i, handler := range remainingHandlers {
				ctx.index = originalIndex + 1 + i
				handler(ctx)
				if ctx.IsAborted() {
					break
				}
			}
			
			// 获取响应
			resp := ctx.Response
			
			// 检查是否需要重试
			shouldRetryResult := shouldRetry(resp, nil, config, attempt)
			if !shouldRetryResult {
				// 不需要重试，退出循环
				break
			}
			
			// 计算延迟时间
			delay := calculateDelay(config, attempt)
			
			// 调用重试回调
			if config.OnRetry != nil {
				config.OnRetry(attempt, nil, delay)
			}
			
			// 等待延迟时间
			time.Sleep(delay)
			
			attempt++
		}
	}
}

// NewRetryMiddlewareWithOptions 使用选项模式创建重试中间件
func NewRetryMiddlewareWithOptions(options ...RetryOption) MiddlewareFunc {
	config := DefaultRetryConfig()
	for _, option := range options {
		option(config)
	}
	return NewRetryMiddleware(config)
}

// RetryOption 重试选项函数类型
type RetryOption func(*RetryConfig)

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(maxRetries int) RetryOption {
	return func(config *RetryConfig) {
		config.MaxRetries = maxRetries
	}
}

// WithInitialDelay 设置初始延迟时间
func WithInitialDelay(delay time.Duration) RetryOption {
	return func(config *RetryConfig) {
		config.InitialDelay = delay
	}
}

// WithMaxDelay 设置最大延迟时间
func WithMaxDelay(delay time.Duration) RetryOption {
	return func(config *RetryConfig) {
		config.MaxDelay = delay
	}
}

// WithBackoffMultiplier 设置退避倍数
func WithBackoffMultiplier(multiplier float64) RetryOption {
	return func(config *RetryConfig) {
		config.BackoffMultiplier = multiplier
	}
}

// WithRetryableStatusCodes 设置可重试的状态码
func WithRetryableStatusCodes(codes ...int) RetryOption {
	return func(config *RetryConfig) {
		config.RetryableStatusCodes = codes
	}
}

// WithRetryCondition 设置自定义重试条件
func WithRetryCondition(condition func(*http.Response, error) bool) RetryOption {
	return func(config *RetryConfig) {
		config.RetryCondition = condition
	}
}

// WithOnRetry 设置重试回调函数
func WithOnRetry(callback func(int, error, time.Duration)) RetryOption {
	return func(config *RetryConfig) {
		config.OnRetry = callback
	}
}

// shouldRetry 判断是否应该重试
func shouldRetry(resp *http.Response, err error, config *RetryConfig, attempt int) bool {
	// 检查是否超过最大重试次数（attempt从0开始，所以要+1）
	if attempt+1 > config.MaxRetries {
		return false
	}
	
	// 如果有自定义重试条件，优先使用
	if config.RetryCondition != nil {
		return config.RetryCondition(resp, err)
	}
	
	// 如果有网络错误，重试
	if err != nil {
		return true
	}
	
	// 如果没有响应，重试
	if resp == nil {
		return true
	}
	
	// 检查状态码是否在可重试列表中
	for _, code := range config.RetryableStatusCodes {
		if resp.StatusCode == code {
			return true
		}
	}
	
	return false
}

// calculateDelay 计算延迟时间（指数退避）
func calculateDelay(config *RetryConfig, attempt int) time.Duration {
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffMultiplier, float64(attempt))
	maxDelay := float64(config.MaxDelay)
	
	if delay > maxDelay {
		delay = maxDelay
	}
	
	return time.Duration(delay)
}

// saveRequestBody 保存请求体内容
func saveRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	
	body, err := req.GetBody()
	if err != nil {
		return nil, errors.New("failed to get request body")
	}
	defer body.Close()
	
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, errors.New("failed to read request body")
	}
	
	return data, nil
}

// restoreRequestBody 恢复请求体内容
func restoreRequestBody(req *http.Request, data []byte) error {
	if data == nil {
		req.Body = nil
		req.ContentLength = 0
		return nil
	}
	
	req.Body = io.NopCloser(bytes.NewReader(data))
	req.ContentLength = int64(len(data))
	
	// 设置GetBody函数，用于后续重试
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	
	return nil
}