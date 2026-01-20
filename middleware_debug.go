package apic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
)

var logConfigOnce sync.Once

// NewDebugMiddleware 创建新的调试中间件
func NewDebugMiddleware() MiddlewareFunc {

	// 只配置一次 log 包（线程安全）
	logConfigOnce.Do(func() {
		log.SetFlags(0)          // 去掉时间戳前缀
		log.SetOutput(os.Stdout) // 输出到标准输出
	})

	return func(ctx *Context) {
		// 输出调试开始标记
		logDebugStart()

		// 记录请求
		logRequest(ctx.Request)

		// 如果是流式请求，启动异步日志处理
		if ctx.Id != nil && ctx.Id.Stream {
			startStreamLogger(ctx)
		}

		// 调用下一个中间件
		ctx.Next()

		// 记录响应
		if ctx.Response != nil {
			logResponse(ctx)
		}

		// 对于流式请求，DEBUG END 会在流式数据完成后打印
		// 对于普通请求，立即打印 DEBUG END
		if ctx.Id == nil || !ctx.Id.Stream {
			logDebugEnd()
		}
	}
}

// 颜色常量
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

// isColorSupported 检查终端是否支持颜色
func isColorSupported() bool {
	// 首先检查COLORTERM环境变量
	colorTerm := os.Getenv("COLORTERM")
	if colorTerm != "" {
		return true
	}

	// 检查TERM环境变量
	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return false
	}

	// 检查是否支持颜色
	return strings.Contains(term, "color") || strings.Contains(term, "xterm")
}

// colorize 为文本添加颜色（如果支持）
func colorize(text, color string) string {
	if isColorSupported() {
		return color + text + ColorReset
	}
	return text
}

// isJSONContent 检查内容类型是否为JSON
func isJSONContent(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "application/json")
}

// formatJSON 格式化JSON字符串
// 当JSON数据过长时，不进行格式化以避免输出过多内容
func formatJSON(data []byte) string {
	const maxJSONLength = 1000
	if len(data) > maxJSONLength {
		return string(data)
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		return string(data)
	}

	return prettyJSON.String()
}

// logDebugStart 输出调试开始标记
func logDebugStart() {
	log.Print(colorize("---------- DEBUG START ----------", ColorCyan))
}

// logDebugEnd 输出调试结束标记
func logDebugEnd() {
	log.Print(colorize("---------- DEBUG END ----------", ColorCyan))
	log.Print("") // 空行
}

// logRequest 记录请求详细信息
func logRequest(req *http.Request) {
	// 动态获取HTTP协议版本
	protoVersion := "HTTP/1.1"
	if req.Proto != "" {
		protoVersion = req.Proto
	}

	log.Print(colorize(fmt.Sprintf("< %s %s %s", req.Method, req.URL.RequestURI(), protoVersion), ColorCyan))
	log.Print(colorize(fmt.Sprintf("< Host: %s", req.URL.Host), ColorCyan))

	// 输出请求头
	for key, values := range req.Header {
		for _, value := range values {
			log.Print(colorize(fmt.Sprintf("< %s: %s", key, value), ColorBlue))
		}
	}

	// 输出请求体（如果存在且不是太大）
	if req.Body != nil && req.ContentLength > 0 && req.ContentLength < 1024 {
		// 读取请求体用于调试，但需要重新设置
		body, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Print(colorize("< ", ColorCyan))
			// 提取请求体部分
			parts := strings.Split(string(body), "\r\n\r\n")
			if len(parts) > 1 {
				requestBody := parts[1]
				// 检查是否为JSON内容并格式化
				contentType := req.Header.Get("Content-Type")
				if isJSONContent(contentType) {
					formattedJSON := formatJSON([]byte(requestBody))
					// 按行输出格式化的JSON
					lines := strings.Split(formattedJSON, "\n")
					for _, line := range lines {
						if line != "" {
							log.Print(colorize(line, ColorYellow))
						}
					}
				} else {
					log.Print(colorize(requestBody, ColorYellow))
				}
			}
		}
	}
}

// startStreamLogger 启动异步流式日志处理器
func startStreamLogger(ctx *Context) {
	// 防止重复启动
	if ctx.StreamLogChan != nil {
		return
	}

	// 创建 buffered channel，避免发送阻塞（缓冲 256 条消息）
	ctx.StreamLogChan = make(chan string, 256)
	ctx.StreamLogDone = make(chan struct{})

	// 启动异步日志处理 goroutine
	go func() {
		// 确保 StreamLogDone 一定会被关闭
		defer close(ctx.StreamLogDone)

		// 使用 recover 防止 panic 导致 goroutine 泄漏
		defer func() {
			if r := recover(); r != nil {
				log.Printf("\n[ERROR] Stream logger panic: %v", r)
			}
		}()

		eventCount := 0
		for line := range ctx.StreamLogChan {
			if line != "" {
				eventCount++
				// 实时打印流式数据
				log.Print(colorize(line, ColorGreen))
			}
		}

		// 所有数据接收完成后，显示统计和结束标记
		if eventCount > 0 {
			log.Print(colorize(fmt.Sprintf("> [Stream completed - received %d events]", eventCount), ColorCyan))
		}

		// 打印调试结束标记
		logDebugEnd()
	}()
}

// logResponse 记录响应详细信息
func logResponse(ctx *Context) {
	resp := ctx.Response
	if resp == nil {
		return
	}

	// 根据状态码选择颜色
	var statusColor string
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		statusColor = ColorGreen
	} else if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		statusColor = ColorYellow
	} else if resp.StatusCode >= 400 {
		statusColor = ColorRed
	} else {
		statusColor = ColorWhite
	}

	// 动态获取HTTP协议版本
	protoVersion := "HTTP/1.1"
	if resp.Proto != "" {
		protoVersion = resp.Proto
	}

	log.Print(colorize(fmt.Sprintf("> %s %d %s", protoVersion, resp.StatusCode, http.StatusText(resp.StatusCode)), statusColor))

	// 输出响应头
	for key, values := range resp.Header {
		for _, value := range values {
			log.Print(colorize(fmt.Sprintf("> %s: %s", key, value), ColorPurple))
		}
	}

	// 输出空行分隔响应头和响应体
	log.Print(colorize("> ", statusColor))

	// 流式响应处理
	if ctx.Id != nil && ctx.Id.Stream {
		// 流式响应的日志通过异步 goroutine 实时打印
		// 这里只显示响应头后的空行
		return
	}

	// 读取并打印响应体
	if resp.Body != nil {
		body, err := io.ReadAll(resp.Body)
		if err == nil && len(body) > 0 {
			// 检查是否为JSON内容并格式化
			contentType := resp.Header.Get("Content-Type")
			if isJSONContent(contentType) {
				formattedJSON := formatJSON(body)
				// 按行输出格式化的JSON
				lines := strings.Split(formattedJSON, "\n")
				for _, line := range lines {
					if line != "" {
						log.Print(colorize(line, ColorGreen))
					}
				}
			} else {
				// 按行打印响应体内容
				lines := strings.Split(string(body), "\n")
				for _, line := range lines {
					if line != "" || len(lines) == 1 {
						log.Print(colorize(line, ColorGreen))
					}
				}
			}
		}
		// 重新创建响应体供后续使用
		resp.Body = io.NopCloser(bytes.NewReader(body))
	}
}
