// Package tegenaria a spider network package

package tegenaria

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	bloom "github.com/bits-and-blooms/bloom/v3"
	"github.com/sirupsen/logrus"
)

// SpiderStats is spiders running stats
type SpiderStats struct {
	ItemScraped uint64 // ItemScraped scraped item counter

	RequestDownloaded uint64 // RequestDownloaded request download counter

	NetworkTraffic int64 // NetworkTraffic network traffic counter

	ErrorCount uint64 // ErrorCount count all error recvice

}

// ErrorHandler a Customizable error handler funcation
// receive error from errchans
type ErrorHandler func(spider SpiderInterface, err *HandleError)

type SpiderEngine struct {
	// spiders all register spiders modules
	spiders *Spiders

	// requestsChan *Request channel
	// sender is SpiderInterface.StartRequest
	// receiver is SpiderEngine.writeCache
	requestsChan chan *Context

	// itemsChan ItemInterface channel
	// sender is SpiderInterface.Parser
	// receiver is SpiderEngine.doPipelinesHandlers
	itemsChan chan *ItemMeta

	// respChan *Response channel,its data is from doRequestResult
	// It will receive by Request.parser and handle
	respChan chan *Context

	// requestResultChan downloader downloads result and will be send by download after download handle finish
	// It will receive by doParse and response will be parse by Request.parser
	requestResultChan chan *Context

	// errorChan all errors will be send to this channel during hold spider process
	// It will receive by doError funcation
	errorChan chan *HandleError

	// cacheChan a *Request channel its data is send by writeCache and the data source is from requests cache
	// Its data will receive by readCache and then will be handle by download
	cacheChan chan *Context

	// quitSignal recv quit signal such as ctrl-c
	quitSignal chan os.Signal

	// startRequestFinish it will be set True after StartRequest is done
	startRequestFinish bool

	// pipelines items process chan.
	// Items should be handled by these pipenlines
	// Such as save item into databases
	pipelines ItemPipelines

	// middlewares are handle request object such as add proxy or header
	downloaderMiddlewares Middlewares

	// Ctx context.Context
	Ctx context.Context

	// DownloadTimeout the request handle timeout value
	DownloadTimeout time.Duration

	// requestDownloader global request downloader
	requestDownloader Downloader

	// allowStatusCode set allow  handle status codes which are not 200,like 404,302
	allowStatusCode []uint64

	// filterDuplicateReq flag if filter duplicate request fingerprint.
	// to filter duplicate request fingerprint set true or not set false.
	filterDuplicateReq bool

	// bloomFilter request fingerprint BloomFilter.
	// it will work if filterDuplicateReq is true
	bloomFilter *bloom.BloomFilter

	// engineStatus the engine status but not using.
	engineStatus int

	// waitGroup the engineScheduler inner goroutine task wait group
	waitGroup *sync.WaitGroup

	// mainWaitGroup the engine core scheduler funcation task wait group
	// It will ctrl readyDone、StartSpiders、recvRequest group
	mainWaitGroup *sync.WaitGroup

	// isDone is all scrap task is done flag
	// It will set for true until all channel is empty and
	// goroutineRunning is 0 and startRequestFinish is true
	isDone bool

	// isRunning the flag for engineScheduler start run
	isRunning bool

	// isClosed a flag for engine if is closed
	isClosed bool

	// schedulerNum the engineScheduler goroutine number default 3
	schedulerNum uint

	// Stats spider status counter and recorder
	Stats *SpiderStats

	// cache Request cache.
	// You can set your custom cache module,like redis
	cache CacheInterface

	// cacheReadNum count is read request number from cache
	cacheReadNum uint

	// ErrorHandler see ErrorHandler funcation description
	ErrorHandler ErrorHandler

	// timer engine running timer
	timer time.Time
}

var (
	Engine    *SpiderEngine // SpiderEngine global and once spider engine
	once      sync.Once
	engineLog *logrus.Entry = GetLogger("engine") // engineLog engine runtime logger
)

// engineScheduler the engine core funcation.
// It will schedule all channel and handle spider task
func (e *SpiderEngine) engineScheduler(spider SpiderInterface) {
Loop:
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
	engineLog.Info("Scheduler is done")
	e.isClosed = true
	e.mainWaitGroup.Done()
}

