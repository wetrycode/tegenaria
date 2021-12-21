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

type SpiderStats struct {
	ItemScraped       uint64
	RequestDownloaded uint64
	NetworkTraffic    int64
	ErrorCount        uint64
}

type SpiderEngine struct {
	spiders            *Spiders
	requestsChan       chan *Request
	spidersChan        chan SpiderInterface
	itemsChan          chan ItemInterface
	respChan           chan *Response
	taskFinishChan     chan int
	requestResultChan  chan *RequestResult
	errorChan          chan error
	cacheChan          chan *Request
	startRequestFinish bool
	goroutineRunning   *int64
	pipelines          ItemPipelines
	Ctx                context.Context
	DownloadTimeout    time.Duration
	requestDownloader  Downloader
	allowStatusCode    []int64
	filterDuplicateReq bool
	bloomFilter        *bloom.BloomFilter
	engineStatus       int
	waitGroup          *sync.WaitGroup
	mainWaitGroup      *sync.WaitGroup
	isDone             bool
	isRunning          bool
	isClosed           bool
	schedulerNum       uint
	Stats              *SpiderStats
	cache              CacheInterface
}

var (
	Engine           *SpiderEngine
	once             sync.Once
	engineLog        *logrus.Entry = GetLogger("engine")
	goroutineRunning int64         = 0
)

type EngineOption func(r *SpiderEngine)

func (e *SpiderEngine) engineScheduler(ctx context.Context, spider SpiderInterface) {
Loop:
	for {
		if e.isRunning {
			select {
			case req := <-e.requestsChan:
				atomic.AddInt64(e.goroutineRunning, 1)
				e.waitGroup.Add(1)
				go e.writeCache(req)
			case request := <-e.cacheChan:
				atomic.AddInt64(e.goroutineRunning, 1)
				e.waitGroup.Add(1)
				go e.doDownload(request)
			case requestResult := <-e.requestResultChan:
				atomic.AddInt64(e.goroutineRunning, 1)
				e.waitGroup.Add(1)
				go e.doRequestResult(requestResult)
			case response := <-e.respChan:
				atomic.AddInt64(e.goroutineRunning, 1)
				e.waitGroup.Add(1)
				go e.doParse(spider, response)
			case item := <-e.itemsChan:
				e.waitGroup.Add(1)
				atomic.AddInt64(e.goroutineRunning, 1)
				go e.doPipelinesHandlers(spider, item)
			case err := <-e.errorChan:
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
	engineLog.Info("调度器完成调度")
	e.isClosed = true
	e.waitGroup.Done()
}
func (e *SpiderEngine) Start(spiderName string) {
	defer func() {
		e.Close()
		engineLog.Info("Spider engine is closed!")
		if p := recover(); p != nil {
			engineLog.Errorf("Close engier fail")
		}
	}()
	spider, ok := e.spiders.SpidersModules[spiderName]
	if !ok {
		panic(fmt.Sprintf("Spider %s not found", spider))
	}
	runtime.GOMAXPROCS(int(e.schedulerNum))
	e.mainWaitGroup.Add(3)
	ctx, cancel := context.WithCancel(e.Ctx)
	go e.readyDone(ctx, cancel)
	go e.StartSpiders(spiderName)
	go e.readCache(ctx)
	for i := 0; i < int(e.schedulerNum); i++ {
		e.waitGroup.Add(1)
		go e.engineScheduler(ctx, spider)
	}
	e.mainWaitGroup.Wait()
	e.waitGroup.Wait()
	e.isDone = true
	engineLog.Infof("下载总量为 %d 错误总量为%d item生成量%d", e.Stats.RequestDownloaded, e.Stats.ErrorCount, e.Stats.ItemScraped)
}

func (e *SpiderEngine) checkTaskStatus() bool {
	return atomic.LoadInt64(e.goroutineRunning) == 0 && e.cache.getSize() == 0
}
func (e *SpiderEngine) checkChanStatus() bool {
	return (len(e.requestsChan) + len(e.requestResultChan) + len(e.respChan) + len(e.itemsChan) + len(e.errorChan) + len(e.cacheChan)) == 0
}
func (e *SpiderEngine) readyDone(ctx context.Context, cancel context.CancelFunc) {
	for {
		if e.startRequestFinish && e.checkTaskStatus() && e.checkChanStatus() {
			cancel()
			engineLog.Info("准备关闭调度器")
			e.mainWaitGroup.Done()
			return
		}
		time.Sleep(time.Second)
		runtime.Gosched()
	}

}
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
func (e *SpiderEngine) writeCache(req *Request) {
	defer e.waitGroup.Done()
	e.cache.enqueue(req)
	atomic.AddInt64(e.goroutineRunning, -1)

}
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
func (e *SpiderEngine) doError(err error) {
	atomic.AddInt64(e.goroutineRunning, -1)
	atomic.AddUint64(&e.Stats.ErrorCount, 1)
	e.waitGroup.Done()
}

func (e *SpiderEngine) doDownload(request *Request) {
	defer func() {
		e.waitGroup.Done()
		atomic.AddInt64(e.goroutineRunning, -1)

	}()
	if e.doFilter(request) {
		atomic.AddUint64(&e.Stats.RequestDownloaded, 1)
		e.requestDownloader.Download(e.Ctx, request, e.requestResultChan)
	}
}
func (e *SpiderEngine) doFilter(r *Request) bool {
	if e.filterDuplicateReq {
		result := r.doUnique(e.bloomFilter)
		if result {
			engineLog.Debugf("Request is not unique")
		}
		return !result
	}
	return true
}
func (e *SpiderEngine) doRequestResult(result *RequestResult) {
	defer func() {
		atomic.AddInt64(e.goroutineRunning, -1)
		e.waitGroup.Done()
	}()
	err := result.Error
	if err != nil {
		e.errorChan <- err
		engineLog.Errorf("Request is fail with error %s", err.Error())
	} else {
		if e.requestDownloader.CheckStatus(int64(result.Response.Status), e.allowStatusCode) {
			e.respChan <- result.Response
		} else {
			engineLog.Warningf("Not allow handle status code %d", result.Response.Status)
			e.errorChan <- fmt.Errorf("%s %d", ErrNotAllowStatusCode.Error(), result.Response.Status)
		}

	}

}
func (e *SpiderEngine) doParse(spider SpiderInterface, resp *Response) {
	defer func() {
		atomic.AddInt64(e.goroutineRunning, -1)
		e.waitGroup.Done()

	}()
	e.Stats.NetworkTraffic += int64(resp.ContentLength)
	resp.Req.parser(resp, e.itemsChan, e.requestsChan)
}

func (e *SpiderEngine) doPipelinesHandlers(spider SpiderInterface, item ItemInterface) {
	defer func() {
		atomic.AddInt64(e.goroutineRunning, -1)
		e.waitGroup.Done()

	}()
	for _, pipeline := range e.pipelines {
		err := pipeline.ProcessItem(spider, item)
		if err != nil {
			e.errorChan <- err
			return
		}
	}
	atomic.AddUint64(&e.Stats.ItemScraped,1)

}
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
			close(e.spidersChan)
			close(e.itemsChan)
			close(e.requestResultChan)
			close(e.respChan)
			close(e.taskFinishChan)
			close(e.errorChan)
		})
	}

}
func (e *SpiderEngine) RegisterPipelines(pipeline PipelinesInterface) {
	e.pipelines = append(e.pipelines, pipeline)
	sort.Sort(e.pipelines)

}
func (e *SpiderEngine) RegisterSpider(spider SpiderInterface) {
	e.spiders.Register(spider)
}

