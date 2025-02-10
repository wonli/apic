package apic

import (
	"context"
	"encoding/json"
	"log"
	"net/url"
)

// Apic is an empty implementation of the Api interface.
// Introducing this in business logic can avoid writing too much boilerplate code.
type Apic struct {
	Api
	eventCallback func(data string)
}

func (a *Apic) Url() string {
	return ""
}

func (a *Apic) Path() string {
	return ""
}

func (a *Apic) Query() url.Values {
	return nil
}

func (a *Apic) Headers() Params {
	return nil
}

func (a *Apic) PostBody() any {
	return nil
}

func (a *Apic) FormData() Params {
	return nil
}

func (a *Apic) WWWFormData() Params {
	return nil
}

func (a *Apic) Setup(api Api, op *Options) (Api, error) {
	return api, nil
}

func (a *Apic) HttpMethod() HttpMethod {
	return POST
}

func (a *Apic) Debug() bool {
	return false
}

func (a *Apic) UseContext(ctx context.Context) error {
	return nil
}

func (a *Apic) OnRequest() error {
	return nil
}

func (a *Apic) OnResponse(resp []byte) (*ResponseData, error) {
	return &ResponseData{Data: resp}, nil
}

func (a *Apic) OnEvent(callback func(data string)) {
	a.eventCallback = callback
}

func (a *Apic) ReceiveEvent(data string) {
	if a.eventCallback != nil {
		a.eventCallback(data)
	}
}

func (a *Apic) OnHttpStatusError(code int, resp []byte) error {
	return nil
}

// AnyToParams converts any type to Params.
func (a *Apic) AnyToParams(d any) Params {
	dByte, err := json.Marshal(d)
	if err != nil {
		log.Printf("Failed to convert type to byte %s", err.Error())
		return nil
	}

	var p Params
	err = json.Unmarshal(dByte, &p)
	if err != nil {
		log.Printf("Failed to convert to Params %s", err.Error())
		return nil
	}

	return p
}