// statsReport output download and scraped stats count
func (e *SpiderEngine) statsReport() {
	engineLog.Infof("DownloadCount %d ErrorCount %d ItemScraped %d Timing %fs",
		e.Stats.RequestDownloaded, e.Stats.ErrorCount, e.Stats.ItemScraped, time.Since(e.timer).Seconds())

}

// statsReportTicker Output running status statistics every 5 seconds
func (e *SpiderEngine) statsReportTicker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		<-ticker.C
		e.statsReport()
	}

}

// listenNotify listen system signal such as ctrl-c
func (e *SpiderEngine) listenNotify() {
	for {
		select {
		case s, ok := <-e.quitSignal:
			if ok {
				engineLog.Warningln("Engine recv signal ", s)
				e.isDone = true
				close(e.cacheChan)
				close(e.quitSignal)
				signal.Stop(e.quitSignal)
				return
			} else {
				return
			}

		default:
			if e.isClosed {
				return
			}

		}

		runtime.Gosched()
	}

}

// Start spider engine start.
// It will schedule all spider system
func (e *SpiderEngine) Start(spiderName string) {
	signal.Notify(e.quitSignal, os.Interrupt, syscall.SIGUSR1, syscall.SIGUSR2)
	e.timer = time.Now()
	engineLog.Infof("Ready to start %s spider \n", spiderName)

	defer func() {
		e.Close()
		engineLog.Info("Spider engine is closed!")
		if p := recover(); p != nil {
			engineLog.Errorf("Close engier fail")
		}
	}()
	// Load an get specify spider object
	spider, ok := e.spiders.SpidersModules[spiderName]
	if !ok {
		panic(fmt.Sprintf("Spider %s not found", spider))
	}
	// engineScheduler number
	runtime.GOMAXPROCS(int(e.schedulerNum))
	e.waitGroup.Add(1)
	// run Spiders StartRequest function and get feeds request
	go e.StartSpiders(spiderName)
	// e.mainWaitGroup.Add(1)
	go e.listenNotify()
	for n := 0; n < int(e.cacheReadNum); n++ {
		e.mainWaitGroup.Add(1)
		// read request from cache and send to cacheChan
		go e.readCache()
	}
	// start schedulers
	for i := 0; i < int(e.schedulerNum); i++ {
		e.mainWaitGroup.Add(1)
		go e.engineScheduler(spider)
	}
	// Output handle stats counter pre 5s
	go e.statsReportTicker()
	engineLog.Info("Spider engine is running\n")

	e.waitGroup.Wait()
	e.isClosed = true
	engineLog.Info("Waitting engine stop\n")

	e.mainWaitGroup.Wait()

	e.isDone = true
	e.statsReport()
}

// checkChanStatus check all channel if empty
func (e *SpiderEngine) checkChanStatus() bool {
	return (len(e.requestsChan) + len(e.requestResultChan) + len(e.respChan) + len(e.itemsChan) + len(e.errorChan) + len(e.cacheChan)) == 0
}

// checkReadyDone monitor engine running status and control ctx status.
// It will check StartRequest if finish and task goroutine number and all channels len.
// if all status is ok it will stop engine and close spider

func (e *SpiderEngine) checkReadyDone() bool {
	if e.startRequestFinish && e.checkChanStatus() && e.isClosed {
		engineLog.Debug("Scheduler ready done")
		return true
	} else {
		return false
	}
}

// recvRequest receive request from cacheChan and do download.
func (e *SpiderEngine) recvRequestHandler(req *Context) {
	defer e.waitGroup.Done()
	if req == nil {
		return
	}
	e.waitGroup.Add(1)
	go e.doDownload(req)

}

// StartSpiders start a spider specify by spider name
func (e *SpiderEngine) StartSpiders(spiderName string) {
	spider, ok := e.spiders.SpidersModules[spiderName]
	defer func() {
		e.startRequestFinish = true
		e.waitGroup.Done()
	}()
	e.isRunning = true

	if !ok {
		panic(fmt.Sprintf("Spider %s not found", spider))
	}
	spider.StartRequest(e.requestsChan)
}