func WithSpidersContext(ctx context.Context) EngineOption {
	return func(r *SpiderEngine) {
		r.Ctx = ctx
	}
}

func WithSpidersTimeout(timeout time.Duration) EngineOption {
	return func(r *SpiderEngine) {
		r.DownloadTimeout = timeout
	}
}
func WithSpidersDownloader(downloader Downloader) EngineOption {
	return func(r *SpiderEngine) {
		r.requestDownloader = downloader
	}
}
func WithAllowStatusCode(allowStatusCode []int64) EngineOption {
	return func(r *SpiderEngine) {
		r.allowStatusCode = allowStatusCode
	}
}
func WithUniqueReq(uniqueReq bool) EngineOption {
	return func(r *SpiderEngine) {
		r.filterDuplicateReq = uniqueReq
	}
}

func WithSchedulerNum(schedulerNum uint) EngineOption {
	return func(r *SpiderEngine) {
		r.schedulerNum = schedulerNum
	}
}

func NewSpiderEngine(opts ...EngineOption) *SpiderEngine {
	once.Do(func() {
		Engine = &SpiderEngine{
			spiders:            NewSpiders(),
			requestsChan:       make(chan *Request, 1024),
			spidersChan:        make(chan SpiderInterface),
			itemsChan:          make(chan ItemInterface, 1024),
			respChan:           make(chan *Response, 1024),
			requestResultChan:  make(chan *RequestResult, 1024),
			errorChan:          make(chan error, 1024),
			cacheChan:          make(chan *Request, 1024),
			taskFinishChan:     make(chan int),
			startRequestFinish: false,
			goroutineRunning:   &goroutineRunning,
			pipelines:          make(ItemPipelines, 0),
			Ctx:                context.TODO(),
			DownloadTimeout:    time.Second * 10,
			requestDownloader:  GoSpiderDownloader,
			allowStatusCode:    []int64{},
			filterDuplicateReq: true,
			bloomFilter:        bloom.New(1024*5, 5),
			engineStatus:       0,
			waitGroup:          &sync.WaitGroup{},
			mainWaitGroup:      &sync.WaitGroup{},
			isDone:             false,
			isRunning:          false,
			schedulerNum:       4,
			Stats:              &SpiderStats{0, 0, 0.0, 0},
			cache:              NewRequestCache(),
		}
		for _, o := range opts {
			o(Engine)
		}
	})
	return Engine
}
