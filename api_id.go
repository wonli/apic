package apic

type ApiId struct {
	Name     string
	Client   Api
	Stream   bool
	Request  *RequestData
	Response *ResponseData
}

// Named registers as a named interface.
func (a *ApiId) Named() *ApiId {
	Apis.Named(a)
	return a
}

// Call 简化调用方法，等同于 apic.NewApiClient().Call(id, op)
func (a *ApiId) Call(op *Options) error {
	client := NewApiClient()
	return client.Call(a, op)
}

// CallApi 简化调用方法，等同于 apic.NewApiClient().CallApi(id, op)
func (a *ApiId) CallApi(op *Options) (*ResponseData, error) {
	client := NewApiClient()
	return client.CallApi(a, op)
}

// CallBindJson 简化调用方法，等同于 apic.NewApiClient().CallBindJson(id, resp, op)
func (a *ApiId) CallBindJson(resp any, op *Options) error {
	client := NewApiClient()
	return client.CallBindJson(a, resp, op)
}

// CallFunc 简化调用方法，等同于 apic.NewApiClient().CallFunc(id, op, callback)
func (a *ApiId) CallFunc(op *Options, callback func(a *Api, data []byte) error) error {
	client := NewApiClient()
	return client.CallFunc(a, op, callback)
}
