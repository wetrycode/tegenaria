# Tegenaria文档
* [Tegenaria文档](#tegenaria文档)
* [1.设计思想](#1设计思想)
* [2.数据请求处理流程图](#2数据请求处理流程图)
   * [2.1.流程说明](#21流程说明)
      * [2.1.1.主要流程](#211主要流程)
      * [2.1.2其他流程](#212其他流程)
   * [2.2.调度模式](#22调度模式)
* [3.组件](#3组件)
   * [3.1.引擎](#31引擎)
   * [3.2.调度器](#32调度器)
   * [3.3.下载器](#33下载器)
   * [3.4.去重处理器](#34去重处理器)
   * [3.5.下载中间件](#35下载中间件)
   * [3.6.管道](#36管道)
   * [3.7.错误处理器](#37错误处理器)
* [4.API](#4api)
   * [4.1.SpiderInterface](#41spiderinterface)
      * [4.1.1.Spider Interface说明](#411spider-interface说明)
      * [4.1.2.实例化Spider](#412实例化spider)
      * [4.1.3.引擎spider启动](#413引擎spider启动)
      * [4.2.Context](#42context)
   * [4.3.Request](#43request)
   * [4.4.RFPDupeFilterInterface](#44rfpdupefilterinterface)
   * [4.5.Response](#45response)
   * [4.6.MiddlerwareInterface](#46middlerwareinterface)
      * [4.6.1.中间件接口说明](#461中间件接口说明)
      * [4.6.2.中间件实例化](#462中间件实例化)
      * [4.6.3.中间件注册](#463中间件注册)
      * [4.6.4.ProcessRequest调度](#464processrequest调度)
      * [4.6.5.ProcessResponse](#465processresponse)
   * [4.7.Downloader](#47downloader)
      * [4.7.1.Tegenaria下载器接口](#471tegenaria下载器接口)
      * [4.7.2.下载器调度](#472下载器调度)
   * [4.8.ItemMeta](#48itemmeta)
   * [4.9.PipelineInterface](#49pipelineinterface)
      * [4.9.1.pipeline接口说明](#491pipeline接口说明)
      * [4.9.2.pipeline注册](#492pipeline注册)
      * [4.9.3.pipeline实例化示例](#493pipeline实例化示例)

# 1.设计思想
本项目在模块功能和数据处理流程方面借鉴了[scrapy](https://github.com/scrapy/scrapy)的设计思想，实现下载、调度、解析及数据处理各模块间解耦。
* 模块间的数据交互基于channel实现。
* 在系统调度方面，不同请求间采用异步处理，单个请求的流程采用同步方式。
* 提供统一的pipelines和middlerwares接口作为业务功能扩展的入口
* 充分利用golang的多核处理能力
* 以引擎和调度器为核心，实现数据扇入扇出

# 2.数据请求处理流程图
![image](images/scheduler.png)

## 2.1.流程说明
### 2.1.1.主要流程
* Spider 通过StartRequest 方法构建种子请求(Context)，并通过Request channel发送到调度器  
* 请求对象会先写到缓存队列(默认使用内存)中 ,在此之前引擎根据设置启用或跳过request去重器
* 若启用request去重器，其会计算request指纹并放入布隆过滤器进行去重处理
* 若是重复的请求且设置引擎不允许重复请求发送，则忽略该请求否则将reuquest写入缓存
* 调度器从缓存中读取Request并通过cache channel发送到下载处理器
* 下载处理器在正式下载之前会按优先级调用下载中间件的ProcessRequest方法  
* 下载器在处理请求结束后会生成RequestResult，包含HandleError和Response,随后请求结果会通过RequestResult channel发送到调度器  
* 调度器接会为每一个接收到的RequestResult启用一个下载结果处理器协程进行处理将HandleError发送到error channel,将非空Response发送到response channel
* 调度器会为每一个通过response channel接收的response启用一个解析器处理协程，在正式对response进行解析之前会按照下载中间件的优先级由低到高执行ProcessResponse,用于处理response,解析器会回调spider的Parser函数
* 解析器在解析过程中实时通过items channel发送Item到调度器，也可以实时发送新的Request到调度器
* 调度器为每一个接收到的item启用一个item处理器协程，在处理item过程中会按照优先级执行Pipelines的ProcessItem方法
* item处理完成后一个完成整的数据请求处理也流程就完成了

### 2.1.2其他流程
* 调度器为接收到的每一条HandleError启用一个错误处理器协程，此处会回调spider定义的ErrorHandler并且可以生成新的request发送到调度器
* 各阶段各处理器产生的HandleError都会通过error channel发送到调度器等待调度处理

## 2.2.调度模式
Tegenaria采用的是异步调度模式，不同流程的不同模块间搞channel通信和传递数据

# 3.组件
## 3.1.引擎

* 引擎是整个Tegenaria框架的核心组件，负责spider启停、数据流处理调度及一些数据统计。
* 引擎内置了整个系统调度过程所需channel
* 引擎为每一个独立的处理器启用单独的协程，用来处理异步任务
* 调度器是内置于引擎且可以根据设置启用指定数量的调度器

```go
// EngineWithSchedulerNum set engine scheduler number
// default to cpu number
func EngineWithSchedulerNum(schedulerNum uint) EngineOption {
	return func(r *SpiderEngine) {
		r.schedulerNum = schedulerNum
		engineLog.Infoln("Set engine scheduler number to ", schedulerNum)

	}
}
```
```go
	// start schedulers
	for i := 0; i < int(e.schedulerNum); i++ {
		e.mainWaitGroup.Add(1)
		go e.engineScheduler(spider)
	}
```
* 缓存读取器内置于引擎可以根据设置启用指定数量的缓存读取器

```go
// EngineWithReadCacheNum set cache reader number
func EngineWithReadCacheNum(cacheReadNum uint) EngineOption {
	return func(r *SpiderEngine) {
		r.cacheReadNum = cacheReadNum
		engineLog.Infoln("Set engine cache reader to ", cacheReadNum)

	}
}
```
```go
	for n := 0; n < int(e.cacheReadNum); n++ {
		e.mainWaitGroup.Add(1)
		// read request from cache and send to cacheChan
		go e.readCache()
	}
```
## 3.2.调度器
调度器的核心任务是接收各个通道的数据并将接收到的数据分发到与之相对应的处理器，并启用独立的协程进行处理。

```go
	for {
		if e.isRunning {
			// 避免cache队列关闭后无法退出
			select {
			case req, ok := <-e.cacheChan:
				if ok {
					e.waitGroup.Add(1)
					go e.recvRequestHandler(req)
				}

			default:
			}
			select {
			case req := <-e.requestsChan:
				// write request to cache
				e.waitGroup.Add(1)
				go e.writeCache(req)
			case requestResult := <-e.requestResultChan:
				// handle request download result
				e.waitGroup.Add(1)
				go e.doRequestResult(requestResult)
			case response := <-e.respChan:
				// handle request response
				e.waitGroup.Add(1)
				go e.doParse(spider, response)
			case item := <-e.itemsChan:
				// handle scape items
				e.waitGroup.Add(1)
				go e.doPipelinesHandlers(spider, item)
			case err := <-e.errorChan:
				// handle error
				e.waitGroup.Add(1)
				go e.doError(spider, err)
			case <-time.After(time.Second * 3):
				if e.checkReadyDone() {
					e.isDone = true
					break Loop
				}
			}
		}

	}
```
## 3.3.下载器
[Tegenaria下载器](https://github.com/wetrycode/tegenaria/blob/master/downloader.go)的网络请求模块基于[net/http](https://pkg.go.dev/net/http)实现，负责处理所有的网络请求，Tegenaria提供了一个通用的下载器接口，供业务自己实现下载器

```go
// Downloader interface
type Downloader interface {
	// Download core funcation
	Download(ctx *Context, result chan<- *Context)

	// CheckStatus check response status code if allow handle
	CheckStatus(statusCode uint64, allowStatus []uint64) bool
	
	// setTimeout set downloader timeout
	setTimeout(timeout time.Duration)
}
```
## 3.4.去重处理器
Tegenaria提供了一个基于布隆过滤器实现的默认的request去重处理器。其主要的实现逻辑如下：

* 将url规范化处理

```go
// canonicalizeUrl canonical request url before calculate request fingerprint
func (f *RFPDupeFilter) canonicalizetionUrl(request *Request, keepFragment bool) url.URL {
	u, _ := url.ParseRequestURI(request.Url)
	u.RawQuery = u.Query().Encode()
	u.ForceQuery = true
	if !keepFragment {
		u.Fragment = ""
	}
	return *u
}
```
* 将请求头重新编码处理

```go
// encodeHeader encode request header before calculate request fingerprint
func (f *RFPDupeFilter) encodeHeader(request *Request) string {
	h := request.Header
	if h == nil {
		return ""
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	// Sort by Header key
	sort.Strings(keys)
	for _, k := range keys {
		// Sort by value
		buf.WriteString(fmt.Sprintf("%s:%s;\n", strings.ToUpper(k), strings.ToUpper(h[k])))
	}
	return buf.String()
}
```
* 基于[第三方sha128计算库获取](https://github.com/spaolacci/murmur3)指纹

```go
func (f *RFPDupeFilter) Fingerprint(request *Request) ([]byte, error) {
	// get sha128
	sha := murmur3.New128()
	_, err := io.WriteString(sha, request.Method)
	if err != nil {
		return nil, err
	}
	// canonical request url
	u := f.canonicalizetionUrl(request, false)
	_, err = io.WriteString(sha, u.String())
	if err != nil {
		return nil, err
	}
	// get request body
	if request.Body != nil {
		body := request.Body
		sha.Write(body)
	}
	// to handle request header
	if len(request.Header) != 0 {
		_, err := io.WriteString(sha, f.encodeHeader(request))
		if err != nil {
			return nil, err
		}
	}
	res := sha.Sum(nil)
	return res, nil
}
```
* 去重处理

```go
// DoDupeFilter deduplicate request filter by bloom
func (f *RFPDupeFilter) DoDupeFilter(request *Request) (bool, error) {
	// Use bloom filter to do fingerprint deduplication
	data, err := f.Fingerprint(request)
	if err != nil {
		return false, err
	}
	return f.bloomFilter.TestOrAdd(data), nil
}
```
## 3.5.下载中间件
Tegenaria提供了下载中间件接口用于实现各类下载中间件,其主要功能如下：

* 在进入下载器前按优先级**由高到低**对request进行统一的处理，例如添加网络代理和User-Agent
* 在进入解析器之前按**由低到高**的优先级对response进行统一的处理，例如验证响应体是否合法，在处理的同时也可以发送新的请求到 request channel
* 引擎提供了一个下载中间件注册接口用于向引擎注册下载中间件

```go
// RegisterDownloadMiddlewares add a download middlewares
func (e *SpiderEngine) RegisterDownloadMiddlewares(middlewares MiddlewaresInterface) {
	e.downloaderMiddlewares = append(e.downloaderMiddlewares, middlewares)
	sort.Sort(e.downloaderMiddlewares)
}
```
## 3.6.管道
Tegenaria提供了pipelines接口用于实现各类管道,其主要功能是按优先级对解析到的item进行处理
## 3.7.错误处理器
用于接收由调度器发到HandleError channel的HandleError数据，此处会回调spider定义的ErrorHandler方法

```go
// doError handle all error which is from errorChan
func (e *SpiderEngine) doError(spider SpiderInterface, err *HandleError) {
	atomic.AddUint64(&e.Stats.ErrorCount, 1)
	e.ErrorHandler(spider, err)
	spider.ErrorHandler(err, e.requestsChan)
	if err.Request != nil {
               // release Request
		freeRequest(err.Request)
	}
	if err.Response != nil {
               // release Response
		freeResponse(err.Response)
	}
	e.waitGroup.Done()
}
```
# 4.API
## 4.1.SpiderInterface
### 4.1.1.Spider Interface说明
Tegenaria spider接口，业务侧自定义的spider必须基于此接口实现业务spider.

```go
type SpiderInterface interface {
	// StartRequest make new request by feed urls
	StartRequest(req chan<- *Context)

	// Parser parse response ,it can generate ItemMeta and send to engine
	// it also can generate new Request
	Parser(resp *Context, item chan<- *ItemMeta, req chan<- *Context) error

	// ErrorHandler it is used to handler all error recive from engine
	ErrorHandler(err *HandleError, req chan<- *Context)

	// GetName get spider name
	GetName() string
}
```
* StartRequest,用于构建种子urls 的起始请求，将起始请求通过管道发送到引擎调度器
* Parser，用于解析response并生成item或request,将生成的数据通过指定的管道发送到引擎调度器
* ErrorHandler，处理捕获到的错误，由引擎回调执行
* GetName,获取spider name
* Tegenaria 提供了一个BaseSpider供业务侧使用

```go
// BaseSpider base spider
type BaseSpider struct {
	// Name spider name
	Name     string

	// FeedUrls feed urls
	FeedUrls []string
}
```
### 4.1.2.实例化Spider
```go
type TestSpider struct {
	*BaseSpider
}

func (s *TestSpider) StartRequest(req chan<- *Context) {
	for _, url := range s.FeedUrls {
		request := NewRequest(url, GET, s.Parser)
		ctx := NewContext(request)
		req <- ctx
	}
}
func (s *TestSpider) Parser(resp *Context, item chan<- *ItemMeta, req chan<- *Context) error {
	return nil
}
func (s *TestSpider) ErrorHandler(err *HandleError, req chan<- *Context){

}
func (s *TestSpider) GetName() string {
	return s.Name
}
```
### 4.1.3.引擎spider启动
```go
// StartSpiders start a spider specify by spider name
func (e *SpiderEngine) startSpiders(spiderName string) {
	spider := e.spiders.SpidersModules[spiderName]
	defer func() {
		e.startRequestFinish = true
		e.waitGroup.Done()
	}()
	e.isRunning = true

	spider.StartRequest(e.requestsChan)
}

// run Spiders StartRequest function and get feeds request
go e.startSpiders(spiderName)
```


### 4.2.Context
* Context 是Tegenaria中除Item之外的最小的调度单元，负责维护在进入pipelines流程之前的数据处理流程的生命周期。

```go
type Context struct {
	// Request
	Request *Request

	// DownloadResult downloader handler result
	DownloadResult *RequestResult

	// Item
	Item ItemInterface

	//Ctx parent context
	Ctx context.Context

	// CtxId
	CtxId string

	// Error
	Error error
}
```
* CtxId是每一条数据处理过程中的唯一流水号，在请求初始化阶段生成，是一个uuid格式的字符串
* 创建一个新的Context

```go
	request := NewRequest(server.URL+"/testGET", GET, testParser)
	var MainCtx context.Context = context.Background()
	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))

	defer func() {
		cancel()
	}()
```


## 4.3.Request
网络请求参数配置接口

```go
// Request a spider request config
type Request struct {
	Url     string            // Set request URL
	Header  map[string]string // Set request header
	Method  string            // Set request Method
	Body    []byte            // Set request body
	Params  map[string]string // Set request query params
	Proxy   *Proxy            // Set request proxy addr
	Cookies map[string]string // Set request cookie
	// Timeout         time.Duration          // Set request timeout
	// TLS             bool                   // Set https request if skip tls check
	Meta            map[string]interface{} // Set other data
	AllowRedirects  bool                   // Set if allow redirects. default is true
	MaxRedirects    int                    // Set max allow redirects number
	parser          Parser                 // Set response parser funcation
	maxConnsPerHost int                    // Set max connect number for per host
	BodyReader      io.Reader              // Set request body reader
	ResponseWriter  io.Writer              // Set request response body writer,like file
	// RequestId       string                 // Set request uuid
}
```
* 网络请求处理对象，使用方法如下，必须提供URL和请求方法以及response解析方法

```go
request := NewRequest(server.URL+"/testGET", GET, testParser)
```
* 解析方法类型定义如下，解析函数生成item或新的request通过指定的通道发送到引擎调度器供后续流程使用

```go
// Parser response parse handler
type Parser func(resp *Context, item chan<- *ItemMeta, req chan<- *Context) error
```


## 4.4.RFPDupeFilterInterface
* request指纹计算及去重组件接口

```go
// RFPDupeFilterInterface Request Fingerprint duplicates filter interface
type RFPDupeFilterInterface interface {
	// Fingerprint compute request fingerprint
	Fingerprint(request *Request) ([]byte, error)

	// DoDupeFilter do request fingerprint duplicates filter
	DoDupeFilter(request *Request) (bool, error)
}
```
* Tegenaria 提供的默认去重组件基于布隆过滤器实现，指纹计算基于sha128计算
* Fingerprint 是指纹计算的核心代码
* DoDupeFilter用于去重处理
* Tegenaria自带的去重处理器使用方法如下：

```go
	server := newTestServer()
	// downloader := NewDownloader()
	headers := map[string]string{
		"Params1":    "params1",
		"Intparams":  "1",
		"Boolparams": "false",
	}
	request1 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers))
	request2 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers))
	request3 := NewRequest(server.URL+"/testHeader2", GET, testParser, RequestWithRequestHeader(headers))

	duplicates := NewRFPDupeFilter(1024*1024, 5)
	if r1, _ := duplicates.DoDupeFilter(request1); r1 {
		t.Errorf("Request1 igerprint sum error expected=%v, get=%v", false, true)
	}
	if r2, _ := duplicates.DoDupeFilter(request2); !r2 {
		t.Errorf("Request2 igerprint sum error expected=%v, get=%v", true, false)

	}
	if r3, _ := duplicates.DoDupeFilter(request3); r3 {
		t.Errorf("Request3 igerprint sum error expected=%v, get=%v", false, true)

	}
```
* 引擎启用去重处理器

```go
// doFilter filer duplicate request if filterDuplicateReq is true
func (e *SpiderEngine) doFilter(ctx *Context, r *Request) bool {
	// filter switch
	if e.filterDuplicateReq {
		// do filter
		result, err := e.RFPDupeFilter.DoDupeFilter(r)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Warningf("Request do unique error %s", err.Error())
			e.errorChan <- NewError(ctx.CtxId, fmt.Errorf("Request do unique error %s", err.Error()), ErrorWithRequest(ctx.Request))
		}
		if result {
			engineLog.WithField("request_id", ctx.CtxId).Debugf("Request is not unique")
		}
		return !result
	}
	return true
}

if e.doFilter(ctx, ctx.Request) && !e.isDone {
		err := e.cache.enqueue(ctx)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("Push request to cache queue error %s", err.Error())
		}
	}
```
## 4.5.Response
* 网络请求响应对象

```go
// Response the Request download response data
type Response struct {
	Status int // Status request response status code
	Header map[string][]string // Header response header
	Delay         float64       // Delay the time of handle download request
	ContentLength int           // ContentLength response content length
	URL           string        // URL of request url
	Buffer        *bytes.Buffer // buffer read response buffer
}
```
* 获取JSON数据

```go
// Json deserialize the response body to json
func (r *Response) Json() map[string]interface{} {
	defer func() {
		if p := recover(); p != nil {
			respLog.Errorf("panic recover! p: %v", p)
		}

	}()
	jsonResp := map[string]interface{}{}
	err := json.Unmarshal(r.Buffer.Bytes(), &jsonResp)
	if err != nil {
		respLog.Errorf("Get json response error %s", err.Error())
	}
	return jsonResp
}
```
* 获取string

```go
// String get response text from response body
func (r *Response) String() string {
	defer func() {
		if p := recover(); p != nil {
			respLog.Errorf("panic recover! p: %v", p)
		}

	}()
	return r.Buffer.String()
}
```
## 4.6.MiddlerwareInterface
### 4.6.1.中间件接口说明
下载中间件接口，用于处理Request和Response，优先级数字越小优先级越高

```go
type MiddlewaresInterface interface {
	GetPriority() int
	ProcessRequest(ctx *Context) error
	ProcessResponse(ctx *Context, req chan<- *Context) error
	GetName()string
}
```
* GetPriority获取中间件的优先级
* ProcessRequest在对Request进行下载处理前对其进行修饰处理，添加代理或User-Agent
* ProcessResponse在Response进入解析器前对response进行处理
* GetName获取中间件名称
* 优先级排序由[sort接口](https://pkg.go.dev/sort)实现

```go
func (p Middlewares) Len() int           { return len(p) }
func (p Middlewares) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Middlewares) Less(i, j int) bool { return p[i].GetPriority() < p[j].GetPriority() }
```
* 下载中间件队列定义

```go
type Middlewares []MiddlewaresInterface
```
### 4.6.2.中间件实例化
```go
type TestDownloadMiddler struct {
	Priority int
	Name     string
}

func (m TestDownloadMiddler) GetPriority() int {
	return m.Priority
}
func (m TestDownloadMiddler) ProcessRequest(ctx *Context) error {
	header := fmt.Sprintf("priority-%d", m.Priority)
	ctx.Request.Header[header] = strconv.Itoa(m.Priority)
	return nil
}

func (m TestDownloadMiddler) ProcessResponse(ctx *Context, req chan<- *Context) error {
	return nil

}
```
### 4.6.3.中间件注册
中间件注册由引擎实现

```go
// RegisterDownloadMiddlewares add a download middlewares
func (e *SpiderEngine) RegisterDownloadMiddlewares(middlewares MiddlewaresInterface) {
	e.downloaderMiddlewares = append(e.downloaderMiddlewares, middlewares)
	sort.Sort(e.downloaderMiddlewares)
}
```
### 4.6.4.ProcessRequest调度
按优先级由高到低调度

```go
	// use download middleware to handle request object
	for _, middleware := range e.downloaderMiddlewares {
		err := middleware.ProcessRequest(ctx)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("Middleware %s handle request error %s", middleware.GetName(), err.Error())
			ctx.Error = err
			e.errorChan <- NewError(ctx.CtxId, err, ErrorWithRequest(ctx.Request))
			return
		}
	}
```
### 4.6.5.ProcessResponse
按优先级由低到高调度

```go
// processResponse do handle download response
func (e *SpiderEngine) processResponse(ctx *Context) {
	if len(e.downloaderMiddlewares) == 0 {
		return
	}
	for index := range e.downloaderMiddlewares {
		middleware := e.downloaderMiddlewares[len(e.downloaderMiddlewares)-index-1]
		err := middleware.ProcessResponse(ctx, e.requestsChan)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("Middleware %s handle response error %s", middleware.GetName(), err.Error())
			ctx.Error = err
			e.errorChan <- NewError(ctx.CtxId, err, ErrorWithRequest(ctx.Request), ErrorWithResponse(ctx.DownloadResult.Response))
			return
		}
	}
}
```
## 4.7.Downloader
### 4.7.1.Tegenaria下载器接口
```go
// Downloader interface
type Downloader interface {
	// Download core funcation
	Download(ctx *Context, result chan<- *Context)

	// CheckStatus check response status code if allow handle
	CheckStatus(statusCode uint64, allowStatus []uint64) bool
	
	// setTimeout set downloader timeout
	setTimeout(timeout time.Duration)
}
```
* Download载器核心的网络请求处理函数，接收请求Context，处理结束后向引擎调度器发送处理结果
* CheckStatus检查响应状态码的是否合法
* setTimeout设置超时时间
* Tegenaria提供了一个默认的基于net/http的下载器

### 4.7.2.下载器调度
```go
// recvRequest receive request from cacheChan and do download.
func (e *SpiderEngine) recvRequestHandler(req *Context) {
	defer e.waitGroup.Done()
	if req == nil {
		return
	}
	e.waitGroup.Add(1)
	go e.doDownload(req)

}

// doDownload handle request download
func (e *SpiderEngine) doDownload(ctx *Context) {
	defer func() {
		e.waitGroup.Done()
	}()
	// use download middleware to handle request object
	for _, middleware := range e.downloaderMiddlewares {
		err := middleware.ProcessRequest(ctx)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("Middleware %s handle request error %s", middleware.GetName(), err.Error())
			ctx.Error = err
			e.errorChan <- NewError(ctx.CtxId, err, ErrorWithRequest(ctx.Request))
			return
		}
	}
	// incr request download number
	atomic.AddUint64(&e.Stats.RequestDownloaded, 1)
	e.requestDownloader.Download(ctx, e.requestResultChan)
}
```
## 4.8.ItemMeta
解析结果字段存储格式元数据接口

```go
// Item as meta data process interface
type ItemInterface interface {
}
type ItemMeta struct{
	CtxId string
	Item ItemInterface
}
```
* 创建新item对象

```go
func NewItem(ctx *Context, item ItemInterface) *ItemMeta {
	return &ItemMeta{
		CtxId: ctx.CtxId,
		Item: item,
	}
}
```
* item调度处理

```go
// doPipelinesHandlers handle items by pipelines chan
func (e *SpiderEngine) doPipelinesHandlers(spider SpiderInterface, item *ItemMeta) {
	defer func() {
		e.waitGroup.Done()

	}()
	for _, pipeline := range e.pipelines {
		engineLog.WithField("request_id", item.CtxId).Debugf("Response parse items into pipelines chans")
		err := pipeline.ProcessItem(spider, item)
		if err != nil {
			handleError := NewError(item.CtxId, err, ErrorWithItem(item))
			e.errorChan <- handleError
			return
		}
	}
	atomic.AddUint64(&e.Stats.ItemScraped, 1)

}
```
## 4.9.PipelineInterface
### 4.9.1.pipeline接口说明
* pipeline主要用于处理item，引擎根据pipelines的优先级由高到低调度ProcessItem

```go
type PipelinesInterface interface {
	// GetPriority get pipeline Priority
	GetPriority() int
	// ProcessItem
	ProcessItem(spider SpiderInterface, item *ItemMeta) error
}
```
* 优先级排序由[sort接口](https://pkg.go.dev/sort)实现
* GetPriority获取中间件的优先级

* ProcessItem item处理函数
* pipelines队列

```go
type ItemPipelines []PipelinesInterface
```
### 4.9.2.pipeline注册
pipeline的注册由引擎完成

```go
// RegisterPipelines add items handle pipelines
func (e *SpiderEngine) RegisterPipelines(pipeline PipelinesInterface) {
	e.pipelines = append(e.pipelines, pipeline)
	sort.Sort(e.pipelines)
	engineLog.Infof("Register %v priority pipeline success\n", pipeline)
}
```
### 4.9.3.pipeline实例化示例
```go
type TestItemPipeline struct {
	Priority int
}
func (p *TestItemPipeline) ProcessItem(spider SpiderInterface, item *ItemMeta) error {
	i := item.Item.(*testItem)
	i.pipelines = append(i.pipelines, p.Priority)
	return nil

}
func (p *TestItemPipeline) GetPriority() int {
	return p.Priority
}
engine := tegenaria.NewSpiderEngine()
engine.RegisterPipelines(TestItemPipeline{Priority: 1})

```




小优先级越高

```go
type MiddlewaresInterface interface {
	GetPriority() int
	ProcessRequest(ctx *Context) error
	ProcessResponse(ctx *Context, req chan<- *Context) error
	GetName()string
}
```
* GetPriority获取中间件的优先级
* ProcessRequest在对Request进行下载处理前对其进行修饰处理，添加代理或User-Agent
* ProcessResponse在Response进入解析器前对response进行处理
* GetName获取中间件名称
* 优先级排序由[sort接口](https://pkg.go.dev/sort)实现

```go
func (p Middlewares) Len() int           { return len(p) }
func (p Middlewares) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Middlewares) Less(i, j int) bool { return p[i].GetPriority() < p[j].GetPriority() }
```
* 下载中间件队列定义

```go
type Middlewares []MiddlewaresInterface
```
### 4.6.2.中间件实例化
```go
type TestDownloadMiddler struct {
	Priority int
	Name     string
}

func (m TestDownloadMiddler) GetPriority() int {
	return m.Priority
}
func (m TestDownloadMiddler) ProcessRequest(ctx *Context) error {
	header := fmt.Sprintf("priority-%d", m.Priority)
	ctx.Request.Header[header] = strconv.Itoa(m.Priority)
	return nil
}

func (m TestDownloadMiddler) ProcessResponse(ctx *Context, req chan<- *Context) error {
	return nil

}
```
### 4.6.3.中间件注册
中间件注册由引擎实现

```go
// RegisterDownloadMiddlewares add a download middlewares
func (e *SpiderEngine) RegisterDownloadMiddlewares(middlewares MiddlewaresInterface) {
	e.downloaderMiddlewares = append(e.downloaderMiddlewares, middlewares)
	sort.Sort(e.downloaderMiddlewares)
}
```
### 4.6.4.ProcessRequest调度
按优先级由高到低调度

```go
	// use download middleware to handle request object
	for _, middleware := range e.downloaderMiddlewares {
		err := middleware.ProcessRequest(ctx)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("Middleware %s handle request error %s", middleware.GetName(), err.Error())
			ctx.Error = err
			e.errorChan <- NewError(ctx.CtxId, err, ErrorWithRequest(ctx.Request))
			return
		}
	}
```
### 4.6.5.ProcessResponse
按优先级由低到高调度

```go
// processResponse do handle download response
func (e *SpiderEngine) processResponse(ctx *Context) {
	if len(e.downloaderMiddlewares) == 0 {
		return
	}
	for index := range e.downloaderMiddlewares {
		middleware := e.downloaderMiddlewares[len(e.downloaderMiddlewares)-index-1]
		err := middleware.ProcessResponse(ctx, e.requestsChan)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("Middleware %s handle response error %s", middleware.GetName(), err.Error())
			ctx.Error = err
			e.errorChan <- NewError(ctx.CtxId, err, ErrorWithRequest(ctx.Request), ErrorWithResponse(ctx.DownloadResult.Response))
			return
		}
	}
}
```
## 4.7.Downloader
### 4.7.1.Tegenaria下载器接口
```go
// Downloader interface
type Downloader interface {
	// Download core funcation
	Download(ctx *Context, result chan<- *Context)

	// CheckStatus check response status code if allow handle
	CheckStatus(statusCode uint64, allowStatus []uint64) bool
	
	// setTimeout set downloader timeout
	setTimeout(timeout time.Duration)
}
```
* Download载器核心的网络请求处理函数，接收请求Context，处理结束后向引擎调度器发送处理结果
* CheckStatus检查响应状态码的是否合法
* setTimeout设置超时时间
* Tegenaria提供了一个默认的基于net/http的下载器

### 4.7.2.下载器调度
```go
// recvRequest receive request from cacheChan and do download.
func (e *SpiderEngine) recvRequestHandler(req *Context) {
	defer e.waitGroup.Done()
	if req == nil {
		return
	}
	e.waitGroup.Add(1)
	go e.doDownload(req)

}

// doDownload handle request download
func (e *SpiderEngine) doDownload(ctx *Context) {
	defer func() {
		e.waitGroup.Done()
	}()
	// use download middleware to handle request object
	for _, middleware := range e.downloaderMiddlewares {
		err := middleware.ProcessRequest(ctx)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("Middleware %s handle request error %s", middleware.GetName(), err.Error())
			ctx.Error = err
			e.errorChan <- NewError(ctx.CtxId, err, ErrorWithRequest(ctx.Request))
			return
		}
	}
	// incr request download number
	atomic.AddUint64(&e.Stats.RequestDownloaded, 1)
	e.requestDownloader.Download(ctx, e.requestResultChan)
}
```
## 4.8.ItemMeta
解析结果字段存储格式元数据接口

```go
// Item as meta data process interface
type ItemInterface interface {
}
type ItemMeta struct{
	CtxId string
	Item ItemInterface
}
```
* 创建新item对象

```go
func NewItem(ctx *Context, item ItemInterface) *ItemMeta {
	return &ItemMeta{
		CtxId: ctx.CtxId,
		Item: item,
	}
}
```
* item调度处理

```go
// doPipelinesHandlers handle items by pipelines chan
func (e *SpiderEngine) doPipelinesHandlers(spider SpiderInterface, item *ItemMeta) {
	defer func() {
		e.waitGroup.Done()

	}()
	for _, pipeline := range e.pipelines {
		engineLog.WithField("request_id", item.CtxId).Debugf("Response parse items into pipelines chans")
		err := pipeline.ProcessItem(spider, item)
		if err != nil {
			handleError := NewError(item.CtxId, err, ErrorWithItem(item))
			e.errorChan <- handleError
			return
		}
	}
	atomic.AddUint64(&e.Stats.ItemScraped, 1)

}
```
## 4.9.PipelineInterface
### 4.9.1.pipeline接口说明
* pipeline主要用于处理item，引擎根据pipelines的优先级由高到低调度ProcessItem

```go
type PipelinesInterface interface {
	// GetPriority get pipeline Priority
	GetPriority() int
	// ProcessItem
	ProcessItem(spider SpiderInterface, item *ItemMeta) error
}
```
* 优先级排序由[sort接口](https://pkg.go.dev/sort)实现
* GetPriority获取中间件的优先级

* ProcessItem item处理函数
* pipelines队列

```go
type ItemPipelines []PipelinesInterface
```
### 4.9.2.pipeline注册
pipeline的注册由引擎完成

```go
// RegisterPipelines add items handle pipelines
func (e *SpiderEngine) RegisterPipelines(pipeline PipelinesInterface) {
	e.pipelines = append(e.pipelines, pipeline)
	sort.Sort(e.pipelines)
	engineLog.Infof("Register %v priority pipeline success\n", pipeline)
}
```
### 4.9.3.pipeline实例化示例
```go
type TestItemPipeline struct {
	Priority int
}
func (p *TestItemPipeline) ProcessItem(spider SpiderInterface, item *ItemMeta) error {
	i := item.Item.(*testItem)
	i.pipelines = append(i.pipelines, p.Priority)
	return nil

}
func (p *TestItemPipeline) GetPriority() int {
	return p.Priority
}
engine := tegenaria.NewSpiderEngine()
engine.RegisterPipelines(TestItemPipeline{Priority: 1})

```




