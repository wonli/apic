package apic

import (
	"io"
	"net/url"
)

// DataType 定义请求数据类型
type DataType int

const (
	DataTypeJSON DataType = iota
	DataTypeForm
	DataTypeWWWForm
	DataTypeQuery
	DataTypeHeader
	DataTypeRaw
	DataTypeXML
	DataTypeText
)

// SetDataRequest 统一的数据设置请求结构
type SetDataRequest struct {
	Type        DataType          `json:"type"`
	Content     any               `json:"content"`
	Headers     map[string]string `json:"headers,omitempty"`      // 额外的请求头
	ContentType string            `json:"content_type,omitempty"` // 自定义Content-Type
	Encoding    string            `json:"encoding,omitempty"`     // 编码方式
}

// NewSetDataRequest 创建新的数据设置请求
func NewSetDataRequest(dataType DataType, content any) *SetDataRequest {
	return &SetDataRequest{
		Type:    dataType,
		Content: content,
		Headers: make(map[string]string),
	}
}

// WithHeaders 添加请求头
func (rd *SetDataRequest) WithHeaders(headers map[string]string) *SetDataRequest {
	for k, v := range headers {
		rd.Headers[k] = v
	}
	return rd
}

// WithContentType 设置Content-Type
func (rd *SetDataRequest) WithContentType(contentType string) *SetDataRequest {
	rd.ContentType = contentType
	return rd
}

// WithEncoding 设置编码
func (rd *SetDataRequest) WithEncoding(encoding string) *SetDataRequest {
	rd.Encoding = encoding
	return rd
}

// 便捷构造函数

// NewJSONData 创建JSON数据
func NewJSONData(data any) *SetDataRequest {
	return NewSetDataRequest(DataTypeJSON, data)
}

// NewFormData 创建Form数据
func NewFormData(data map[string]string) *SetDataRequest {
	return NewSetDataRequest(DataTypeForm, data)
}

// NewWWWFormData 创建WWW Form数据
func NewWWWFormData(data url.Values) *SetDataRequest {
	return NewSetDataRequest(DataTypeWWWForm, data)
}

// NewQueryData 创建Query参数
func NewQueryData(data url.Values) *SetDataRequest {
	return NewSetDataRequest(DataTypeQuery, data)
}

// NewRawData 创建原始数据
func NewRawData(data []byte) *SetDataRequest {
	return NewSetDataRequest(DataTypeRaw, data)
}

// NewReaderData 创建Reader数据
func NewReaderData(reader io.Reader) *SetDataRequest {
	return NewSetDataRequest(DataTypeRaw, reader)
}

// NewXMLData 创建XML数据
func NewXMLData(data any) *SetDataRequest {
	return NewSetDataRequest(DataTypeXML, data)
}

// NewTextData 创建文本数据
func NewTextData(text string) *SetDataRequest {
	return NewSetDataRequest(DataTypeText, text)
}

// NewHeaderData 创建请求头数据
func NewHeaderData(headers map[string]string) *SetDataRequest {
	return NewSetDataRequest(DataTypeHeader, headers)
}
