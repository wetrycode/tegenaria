// Package tegenaria a spider network package

package tegenaria

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
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

type SpiderEngine struct {
	// spiders all register spiders modules
	spiders *Spiders

	// requestsChan *Request channel
	// sender is SpiderInterface.StartRequest
	// receiver is SpiderEngine.writeCache
	requestsChan chan *Request

	// itemsChan ItemInterface channel
	// sender is SpiderInterface.Parser
	// receiver is SpiderEngine.doPipelinesHandlers
	itemsChan chan ItemInterface

	// respChan *Response channel,its data is from doRequestResult
	// It will receive by Request.parser and handle
	respChan chan *Response

	// requestResultChan downloader downloads result and will be send by download after download handle finish
	// It will receive by doParse and response will be parse by Request.parser
	requestResultChan chan *RequestResult

	// errorChan all errors will be send to this channel during hold spider process
	// It will receive by doError funcation
	errorChan chan error

	// cacheChan a *Request channel its data is send by writeCache and the data source is from requests cache
	// Its data will receive by readCache and then will be handle by download
	cacheChan chan *Request

	// startRequestFinish it will be set True after StartRequest is done
	startRequestFinish bool

	// goroutineRunning all running tasks goroutine counter.
	// It is an important flag to judge whether the tasks are complated.
	// It will incr when an task goroutine start  and dec after goroutine is done.
	goroutineRunning *int64

	// pipelines items process chan.
	// Items should be handled by these pipenlines
	pipelines ItemPipelines

	// Ctx context.Context
	Ctx context.Context

	// DownloadTimeout the request handle timeout value
	DownloadTimeout time.Duration

	// requestDownloader global request downloader
	requestDownloader Downloader

	// allowStatusCode set allow  handle status codes which are not 200,like 404,302
	allowStatusCode []int64

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

	// isDone all scrap task is done flag
	// It will set for true until all channel is empty and goroutineRunning is 0 and startRequestFinish is true
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

	// concurrencyNum the limit number of request task goroutine at the same time
	concurrencyNum int64

	// currentRequest the number of running request handler at the same time
	currentRequest int64
}

var (
	Engine           *SpiderEngine // SpiderEngine global and once spider engine
	once             sync.Once
	engineLog        *logrus.Entry = GetLogger("engine") // engineLog engine runtime logger
	goroutineRunning int64         = 0                   // goroutineRunning all running tasks goroutine counter
)

// EngineOption the options params of NewDownloader
type EngineOption func(r *SpiderEngine)

// engineScheduler the engine core funcation.
// It will schedule all channel and handle spider task
func (e *SpiderEngine) engineScheduler(ctx context.Context, spider SpiderInterface) {
Loop:
	for {
		if e.isRunning {
			select {
			case req := <-e.requestsChan:
				// write request to cache
				// add goroutineRunning number
				atomic.AddInt64(e.goroutineRunning, 1)
				e.waitGroup.Add(1)
				go e.writeCache(req)
			case requestResult := <-e.requestResultChan:
				// handle request download result
				atomic.AddInt64(e.goroutineRunning, 1)
				e.waitGroup.Add(1)
				go e.doRequestResult(requestResult)
			case response := <-e.respChan:
				// handle request response
				atomic.AddInt64(e.goroutineRunning, 1)
				e.waitGroup.Add(1)
				go e.doParse(spider, response)
			case item := <-e.itemsChan:
				// handle scape items
				e.waitGroup.Add(1)
				atomic.AddInt64(e.goroutineRunning, 1)
				go e.doPipelinesHandlers(spider, item)
			case err := <-e.errorChan:
				// handle error
				e.waitGroup.Add(1)
				atomic.AddInt64(e.goroutineRunning, 1)
				go e.doError(err)
			case <-ctx.Done():
				break Loop
			default:
			}
		}
		runtime.Gosched()

	}
	engineLog.Info("Scheduler is done")
	e.isClosed = true
	e.waitGroup.Done()
}

// Start spider engine start.
// It will schedule all spider system
func (e *SpiderEngine) Start(spiderName string) {
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
	e.mainWaitGroup.Add(3)
	ctx, cancel := context.WithCancel(e.Ctx)
	// Monitor task running status and control ctx status
	go e.readyDone(ctx, cancel)
	// run Spiders StartRequest function and get feeds request
	go e.StartSpiders(spiderName)
	// read request from cacheChan and do download handle
	go e.recvRequest(ctx)
	for n := 0; n < int(e.cacheReadNum); n++ {
		e.mainWaitGroup.Add(1)
		// read request from cache and send to cacheChan
		go e.readCache(ctx)
	}
	// start schedulers
	for i := 0; i < int(e.schedulerNum); i++ {
		e.waitGroup.Add(1)
		go e.engineScheduler(ctx, spider)
	}
	e.mainWaitGroup.Wait()
	e.waitGroup.Wait()
	e.isDone = true
	// Output spiders stats
	engineLog.Infof("DownloadCount %d ErrorCount %d ItemScraped %d", e.Stats.RequestDownloaded, e.Stats.ErrorCount, e.Stats.ItemScraped)
}

