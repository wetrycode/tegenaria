#### Response

```Response```对象主要用于对请求响应进行封装处理，其定义如下：
```go
// Response 请求响应体的结构
type Response struct {
	// Status状态码
	Status int
	// Headers 响应头
	Headers map[string][]string // Header response header
	// Delay 请求延迟
	Delay float64 // Delay the time of handle download request
	// ContentLength 响应体大小
	ContentLength uint64 // ContentLength response content length
	// URL 请求url
	URL string // URL of request url
	// Buffer 响应体缓存
	Buffer *bytes.Buffer // buffer read response buffer

	Body io.ReadCloser

	readLock sync.Mutex
}
```

#### 参数

- `Status` 请求响应状态码  

- `Headers` 响应头  

- `Delay` 从请求发起到请求响应的时间间隔  

- `ContentLength` 响应体大小  

- `URL` 请求的url  

- `Buffer` 响应缓存  

- `Body` 响应读取器  

- `readLock` 数据锁防止buffer被多次读取

#### 方法
`Response`对象了两个方法用于序列化响应  

- ```func (r *Response) Json() (map[string]interface{}, error)``` 序列化为json格式数据  

- ```func (r *Response) String() (string, error)``` 序列化为stringa数据类型