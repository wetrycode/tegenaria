#### Context

context负责维护单个爬虫请求的声明周期，  
是引擎调度的最小工作单元,其携带了爬虫从请求到解析再到pipeline处理的完整信息,  
其定义如下:
```go
// Context 在引擎中的数据流通载体，负责单个抓取任务的生命周期维护
type Context struct {
	// Request 请求对象
	Request *Request

	// Response 响应对象
	Response *Response

	//parent 父 context
	parent context.Context

	// CtxID context 唯一id由uuid生成
	CtxID string

	// Error 处理过程中的错误信息
	Error error

	// Cancel context.CancelFunc
	Cancel context.CancelFunc

	// Items 读写item的管道
	Items chan *ItemMeta

	// Spider 爬虫实例
	Spider SpiderInterface
}

func NewContext(request *Request, Spider SpiderInterface, opts ...ContextOption) *Context {
	ctx := &Context{}
	parent, cancel := context.WithCancel(context.TODO())
	ctx.Request = request
	ctx.Spider = Spider
	ctx.CtxID = GetUUID()
	ctx.Cancel = cancel
	ctx.Items = make(chan *ItemMeta, 32)
	ctx.parent = parent
	ctx.Error = nil
	log.Infof("Generate a new request%s %s", ctx.CtxID, request.Url)

	for _, o := range opts {
		o(ctx)
	}
	return ctx

}
```

#### 参数  

- `Request` [请求](request.md)对象

- `Response` [请求的响应](response.md)对象

- `parent` 父`Context`对象

- `CtxID` 单个抓取任务的流水号，是`Context`对象整个生命周期的唯一标识,默认采用uuid格式,可以用于跟踪日志  

- `Error` `Context`对象在流转过程中捕获到的错误信息  

- `Cancel` `context.CancelFunc`函数的实例，可以控制任务的状态  

- `Items` 是一个channle,它负责向引擎的pipelines组件提交抓取到的数据字段,该channel会在一个单独的协程中进行处理  

- `spider` SpiderInterface爬虫实例对象  

#### 可选参数

- ```func WithContextID(ctxID string) ContextOption``` 传入用户自定义的流水号  

- ```WithItemChannelSize(size int) ContextOption``` 传入items channel的大小  

#### 示例
```go
	request := NewRequest("http://www.example.com", GET, testParser)

	ctx := NewContext(request, spider1, WithContextID("1234567890"))
```
