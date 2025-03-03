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
