package main

import (
	"encoding/json"
	"log"
	"net/url"

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