// writeCache write request from requestsChan to cache
func (e *SpiderEngine) writeCache(ctx *Context) {
	defer func() {
		e.waitGroup.Done()
	}()

	if e.doFilter(ctx, ctx.Request) && !e.isDone {
		err := e.cache.enqueue(ctx)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("Push request to cache queue error %s", err.Error())
		}
	}

}

// readCache read request from cache to cacheChan
func (e *SpiderEngine) readCache() {
	defer func() {
		e.mainWaitGroup.Done()
		engineLog.Debug("Close read cache\n")
		if p := recover(); p != nil {
			engineLog.Errorln("Read cache error \n", p)

		}
	}()
	for {
		req, err := e.cache.dequeue()
		if req != nil && err == nil {
			request := req.(*Context)
			e.cacheChan <- request
		}
		if e.isDone {
			return
		}
		runtime.Gosched()
	}
}

// doError handle all error which is from errorChan
func (e *SpiderEngine) doError(spider SpiderInterface, err *HandleError) {
	atomic.AddUint64(&e.Stats.ErrorCount, 1)
	e.ErrorHandler(spider, err)
	e.waitGroup.Done()
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
			e.errorChan <- NewError(ctx.CtxId, err)
			return
		}
	}
	// incr request download number
	atomic.AddUint64(&e.Stats.RequestDownloaded, 1)
	e.requestDownloader.Download(ctx, e.requestResultChan)
}

// doFilter filer duplicate request if filterDuplicateReq is true
func (e *SpiderEngine) doFilter(ctx *Context, r *Request) bool {
	if e.filterDuplicateReq {
		result, err := r.doUnique(e.bloomFilter)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Warningf("Request do unique error %s", err.Error())
			e.errorChan <- NewError(ctx.CtxId, fmt.Errorf("Request do unique error %s", err.Error()))
		}
		if result {
			engineLog.WithField("request_id", ctx.CtxId).Debugf("Request is not unique")
		}
		return !result
	}
	return true
}

// processResponse do handle download response
func (e *SpiderEngine) processResponse(ctx *Context) {
	if len(e.downloaderMiddlewares) == 0 {
		return
	}
	for index := range e.downloaderMiddlewares {
		middleware := e.downloaderMiddlewares[len(e.downloaderMiddlewares)-index-1]
		err := middleware.ProcessResponse(ctx)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("Middleware %s handle response error %s", middleware.GetName(), err.Error())
			ctx.Error = err
			e.errorChan <- NewError(ctx.CtxId, err)
			return
		}
	}
}

// doRequestResult handle download respose result
func (e *SpiderEngine) doRequestResult(result *Context) {
	defer func() {
		e.waitGroup.Done()

	}()

	if result.DownloadResult.Response != nil {
		e.processResponse(result)
	}
	err := result.DownloadResult.Error
	if err != nil {
		result.Error = err
		e.errorChan <- NewError(result.CtxId, err)
		engineLog.WithField("request_id", result.CtxId).Errorf("Request is fail with error %s", err.Error())
		freeRequest(result.Request)
		if result.DownloadResult.Response != nil {

			freeResponse(result.DownloadResult.Response)
		}

	} else {
		if e.requestDownloader.CheckStatus(uint64(result.DownloadResult.Response.Status), e.allowStatusCode) {
			// response status code is ok
			// send response to respChan
			engineLog.WithField("request_id", result.CtxId).Debugf("Request %s success status code %d", result.Request.Url, result.DownloadResult.Response.Status)

			e.respChan <- result

		} else {
			// send error
			engineLog.WithField("request_id", result.CtxId).Warningf("Not allow handle status code %d %s", result.DownloadResult.Response.Status, result.Request.Url)
			result.Error = fmt.Errorf("%s %d", ErrNotAllowStatusCode.Error(), result.DownloadResult.Response.Status)
			e.errorChan <- NewError(result.CtxId, result.Error)
			freeRequest(result.Request)
			if result.DownloadResult.Response != nil {

				freeResponse(result.DownloadResult.Response)
			}
		}

	}

}

