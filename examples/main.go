package main

import (
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/wonli/apic/v2"
)

func main() {
	musicApi := &apic.ApiId{
		Name: "music",
		Client: &RandMusicApi{
			Apic: &apic.Apic{},
		},
	}

	// 使用中间件的调用链
	api, err := apic.Init().
		// 添加日志中间件
		Use(apic.MiddlewareFunc(func(ctx *apic.Context) {
			// ctx.Next() 前是请求处理
			log.Printf("[MIDDLEWARE] 发送请求: %s %s", ctx.Request.Method, ctx.Request.URL.String())

			// 调用下一个中间件或实际的HTTP请求
			ctx.Next()

			// ctx.Next() 后是响应处理
			if ctx.Response != nil {
				log.Printf("[MIDDLEWARE] 收到响应: %d", ctx.Response.StatusCode)
			}
		})).
		// 添加重试中间件
		Use(apic.NewRetryMiddlewareWithOptions(
			apic.WithMaxRetries(3),                                 // 最大重试3次
			apic.WithInitialDelay(100*time.Millisecond),            // 初始延迟100ms
			apic.WithMaxDelay(5*time.Second),                       // 最大延迟5秒
			apic.WithBackoffMultiplier(2.0),                        // 指数退避倍数2.0
			apic.WithRetryableStatusCodes(500, 502, 503, 504, 302), // 可重试的状态码
			apic.WithOnRetry(func(attempt int, err error, delay time.Duration) {
				log.Printf("[RETRY] 第%d次重试，延迟%v", attempt+1, delay)
			}),
		)).
		CallApi(musicApi, nil)
	if err != nil {
		log.Panicln(err.Error())
		return
	}

	var res *ResponseData
	if api.Data != nil {
		err = json.Unmarshal(api.Data, &res)
		if err != nil {
			log.Printf("JSON解析失败: %s", err.Error())
			return
		}

		log.Println(res.Data.Name)
		log.Println(res.Data.Url)
	} else {
		log.Println("API返回了空响应")
	}
}

type ResponseData struct {
	Code int `json:"code"`
	Data struct {
		Name        string `json:"name"`
		Url         string `json:"url"`
		Picurl      string `json:"picurl"`
		Artistsname string `json:"artistsname"`
	} `json:"data"`
}

type RandMusicApi struct {
	*apic.Apic
}

func (m *RandMusicApi) Url() string {
	return "https://api.uomg.com"
}

func (m *RandMusicApi) Path() string {
	return "/api/visitor.info"
}

func (m *RandMusicApi) Query() url.Values {
	u := url.Values{}

	return u
}

func (m *RandMusicApi) HttpMethod() apic.HttpMethod {
	return apic.GET
}

func (m *RandMusicApi) Debug() bool {
	return true
}
