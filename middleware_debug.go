package apic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
)

// NewDebugMiddleware 创建新的调试中间件
func NewDebugMiddleware() MiddlewareFunc {
	return func(ctx *Context) {
		// 输出调试开始标记
		logDebugStart()

		// 记录请求
		logRequest(ctx.Request)

		// 调用下一个中间件
		ctx.Next()

		// 记录响应
		if ctx.Response != nil {
			logResponse(ctx)
		}

		// 输出调试结束标记
		logDebugEnd()
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
	fmt.Printf("%s\n", colorize("---------- DEBUG START ----------", ColorCyan))
}

// logDebugEnd 输出调试结束标记
func logDebugEnd() {
	fmt.Printf("%s\n", colorize("---------- DEBUG END ----------", ColorCyan))
	fmt.Println()
}

// logRequest 记录请求详细信息
func logRequest(req *http.Request) {
	// 动态获取HTTP协议版本
	protoVersion := "HTTP/1.1"
	if req.Proto != "" {
		protoVersion = req.Proto
	}

	fmt.Printf("%s\n", colorize(fmt.Sprintf("< %s %s %s", req.Method, req.URL.RequestURI(), protoVersion), ColorCyan))
	fmt.Printf("%s\n", colorize(fmt.Sprintf("< Host: %s", req.URL.Host), ColorCyan))

	// 输出请求头
	for key, values := range req.Header {
		for _, value := range values {
			fmt.Printf("%s\n", colorize(fmt.Sprintf("< %s: %s", key, value), ColorBlue))
		}
	}

	// 输出请求体（如果存在且不是太大）
	if req.Body != nil && req.ContentLength > 0 && req.ContentLength < 1024 {
		// 读取请求体用于调试，但需要重新设置
		body, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			fmt.Printf("%s\n", colorize("< ", ColorCyan))
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
							fmt.Printf("%s\n", colorize(line, ColorYellow))
						}
					}
				} else {
					fmt.Printf("%s\n", colorize(requestBody, ColorYellow))
				}
			}
		}
	}
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

	fmt.Printf("%s\n", colorize(fmt.Sprintf("> %s %d %s", protoVersion, resp.StatusCode, http.StatusText(resp.StatusCode)), statusColor))

	// 输出响应头
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("%s\n", colorize(fmt.Sprintf("> %s: %s", key, value), ColorPurple))
		}
	}

	// 输出空行分隔响应头和响应体
	fmt.Printf("%s\n", colorize("> ", statusColor))

	// 流式响应不读取正文，避免阻塞真正的流处理
	// 优先使用 ApiId.Stream 标志判断，这是最准确的方式
	if ctx.Id != nil && ctx.Id.Stream {
		fmt.Printf("%s\n", colorize("> [streaming body omitted - stream mode enabled]", statusColor))
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
						fmt.Printf("%s\n", colorize(line, ColorGreen))
					}
				}
			} else {
				// 按行打印响应体内容
				lines := strings.Split(string(body), "\n")
				for _, line := range lines {
					if line != "" || len(lines) == 1 {
						fmt.Printf("%s\n", colorize(line, ColorGreen))
					}
				}
			}
		}
		// 重新创建响应体供后续使用
		resp.Body = io.NopCloser(bytes.NewReader(body))
	}
}