// doParse parse request response
func (e *SpiderEngine) doParse(spider SpiderInterface, resp *Context) {
	defer func() {
		e.waitGroup.Done()
		// release Request and Response object memory to buffer
		freeRequest(resp.Request)

		freeResponse(resp.DownloadResult.Response)
	}()
	e.Stats.NetworkTraffic += int64(resp.DownloadResult.Response.ContentLength)
	resp.Request.parser(resp, e.itemsChan, e.requestsChan)
}

// doPipelinesHandlers handle items by pipelines chan
func (e *SpiderEngine) doPipelinesHandlers(spider SpiderInterface, item *ItemMeta) {
	defer func() {
		e.waitGroup.Done()

	}()
	for _, pipeline := range e.pipelines {
		engineLog.WithField("request_id", item.CtxId).Debugf("Response parse items into pipelines chans")
		err := pipeline.ProcessItem(spider, item)
		if err != nil {
			// TODO check response if is relase
			handleError := NewError(item.CtxId, err)
			e.errorChan <- handleError
			return
		}
	}
	atomic.AddUint64(&e.Stats.ItemScraped, 1)

}

// Close engine and close all channels
func (e *SpiderEngine) Close() {
	if e.checkReadyDone() {
		once.Do(func() {
			close(e.requestsChan)
			close(e.itemsChan)
			close(e.requestResultChan)
			close(e.respChan)
			close(e.errorChan)
		})
	}

}

// RegisterPipelines add items handle pipelines
func (e *SpiderEngine) RegisterPipelines(pipeline PipelinesInterface) {
	e.pipelines = append(e.pipelines, pipeline)
	sort.Sort(e.pipelines)
	engineLog.Infof("Register %v priority pipeline success\n", pipeline)

}

// RegisterDownloadMiddlewares add a download middlewares
func (e *SpiderEngine) RegisterDownloadMiddlewares(middlewares MiddlewaresInterface) {
	e.downloaderMiddlewares = append(e.downloaderMiddlewares, middlewares)
	sort.Sort(e.downloaderMiddlewares)
}

// RegisterSpider add spiders
func (e *SpiderEngine) RegisterSpider(spider SpiderInterface) {
	err := e.spiders.Register(spider)
	if err != nil {
		panic(err)
	}
	engineLog.Infof("Register %s spider success\n", spider.GetName())
}

// // DefaultErrorHandler error default handler
func DefaultErrorHandler(spider SpiderInterface, err *HandleError) {

}

func NewSpiderEngine(opts ...EngineOption) *SpiderEngine {
	// once.Do(func() {

	// })
	Engine = &SpiderEngine{
		spiders:               NewSpiders(),
		requestsChan:          make(chan *Context, 1024),
		itemsChan:             make(chan *ItemMeta, 1024),
		respChan:              make(chan *Context, 1024),
		requestResultChan:     make(chan *Context, 1024),
		errorChan:             make(chan *HandleError, 1024),
		cacheChan:             make(chan *Context, 1024),
		quitSignal:            make(chan os.Signal, 1),
		startRequestFinish:    false,
		pipelines:             make(ItemPipelines, 0),
		downloaderMiddlewares: make(Middlewares, 0),

		Ctx:                context.TODO(),
		DownloadTimeout:    time.Second * 10,
		requestDownloader:  NewDownloader(),
		allowStatusCode:    []uint64{},
		filterDuplicateReq: true,
		bloomFilter:        bloom.New(1024*4, 5),
		engineStatus:       0,
		waitGroup:          &sync.WaitGroup{},
		mainWaitGroup:      &sync.WaitGroup{},
		isDone:             false,
		isRunning:          false,
		schedulerNum:       4,
		Stats:              &SpiderStats{0, 0, 0.0, 0},
		cache:              NewRequestCache(),
		cacheReadNum:       2,
		ErrorHandler:       DefaultErrorHandler,
	}
	for _, o := range opts {
		o(Engine)
	}
	return Engine
}

// SetDownloadTimeout set download timeout
func (e *SpiderEngine) SetDownloadTimeout(timeout time.Duration) {
	e.DownloadTimeout = timeout
	e.requestDownloader.setTimeout(timeout)
}

// SetAllowedStatus set allowed response status codes
func (e *SpiderEngine) SetAllowedStatus(allowedStatusCode []uint64) {
	e.allowStatusCode = allowedStatusCode
}

//
