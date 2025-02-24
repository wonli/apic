package apic

import "net/url"

// Options request options
type Options struct {
	Query        url.Values
	PostBody     Params
	Headers      Params
	Setup        Params
	UseRawHeader bool // 是否使用原始头
	Debug        bool // 是否开启debug
}
