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
	startRequestFinish bool
	goroutineRunning   *int64
	pipelines          ItemPipelines
	Ctx                context.Context
	DownloadTimeout    time.Duration
	requestDownloader  Downloader
	allowStatusCode    []int64
	filterDuplicateReq bool
	bloomFilter        *bloom.BloomFilter
	engineStatus      int
	waitGroup          *sync.WaitGroup
	isDone             bool
	isRunning          bool
	isClosed           bool
	schedulerNum       uint
	Stats              *SpiderStats
}

var (
	Engine          *SpiderEngine
	once             sync.Once
	engineLog       *logrus.Entry = GetLogger("engine")
	goroutineRunning int64         = 0
)

type EngineOption func(r *SpiderEngine)

func (e *SpiderEngine) engineScheduler(ctx context.Context, spider SpiderInterface) {
Loop:
	for {
		if e.isRunning {
			select {
			case request := <-e.requestsChan:
				atomic.AddInt64(e.goroutineRunning, 1)
				e.waitGroup.Add(1)
				go e.doDownload(request)
			case requestResult := <-e.requestResultChan:
				*(e.goroutineRunning)++
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

	}
	engineLog.Info("调度器完成调度")
	e.isClosed = true
	e.waitGroup.Done()
}
func (e *SpiderEngine) Start(spiderName string) {
	defer func() {
		e.Close()
		engineLog.Info("Spider enginer is closed!")
		if p := recover(); p != nil {
			engineLog.Errorf("Close engier fail")
		}
	}()
	spider, ok := e.spiders.SpidersModules[spiderName]
	if !ok {
		panic(fmt.Sprintf("Spider %s not found", spider))
	}
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)
	e.waitGroup.Add(2)
	ctx, cancel := context.WithCancel(e.Ctx)
	go e.readyDone(ctx, cancel)
	go e.StartSpiders(spiderName)
	for i := 0; i < int(e.schedulerNum); i++ {
		e.waitGroup.Add(1)
		go e.engineScheduler(ctx, spider)
	}

	e.waitGroup.Wait()
	e.isDone = true
}

func (e *SpiderEngine) checkTaskStatus() bool {
	engineLog.Infof("正在运行的协程任务数: %d", *e.goroutineRunning)
	return *e.goroutineRunning == 0
}
func (e *SpiderEngine) checkChanStatus() bool {
	return (len(e.requestsChan) + len(e.requestResultChan) + len(e.respChan) + len(e.itemsChan) + len(e.errorChan)) == 0
}
func (e *SpiderEngine) readyDone(ctx context.Context, cancel context.CancelFunc) {
	for {
		if e.startRequestFinish && e.checkTaskStatus() && e.checkChanStatus() {
			cancel()
			engineLog.Info("准备关闭调度器")
			e.waitGroup.Done()
		}
		time.Sleep(time.Second)
	}

}
func (e *SpiderEngine) StartSpiders(spiderName string) {
	spider, ok := e.spiders.SpidersModules[spiderName]
	defer func() {
		e.startRequestFinish = true
		e.isRunning = true
		e.waitGroup.Done()
		fmt.Print("请求结束\n")
	}()
	if !ok {
		panic(fmt.Sprintf("Spider %s not found", spider))
	}
	spider.StartRequest(e.requestsChan)
}
func (e *SpiderEngine) doError(err error) {
	atomic.AddInt64(e.goroutineRunning, -1)
	e.waitGroup.Done()
}

func (e *SpiderEngine) doDownload(request *Request) {
	defer func() {
		e.waitGroup.Done()
		atomic.AddInt64(e.goroutineRunning, -1)

	}()
	if e.doFilter(request) {
		e.Stats.RequestDownloaded++
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
	e.Stats.ItemScraped++

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
			requestsChan:       make(chan *Request, 1024*10),
			spidersChan:        make(chan SpiderInterface),
			itemsChan:          make(chan ItemInterface),
			respChan:           make(chan *Response),
			requestResultChan:  make(chan *RequestResult),
			errorChan:          make(chan error),
			taskFinishChan:     make(chan int),
			startRequestFinish: false,
			goroutineRunning:   &goroutineRunning,
			pipelines:          make(ItemPipelines, 0),
			Ctx:                context.TODO(),
			DownloadTimeout:    time.Second * 10,
			requestDownloader:  GoSpiderDownloader,
			allowStatusCode:    []int64{},
			filterDuplicateReq: true,
			bloomFilter:        bloom.New(1024*1024, 5),
			engineStatus:      0,
			waitGroup:          &sync.WaitGroup{},
			isDone:             false,
			isRunning:          false,
			schedulerNum:       3,
			Stats:              &SpiderStats{0, 0, 0.0},
		}
		for _, o := range opts {
			o(Engine)
		}
	})
	return Engine
}
