# [v0.5.0](https://github.com/wetrycode/tegenaria/compare/v0.4.6...v0.5.0) (2023-03-23)


### Refactor

* 拆分引擎组件，抽离独立的组件接口，提供自定义组件的能力  
* 移除定时轮询接口
* 优化引擎内部对单个请求进行调度的流程  
* 移除`Context`全局管理器，在引擎端引入context计数器
* 移除`Request`和`Context`对象内存池，解决多协程场景下Request复用错误的问题
* `Request`对象的`Parser`类型由`func(resp *Context, req chan<- *Context) error`方法改为`string`方便对`Request`对象进行序列化
* 优化爬虫停止判断策略，增加[组件接口判断逻辑](https://github.com/wetrycode/tegenaria/blob/master/engine.go#L346)，允许用户自定义爬虫终止逻辑  


### Features

* 新增[分布式抓取组件](https://github.com/wetrycode/tegenaria/tree/master/distributed)，提供分布式部署和抓取的能力
* 新增[gRPC和http接口](https://github.com/wetrycode/tegenaria/tree/master/service)，提供实时远程控和查询制爬虫状态的能力
* 引擎内部新增一个[运行时的状态管理器](https://github.com/wetrycode/tegenaria/blob/v0.5.0/stats.go#L50)，用于控制爬虫的启停

* `Request`对象新增一个`DoNotFilter`字段，支持Request粒度下的去重控制
* `Request`对象新增方法`ToMap() (map[string]interface{}, error)`,用于将`Request`对象进行序列化
* `Request`对象新增方法`RequestFromMap(src map[string]interface{}, opts ...RequestOption) *Request`,用于将`map[string]interface{}`对象进行反序列化为`Request`对象
* `Request`新增初始化选项`RequestWithPostForm(payload url.Values) RequestOption`用于接收`application/x-www-form-urlencoded`参数
* `Request`新增初始化选项`RequestWithBodyReader(body io.Reader) RequestOption`用于从内存读取二进制数据
* `Request`新增初始化选项`RRequestWithDoNotFilter(doNotFilter bool) RequestOption`用于控制`Request`对象是否参与去重
* `Response`新增一个接口`WriteTo(writer io.Writer) (int64, error)`,允许用户将response写入自定义的`io.Writer`,例如一个本地文件io实例，实现文件下载

* 新增`ComponentInterface`[组件接口](https://github.com/wetrycode/tegenaria/blob/v0.5.0/components.go)，允许用户自定义组件

* 新增`DistributedWorkerInterface`[分布式节点控制接口](https://github.com/wetrycode/tegenaria/blob/v0.5.0/distributed.go)允许用户实现自定义对节点的控制逻辑

### Style

* 重命名`Request`和`Response`的`Header`为`Headers`

* 重命名`CacheInterface`接口的所有方法以大写字母开头  