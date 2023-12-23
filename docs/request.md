#### Request

`Request`接口主要用户构建请求对象，负责承载请求参数和数据，包括 URL、请求方式、对应的解析函数名及代理等参数.

```go
// Request 请求对象的结构
type Request struct {
	// Url 请求Url
	Url string `json:"url"`
	// Headers 请求头
	Headers map[string]string `json:"header"`
	// Method 请求方式
	Method RequestMethod `json:"method"`
	// Body 请求body
	body []byte `json:"-"`
	// Params 请求url的参数
	Params map[string]string `json:"params"`
	// Proxy 代理实例
	Proxy *Proxy `json:"-"`
	// Cookies 请求携带的cookies
	Cookies map[string]string `json:"cookies"`
	// Meta 请求携带的额外的信息
	Meta map[string]interface{} `json:"meta"`
	// AllowRedirects 是否允许跳转默认允许
	AllowRedirects bool `json:"allowRedirects"`
	// MaxRedirects 最大的跳转次数
	MaxRedirects int `json:"maxRedirects"`
	// Parser 该请求绑定的响应解析函数，必须是一个spider实例
	Parser string `json:"parser"`
	// MaxConnsPerHost 单个域名最大的连接数
	MaxConnsPerHost int `json:"maxConnsPerHost"`
	// BodyReader 用于读取body
	bodyReader io.Reader `json:"-"`
	// AllowStatusCode 允许的状态码
	AllowStatusCode []uint64 `json:"allowStatusCode"`
	// Timeout 请求超时时间
	Timeout time.Duration `json:"timeout"`
	// DoNotFilter
	DoNotFilter bool
}

// 请注意parser函数必须是某一个spiderinterface实例的解析函数
// 否则无法正常调用该解析函数
func NewRequest(url string, method RequestMethod, parser Parser, opts ...RequestOption) *Request {
	request := &Request{
		Url:             url,
		Method:          method,
		Parser:          GetFunctionName(parser),
		Headers:          make(map[string]string),
		Meta:            make(map[string]interface{}),
		Cookies: make(map[string]string),
		DoNotFilter:     false,
		AllowRedirects:  true,
		MaxRedirects:    3,
		AllowStatusCode: make([]uint64, 0),
		Timeout:         -1 * time.Second,
		bodyReader:      nil,
	}
	for _, o := range opts {
		o(request)
	}
	request.updateQueryParams()
	return request

}
```

#### 参数说明

- `Url` 完整的请求链接

- `Method` 请求方式

  - `GET`

  - `POST`

  - `PUT`

  - `DELETE`

  - `OPTIONS`

  - `HEAD`

- `Parser` 请求响应解析函数，请注意该函数必须是注册到引擎的 spider 实例的解析函数  
  `func(resp *Context, req chan<- *Context) error`

- `Body` 请求体,支持的加载方式如下

  - `RequestWithRequestBody(body map[string]interface{}) RequestOption` 加载 map[string]interface{}

  - `func RequestWithRequestBytesBody(body []byte) RequestOption` 直接加载 bytes 二进制

  - `func RequestWithBodyReader(body io.Reader) RequestOption` 可以用于上传文件

  - `func RequestWithPostForm(payload url.Values) RequestOption` 加载`application/x-www-form-urlencoded`参数

- `Headers` 请求头在构建请求对象时传入可选参数加载  
  `func RequestWithRequestHeader(headers map[string]string) RequestOption`

- `Meta` 额外的请求参数，通过可选参数传入  
  `func RequestWithRequestMeta(meta map[string]interface{}) RequestOption`

- `AllowRedirects` 是否运行链接跳转,默认允许，关联可选参数  
  `func RequestWithAllowRedirects(allowRedirects bool) RequestOption`

- `MaxRedirects` 最大跳转次数,默认值为 3,若设置为 0 则不允许跳转,关联可选参数  
  `func RequestWithMaxRedirects(maxRedirects int) RequestOption`

- `AllowStatusCode` 合法响应码列表，默认为空并按照 http 协议标准状态进行过滤,关联可选参数  
  `func RequestWithAllowedStatusCode(allowStatusCode []uint64) RequestOption`

- `Timeout` 超时时间，默认不设置，关联可选参数  
  `func RequestWithTimeout(timeout time.Duration) RequestOption`

- `DoNotFilter` 不参与去重处理，默认为 false,关联可选参数  
  `func RequestWithDoNotFilter(doNotFilter bool) RequestOption`

- `MaxConnsPerHost` 单个域名最大的连接数,关联的可选参数  
  `func RequestWithMaxConnsPerHost(maxConnsPerHost int) RequestOption`

- `Cookies` 请求携带的cookies,关联的可选参数  
`func RequestWithRequestCookies(cookies map[string]string) RequestOption`  

#### 方法
```Request```对象包含了两个方法对其进行结构化处理  

- ```func (r *Request) ToMap() (map[string]interface{}, error)``` 将```Request```对象转换成```map[string]interface{}```  

- ```func RequestFromMap(src map[string]interface{}, opts ...RequestOption) *Request```基于```map[string]interface{}```和可选参数构建新的```Request```对象

#### 示例

```go

	body := make(map[string]interface{})
	body["test"] = "test"
	meta := make(map[string]interface{})
	meta["key"] = "value"
	header := make(map[string]string)
	header["Content-Type"] = "application/json"
	requestOptions := []tegenaria.RequestOption{}
	// 添加请求的可选参数
	requestOptions = append(requestOptions, tegenaria.RequestWithRequestHeaders(header))
	requestOptions = append(requestOptions, tegenaria.RequestWithRequestBody(body))
	requestOptions = append(requestOptions, tegenaria.RequestWithRequestProxy(proxy))
	requestOptions = append(requestOptions, tegenaria.RequestWithRequestMeta(meta))
	requestOptions = append(requestOptions, tegenaria.RequestWithMaxConnsPerHost(16))
	requestOptions = append(requestOptions, tegenaria.RequestWithMaxRedirects(-1))

	urlReq := fmt.Sprintf("%s/testPOST", tServer.URL)
	request := tegenaria.NewRequest(urlReq, tegenaria.POST, spider1.Parser, requestOptions...)
```