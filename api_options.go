package apic

import (
	"net/url"
	"time"
)

// Options request options
type Options struct {
	Query        url.Values
	PostBody     Params
	Headers      Params
	Setup        Params
	UseRawHeader bool          // 是否使用原始头
	Debug        bool          // 是否开启debug
	Timeout      time.Duration // 请求超时时间，如果为0则使用默认值30秒
}
