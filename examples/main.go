package main

import (
	"encoding/json"
	"log"
	"net/url"

	"github.com/wonli/apic"
)

func main() {
	musicApi := &apic.ApiId{
		Name:   "music",
		Client: &RandMusicApi{},
	}

	api, err := apic.Init().CallApi(musicApi, nil)
	if err != nil {
		log.Panicln(err.Error())
		return
	}

	var res *ResponseData
	if api.Data != nil {
		err = json.Unmarshal(api.Data, &res)
		if err != nil {
			log.Panicln(err.Error())
		}
	}

	log.Println(res.Data.Name)
	log.Println(res.Data.Url)
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