// checkTaskStatus check goroutineRunning number and cache size.
// if goroutineRunning is zero and cache is empty it will return true, or not return false
func (e *SpiderEngine) checkTaskStatus() bool {
	return atomic.LoadInt64(e.goroutineRunning) == 0 && e.cache.getSize() == 0
}

// checkChanStatus check all channel if empty
func (e *SpiderEngine) checkChanStatus() bool {
	return (len(e.requestsChan) + len(e.requestResultChan) + len(e.respChan) + len(e.itemsChan) + len(e.errorChan) + len(e.cacheChan)) == 0
}

// readyDone monitor engine running status and control ctx status.
// It will check StartRequest if finish and task goroutine number and all channels len.
// if all status is ok it will stop engine and close spider
func (e *SpiderEngine) readyDone(ctx context.Context, cancel context.CancelFunc) {
	for {
		if e.startRequestFinish && e.checkTaskStatus() && e.checkChanStatus() {
			cancel()
			engineLog.Info("Scheduler ready done")
			e.mainWaitGroup.Done()
			return
		}
		time.Sleep(time.Second)
		runtime.Gosched()
	}

}

// recvRequest receive request from cacheChan and do download.
func (e *SpiderEngine) recvRequest(ctx context.Context) {
	defer e.mainWaitGroup.Done()
	for {
		if e.isRunning {
			select {
			case req := <-e.cacheChan:
				for {
					// Concurrency control,it will wait to add new download task until currentRequest < concurrencyNum
					if atomic.LoadInt64(&e.currentRequest) <= e.concurrencyNum {
						e.waitGroup.Add(1)
						atomic.AddInt64(e.goroutineRunning, 1)
						go e.doDownload(req)
						break
					}
					runtime.Gosched()
				}

			case <-ctx.Done():
				return
			default:
			}
			runtime.Gosched()
		}
	}

}

// StartSpiders start a spider specify by spider name
func (e *SpiderEngine) StartSpiders(spiderName string) {
	spider, ok := e.spiders.SpidersModules[spiderName]
	defer func() {
		e.startRequestFinish = true
		e.mainWaitGroup.Done()
	}()
	e.isRunning = true

	if !ok {
		panic(fmt.Sprintf("Spider %s not found", spider))
	}
	spider.StartRequest(e.requestsChan)
}

// writeCache write request from requestsChan to cache
func (e *SpiderEngine) writeCache(req *Request) {
	defer e.waitGroup.Done()
	e.cache.enqueue(req)
	atomic.AddInt64(e.goroutineRunning, -1)

}

// readCache read request from cache to cacheChan
func (e *SpiderEngine) readCache(ctx context.Context) {
	defer e.mainWaitGroup.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			req, err := e.cache.dequeue()
			if req != nil && err == nil {
				request := req.(*Request)
				e.cacheChan <- request
			}

		}
		runtime.Gosched()
	}
}

// doError handle all error which is from errorChan
func (e *SpiderEngine) doError(err error) {
	atomic.AddInt64(e.goroutineRunning, -1)
	atomic.AddUint64(&e.Stats.ErrorCount, 1)
	e.waitGroup.Done()
}

// doDownload handle request download
func (e *SpiderEngine) doDownload(request *Request) {
	defer func() {
		e.waitGroup.Done()
		atomic.AddInt64(e.goroutineRunning, -1)

	}()
	if e.doFilter(request) {
		atomic.AddInt64(&e.currentRequest, 1)
		// incr request download number
		atomic.AddUint64(&e.Stats.RequestDownloaded, 1)
		e.requestDownloader.Download(e.Ctx, request, e.requestResultChan)
		atomic.AddInt64(&e.currentRequest, -1)

	}
}

// doFilter filer duplicate request if filterDuplicateReq is true
func (e *SpiderEngine) doFilter(r *Request) bool {
	if e.filterDuplicateReq {
		result := r.doUnique(e.bloomFilter)
		if result {
			engineLog.WithField("request_id", r.RequestId).Debugf("Request is not unique")
		}
		return !result
	}
	return true
}

