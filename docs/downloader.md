#### 下载器
tegenaria 提供了下载器组件接口，允许用户实现自定义的下载器, 接口的定义如下：

```go
// Downloader 下载器接口
type Downloader interface {
	// Download 下载函数
	Download(ctx *Context) (*Response, error)

	// CheckStatus 检查响应状态码的合法性
	CheckStatus(statusCode uint64, allowStatus []uint64) bool
}
```

#### 说明

- ```Download(ctx *Context) (*Response, error)``` 下载器核心逻辑，传入`Context`对象，进行下载操作，并封装`Response`对象作为响应值  

- ```CheckStatus(statusCode uint64, allowStatus []uint64) bool``` 检查响应状态码的合法性

#### 默认的下载器

tegenaria 提供了一个默认的下载器,定义如下:
```go
type SpiderDownloader struct {
	// transport http.Transport 用于设置连接和连接池
	transport *http.Transport
	// client http.Client 网络请求客户端
	client *http.Client
	// ProxyFunc 对单个请求进行代理设置
	ProxyFunc func(req *http.Request) (*url.URL, error)
}
```

- 默认下载器基于golang内置的网络库[net/http](https://pkg.go.dev/net/http)实现

- 参数说明：
    - `transport` `http.Transport`实例对象，包含了连接参数的配置 

    - `client` `http.Client`实例对象，网络请求客户端

    - `ProxyFunc` 代理管理器，负责对代理进行维护和控制  

- 可选参数

    - `DownloaderWithtransport(transport *http.Transport) DownloaderOption` 设置自定义的`transport`对象值  

    - `DownloadWithClient(client http.Client) DownloaderOption` 设置自定义的`client`对象

    - `DownloadWithTimeout(timeout time.Duration) DownloaderOption` 设置超时时间

    - `DownloadWithTLSConfig(tls *tls.Config) DownloaderOption` tls配置

    - `DownloadWithH2(h2 bool) DownloaderOption` 是否启用http2连接

#### 如何使用自定义的downloader

- 实现`Downloader`接口

- 在构建引擎时通过可选参数`EngineWithDownloader(downloader Downloader) EngineOption`传入

- 使用示例:
```go
	engine := NewEngine(EngineWithDownloader(NewDownloader()))
```