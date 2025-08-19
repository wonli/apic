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

func (a *Apic) FormData() any {
	return nil
}

func (a *Apic) WWWFormData() any {
	return nil
}

func (a *Apic) SetData(data *SetDataRequest) error {
	// 这里可以根据需要处理数据设置逻辑
	// 目前保持空实现，确保接口兼容
	return nil
}

func (a *Apic) Setup() (Api, error) {
	return a.Api, nil
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

// 向后兼容的方法实现

// SetJSON 设置JSON数据 - 已废弃
// Deprecated: 使用SetData(NewJSONData(data))替代
func (a *Apic) SetJSON(data interface{}) Api {
	// 转换为新的SetData调用
	a.SetData(NewJSONData(data))
	return a
}

// SetForm 设置表单数据 - 已废弃
// Deprecated: 使用SetData(NewFormData(data))替代
func (a *Apic) SetForm(data interface{}) Api {
	// 转换为新的SetData调用
	if formData, ok := data.(map[string]string); ok {
		a.SetData(NewFormData(formData))
	} else {
		// 尝试转换其他类型
		a.SetData(NewFormData(convertToStringMap(data)))
	}
	return a
}

// SetWWWForm 设置WWW表单数据 - 已废弃
// Deprecated: 使用SetData(NewWWWFormData(data))替代
func (a *Apic) SetWWWForm(data interface{}) Api {
	// 转换为新的SetData调用
	if urlValues, ok := data.(url.Values); ok {
		a.SetData(NewWWWFormData(urlValues))
	} else {
		// 尝试转换其他类型
		a.SetData(NewWWWFormData(convertToURLValues(data)))
	}
	return a
}

// SetQuery 设置查询参数 - 已废弃
// Deprecated: 使用SetData(NewQueryData(data))替代
func (a *Apic) SetQuery(data interface{}) Api {
	// 转换为新的SetData调用
	if urlValues, ok := data.(url.Values); ok {
		a.SetData(NewQueryData(urlValues))
	} else {
		// 尝试转换其他类型
		a.SetData(NewQueryData(convertToURLValues(data)))
	}
	return a
}

// SetHeader 设置请求头 - 已废弃
// Deprecated: 使用SetData(NewHeaderData(data))替代
func (a *Apic) SetHeader(data interface{}) Api {
	// 转换为新的SetData调用
	if headers, ok := data.(map[string]string); ok {
		a.SetData(NewHeaderData(headers))
	} else {
		// 尝试转换其他类型
		a.SetData(NewHeaderData(convertToStringMap(data)))
	}
	return a
}

// SetHeaderRaw 设置原始请求头 - 已废弃
// Deprecated: 使用SetData(NewHeaderData(data))替代
func (a *Apic) SetHeaderRaw(data interface{}) Api {
	// 与SetHeader相同的实现
	return a.SetHeader(data)
}

// NoAutoContentType 禁用自动Content-Type - 已废弃
// Deprecated: 在SetData中设置ContentType
func (a *Apic) NoAutoContentType() Api {
	// 这个方法在新的实现中不再需要，因为Content-Type由SetDataRequest控制
	return a
}

// SetProxy 设置代理 - 已废弃
// Deprecated: 使用ApiClients.WithProxy()替代
func (a *Apic) SetProxy(proxy string) Api {
	// 这个方法在新的实现中应该通过ApiClients.WithProxy()来设置
	// 这里保持空实现以保持兼容性
	return a
}