// doRequestResult handle download respose result
func (e *SpiderEngine) doRequestResult(result *RequestResult) {
	defer func() {
		atomic.AddInt64(e.goroutineRunning, -1)
		e.waitGroup.Done()
	}()
	err := result.Error
	if err != nil {
		e.errorChan <- err
		engineLog.WithField("request_id", result.RequestId).Errorf("Request is fail with error %s", err.Error())
		freeResponse(result.Response)

	} else {
		if e.requestDownloader.CheckStatus(result.Response.Status, e.allowStatusCode) {
			// response status code is ok
			// send response to respChan
			e.respChan <- result.Response
		} else {
			// send error
			engineLog.WithField("request_id", result.RequestId).Warningf("Not allow handle status code %d %s", result.Response.Status, result.Response.Req.Url)
			e.errorChan <- fmt.Errorf("%s %d", ErrNotAllowStatusCode.Error(), result.Response.Status)
			freeResponse(result.Response)
		}

	}

}

// doParse parse request response
func (e *SpiderEngine) doParse(spider SpiderInterface, resp *Response) {
	defer func() {
		atomic.AddInt64(e.goroutineRunning, -1)
		e.waitGroup.Done()

	}()
	e.Stats.NetworkTraffic += int64(resp.ContentLength)
	resp.Req.parser(resp, e.itemsChan, e.requestsChan)
	freeResponse(resp)
}

// doPipelinesHandlers handle items by pipelines chan
func (e *SpiderEngine) doPipelinesHandlers(spider SpiderInterface, item ItemInterface) {
	defer func() {
		atomic.AddInt64(e.goroutineRunning, -1)
		e.waitGroup.Done()

	}()
	for _, pipeline := range e.pipelines {
		engineLog.Infof("Response parse items into pipelines chans")
		err := pipeline.ProcessItem(spider, item)
		if err != nil {
			e.errorChan <- err
			return
		}
	}
	atomic.AddUint64(&e.Stats.ItemScraped, 1)

}

// Close engine and close all channels
func (e *SpiderEngine) Close() {
	defer func() {
		if p := recover(); p != nil {
			engineLog.Errorf("Close engier fail")
			panic("Close engier fail")
		}
	}()
	if e.checkChanStatus() {
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

}

// RegisterSpider add spiders
func (e *SpiderEngine) RegisterSpider(spider SpiderInterface) {
	e.spiders.Register(spider)
}

func EngineWithSpidersContext(ctx context.Context) EngineOption {
	return func(r *SpiderEngine) {
		r.Ctx = ctx
	}
}

func EngineWithSpidersTimeout(timeout time.Duration) EngineOption {
	return func(r *SpiderEngine) {
		r.DownloadTimeout = timeout
	}
}
func EngineWithSpidersDownloader(downloader Downloader) EngineOption {
	return func(r *SpiderEngine) {
		r.requestDownloader = downloader
	}
}
func EngineWithAllowStatusCode(allowStatusCode []int64) EngineOption {
	return func(r *SpiderEngine) {
		r.allowStatusCode = allowStatusCode
	}
}
func EngineWithUniqueReq(uniqueReq bool) EngineOption {
	return func(r *SpiderEngine) {
		r.filterDuplicateReq = uniqueReq
	}
}

func EngineWithSchedulerNum(schedulerNum uint) EngineOption {
	return func(r *SpiderEngine) {
		r.schedulerNum = schedulerNum
	}
}
func EngineWithReadCacheNum(cacheReadNum uint) EngineOption {
	return func(r *SpiderEngine) {
		r.cacheReadNum = cacheReadNum
	}
}
func EngineWithRequestNum(requestNum uint) EngineOption {
	return func(r *SpiderEngine) {
		r.cacheChan = make(chan *Request, requestNum)
	}
}
func EngineWithConcurrencyNum(concurrencyNum int64) EngineOption {
	return func(r *SpiderEngine) {
		r.concurrencyNum = concurrencyNum
	}
}

func NewSpiderEngine(opts ...EngineOption) *SpiderEngine {
	once.Do(func() {
		Engine = &SpiderEngine{
			spiders:            NewSpiders(),
			requestsChan:       make(chan *Request, 1024),
			itemsChan:          make(chan ItemInterface, 1024),
			respChan:           make(chan *Response, 1024),
			requestResultChan:  make(chan *RequestResult, 1024),
			errorChan:          make(chan error, 1024),
			cacheChan:          make(chan *Request, 1024),
			startRequestFinish: false,
			goroutineRunning:   &goroutineRunning,
			pipelines:          make(ItemPipelines, 0),
			Ctx:                context.TODO(),
			DownloadTimeout:    time.Second * 10,
			requestDownloader:  NewDownloader(),
			allowStatusCode:    []int64{},
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
			concurrencyNum:     256,
			currentRequest:     0,
		}
		for _, o := range opts {
			o(Engine)
		}
	})
	return Engine
}
