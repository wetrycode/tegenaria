# Tegenaria
* [Tegenaria](#tegenaria)
* [1. Design Ideas](#1-design-ideas)
* [2. Data request processing flowchart](#2-data-request-processing-flowchart)
   * [2.1. Flow description](#21-flow-description)
      * [2.1.1. Main flow](#211-main-flow)
      * [2.1.2 Other processes](#212-other-processes)
   * [2.2. Scheduling Model](#22-scheduling-model)
* [3. Components](#3-components)
   * [3.1. Engine](#31-engine)
   * [3.2. Scheduler](#32-scheduler)
   * [3.3. Downloader](#33-downloader)
   * [3.4. De-duplication processors](#34-de-duplication-processors)
   * [3.5. Download middleware](#35-download-middleware)
   * [3.5. Download middleware](#35-download-middleware-1)
* [4.API](#4api)
   * [4.1.SpiderInterface](#41spiderinterface)
      * [4.1.1.Spider Interface description](#411spider-interface-description)
      * [4.1.2. Instantiating Spider](#412-instantiating-spider)
      * [4.1.3. engine start spider](#413-engine-start-spider)
      * [4.2.Context](#42context)
   * [4.3.Request](#43request)
   * [4.4.RFPDupeFilterInterface](#44rfpdupefilterinterface)
   * [4.5.Response](#45response)
   * [4.6. MiddlerwareInterface](#46-middlerwareinterface)
      * [4.6.1. Description of the middleware interface](#461-description-of-the-middleware-interface)
      * [4.6.2. Middleware instantiation](#462-middleware-instantiation)
      * [4.6.3. Middleware registration](#463-middleware-registration)
      * [4.6.4. ProcessRequest scheduling](#464-processrequest-scheduling)
      * [4.6.5.ProcessResponse](#465processresponse)
   * [4.7.Downloader](#47downloader)
      * [4.7.1. Tegenaria downloader interface](#471-tegenaria-downloader-interface)
      * [4.7.2. scheduling downloader](#472-scheduling-downloader)
   * [4.8.ItemMeta](#48itemmeta)
   * [4.9.PipelineInterface](#49pipelineinterface)
      * [4.9.1. pipeline interface description](#491-pipeline-interface-description)
      * [4.9.2. pipeline registration](#492-pipeline-registration)
      * [4.9.3.pipeline instantiation example](#493pipeline-instantiation-example)
# 1. Design Ideas
This project borrows the design idea of [scrapy](https://github.com/scrapy/scrapy) in terms of module function and data processing flow to realize the decoupling between modules of downloading, scheduling, parsing and data processing.
* The data interaction between modules is based on channel implementation.
* In terms of system scheduling, asynchronous processing is used between different requests, and the process of a single request is synchronized.
* Provide a unified pipelines and middlerwares interface as the entry point for business function extensions
* Leverage the multi-core processing power of golang
* Data fan-in and fan-out with engine and scheduler as the core

# 2. Data request processing flowchart
![image](images/scheduler.png)

## 2.1. Flow description
### 2.1.1. Main flow
* Spider constructs the seed request (Context) through the StartRequest method and sends it to the scheduler through the Request channel  
* The request object is first written to the cache queue (which uses memory by default), before the engine enables or skips the request de-duplicator depending on the settings
* If the request de-duplicator is enabled, it calculates the request fingerprint and puts it into a Bloom filter for de-duplication.
* If it is a duplicate request and the engine is set not to allow duplicate requests to be sent, the request is ignored otherwise the reuquest is written to the cache
* Scheduler reads Request from the cache and start the download processor
* The download processor calls the ProcessRequest method of the download middleware in order of priority before the official download.  
* The downloader generates a RequestResult after processing the request, including HandleError and Response, and then sends the request result to the scheduler via the RequestResult channel  
* The scheduler will enable a download result handler concurrently for each received RequestResult to process the HandleError to the error channel and the non-empty Response to the response channel.
* The scheduler will enable a parser processing concurrent for each received response through the response channel, and will execute ProcessResponse for processing the response according to the priority of the download middleware from lowest to highest before formally parsing the response, and the parser will call back the spider's Parser function
* The parser sends items to the dispatcher through the items channel in real time during the parsing process, and can also send new Requests to the dispatcher in real time.
* The dispatcher enables an item processor concurrently for each received item, and executes Pipelines' ProcessItem method according to priority during item processing
* Once the item processing is complete, a complete data request process is also completed

### 2.1.2 Other processes
* The scheduler enables an error handler for each HandleError received, which calls back the ErrorHandler defined by the spider and can generate a new request to be sent to the scheduler
* The HandleError generated by each processor at each stage is sent to the scheduler through the error channel and awaits dispatching.

## 2.2. Scheduling Model
Tegenaria uses an asynchronous scheduling model, where different modules of different processes communicate and pass data through the channel.

# 3. Components
## 3.1. Engine

* The engine is the core component of the whole Tegenaria framework, responsible for spider start/stop, data flow processing scheduling and some data statistics.
* The engine has built-in channels required for the whole system scheduling process.
* The engine enables a separate concurrent for each independent processor to handle asynchronous tasks
* Schedulers are built into the engine and can be set to enable a specified number of schedulers

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
* Cache readers are built into the engine to enable a specified number of cache readers depending on the settings

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
## 3.2. Scheduler
The core task of the scheduler is to receive data from each channel and distribute the received data to the corresponding processors and enable independent co-processing for processing.

```go
	for {
		if e.isRunning {
			// Avoid failing to exit when the cache queue is closed
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
			case response := <-e.requestChan:
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
## 3.3. Downloader
The network request module of the [Tegenaria downloader](https://github.com/wetrycode/tegenaria/blob/master/downloader.go) is based on the [net/http](https://pkg.go.dev/net/http) implementation Tegenaria provides a generic downloader interface for the business to implement its own downloader, which is responsible for handling all network requests

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
## 3.4. De-duplication processors
Tegenaria provides a default request de-duplication processor based on a Bloom filter implementation. The main implementation logic is as follows.

* normalize the url to process

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
* Recode the request header for processing

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
* Based on [third-party sha128 calculation library to obtain](https://github.com/spaolacci/murmur3) fingerprints

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
* De-duplication

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
## 3.5. Download middleware
Tegenaria provides a download middleware interface for implementing various types of download middleware, whose main functions are as follows.

* Uniform processing of requests by priority **from high to low** before entering the downloader, such as adding web agents and User-Agents
* Uniform processing of the response before entering the parser by priority **from low to high**, such as verifying whether the response body is legitimate, and sending a new request to the request channel at the same time.
* The engine provides a download middleware registration interface for registering download middleware with the engine

```go
// RegisterDownloadMiddlewares add a download middlewares
func (e *SpiderEngine) RegisterDownloadMiddlewares(middlewares MiddlewaresInterface) {
	e.downloaderMiddlewares = append(e.downloaderMiddlewares, middlewares)
	sort.Sort(e.downloaderMiddlewares)
}
```
## 3.5. Download middleware
Tegenaria provides a download middleware interface for implementing various types of download middleware, whose main functions are as follows.

* Uniform processing of requests by priority **from high to low** before entering the downloader, such as adding web agents and User-Agents
* Uniform processing of the response before entering the parser by priority **from low to high**, such as verifying whether the response body is legitimate, and sending a new request to the request channel at the same time.
* The engine provides a download middleware registration interface for registering download middleware with the engine

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
### 4.1.1.Spider Interface description
Tegenaria spider interface, developer can custom spider must be based on this interface to achieve custom spider.

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
* StartRequest, which is used to build the start request for seed urls and send the start request to the engine scheduler through the pipeline
* Parser, used to parse the response and generate the item or request, and send the generated data to the engine dispatcher via the specified pipeline
* ErrorHandler, handle the error captured by the engine callback execution
* GetName, get the spider name
* Tegenaria provides a BaseSpider for the business side to use

```go
// BaseSpider base spider
type BaseSpider struct {
	// Name spider name
	Name     string

	// FeedUrls feed urls
	FeedUrls []string
}
```
### 4.1.2. Instantiating Spider
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
### 4.1.3. engine start spider
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
* Context is the smallest scheduling unit in Tegenaria other than Item and is responsible for maintaining the lifecycle of the data processing process before it enters the pipelines process.

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
* CtxId is a unique stream number for each data processing, generated during the request initialization phase, and is a string in uuid format
* Create a new Context

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
Network request parameter configuration interface

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
* Web request processing object, used as follows, must provide URL and request method and response parsing method

```go
request := NewRequest(server.URL+"/testGET", GET, testParser)
```
* The parsing method type is defined as follows. The parsing function generates an item or a new request and sends it to the engine dispatcher via the specified channel for use in subsequent processes

```go
// Parser response parse handler
type Parser func(resp *Context, item chan<- *ItemMeta, req chan<- *Context) error
```


## 4.4.RFPDupeFilterInterface
* request fingerprint calculation and decomposition component interface

```go
// RFPDupeFilterInterface Request Fingerprint duplicates filter interface
type RFPDupeFilterInterface interface {
	// Fingerprint compute request fingerprint
	Fingerprint(request *Request) ([]byte, error)

	// DoDupeFilter do request fingerprint duplicates filter
	DoDupeFilter(request *Request) (bool, error)
}
```
* Tegenaria provides a default decomposition component based on Bloom filter implementation, and fingerprint calculation is based on sha128 calculation
* Fingerprint is the core code for fingerprint computation
* DoDupeFilter is used for the de-duplication process
* Tegenaria comes with a de-dupe processor that is used as follows.

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
* Engine start de-duplication processor

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
* Web request response object

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
* Read JSON data

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
* Read string

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
## 4.6. MiddlerwareInterface
### 4.6.1. Description of the middleware interface
Download middleware interface for Request and Response processing, the smaller the priority number the higher the priority

```go
type MiddlewaresInterface interface {
	GetPriority() int
	ProcessRequest(ctx *Context) error
	ProcessResponse(ctx *Context, req chan<- *Context) error
	GetName()string
}
```
* GetPriority gets the priority of the middleware
* ProcessRequest modifies the Request before it is downloaded and processed, adding a proxy or User-Agent.
* ProcessResponse processes the response before it enters the parser
* GetName gets the middleware name
* Priority sorting is implemented by [sort interface](https://pkg.go.dev/sort)

```go
func (p Middlewares) Len() int           { return len(p) }
func (p Middlewares) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Middlewares) Less(i, j int) bool { return p[i].GetPriority() < p[j].GetPriority() }
```
* Download middleware queue definition

```go
type Middlewares []MiddlewaresInterface
```
### 4.6.2. Middleware instantiation
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
### 4.6.3. Middleware registration
Middleware registration is implemented by the engine

```go
// RegisterDownloadMiddlewares add a download middlewares
func (e *SpiderEngine) RegisterDownloadMiddlewares(middlewares MiddlewaresInterface) {
	e.downloaderMiddlewares = append(e.downloaderMiddlewares, middlewares)
	sort.Sort(e.downloaderMiddlewares)
}
```
### 4.6.4. ProcessRequest scheduling
Scheduling from highest to lowest priority

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
Scheduling from lowest to highest priority

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
### 4.7.1. Tegenaria downloader interface
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
* The network request processing function in the core of the downloader, which receives the request Context and sends the processing result to the engine scheduler after processing.
* `CheckStatus` checks if the response status code is legal.
* `setTimeout` to set the timeout
* Tegenaria provides a default net/http based downloader

### 4.7.2. scheduling downloader

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
Parsing the result field to store metadata interface

```go
// Item as meta data process interface
type ItemInterface interface {
}
type ItemMeta struct{
	CtxId string
	Item ItemInterface
}
```
* Create a new item object

```go
func NewItem(ctx *Context, item ItemInterface) *ItemMeta {
	return &ItemMeta{
		CtxId: ctx.CtxId,
		Item: item,
	}
}
```
* item dispatch processing

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
### 4.9.1. pipeline interface description
* pipeline is mainly used for processing item, the engine schedules ProcessItem according to the priority of pipelines from highest to lowest

```go
type PipelinesInterface interface {
	// GetPriority get pipeline Priority
	GetPriority() int
	// ProcessItem
	ProcessItem(spider SpiderInterface, item *ItemMeta) error
}
```
* Priority sorting is implemented by [sort interface](https://pkg.go.dev/sort)
* GetPriority gets the priority of the middleware

* ProcessItem item processing function
* pipelines queue

```go
type ItemPipelines []PipelinesInterface
```
### 4.9.2. pipeline registration
Registration of pipeline is done by the engine

```go
// RegisterPipelines add items handle pipelines
func (e *SpiderEngine) RegisterPipelines(pipeline PipelinesInterface) {
	e.pipelines = append(e.pipelines, pipeline)
	sort.Sort(e.pipelines)
	engineLog.Infof("Register %v priority pipeline success\n", pipeline)
}
```
### 4.9.3.pipeline instantiation example
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



