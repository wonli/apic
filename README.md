# API调用工具

快速调用第三方接口

### 用法

先定义结构体，在结构体中定义api路径、参数，如果需要鉴权也可以统一处理。

```go
type RandMusicApi struct {
	*apic.Apic
}

func (m *RandMusicApi) Url() string {
	return "https://api.uomg.com"
}

func (m *RandMusicApi) Path() string {
	return "/api/rand.music"
}

func (m *RandMusicApi) Query() url.Values {
	u := url.Values{}
	u.Add("sort", "热歌榜")
	u.Add("format", "json")

	return u
}

func (m *RandMusicApi) HttpMethod() apic.HttpMethod {
	return apic.GET
}

func (m *RandMusicApi) Debug() bool {
	return true
}
```

### 使用方法

使用 `apic.Init().CallApi(musicApi, nil)` 请求接口，可以在请求自定义参数，请求结果在`api`返回数据中处理


```go
musicApi := &apic.ApiId{
    Name:   "music",
    Client: &RandMusicApi{},
}

api, err := apic.Init().CallApi(musicApi, nil)
if err != nil {
    log.Panicln(err.Error())
    return
}
```