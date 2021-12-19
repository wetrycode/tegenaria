package tegenaria

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	bloom "github.com/bits-and-blooms/bloom/v3"
	"github.com/sirupsen/logrus"
)

type SpiderStats struct {
	ItemScraped       uint64
	RequestDownloaded uint64
	NetworkTraffic    int64
}

type SpiderEnginer struct {
	spiders            *Spiders
	requestsChan       chan *Request
	spidersChan        chan SpiderInterface
	itemsChan          chan ItemInterface
	respChan           chan *Response
	taskFinishChan     chan int
	requestResultChan  chan *RequestResult
	errorChan          chan error
	startRequestFinish bool
	goroutineRunning   *uint
	pipelines          ItemPipelines
	Ctx                context.Context
	DownloadTimeout    time.Duration
	requestDownloader  Downloader
	allowStatusCode    []int64
	filterDuplicateReq bool
	bloomFilter        *bloom.BloomFilter
	enginerStatus      int
	waitGroup          *sync.WaitGroup
	isDone             bool
	isRunning          bool
	isClosed           bool
	schedulerNum       uint
	Stats              *SpiderStats
}

var (
	Enginer          *SpiderEnginer
	once             sync.Once
	enginerLog       *logrus.Entry = GetLogger("enginer")
	goroutineRunning uint          = 0
)

type EnginerOption func(r *SpiderEnginer)

func (e *SpiderEnginer) enginerScheduler(ctx context.Context, spider SpiderInterface) {
Loop:
	for {
		if e.isRunning {
			select {
			case request := <-e.requestsChan:
				*(e.goroutineRunning)++
				e.waitGroup.Add(1)
				go e.doDownload(request)
			case requestResult := <-e.requestResultChan:
				*(e.goroutineRunning)++
				e.waitGroup.Add(1)
				go e.doRequestResult(requestResult)
			case response := <-e.respChan:
				*(e.goroutineRunning)++
				e.waitGroup.Add(1)
				go e.doParse(spider, response)
			case item := <-e.itemsChan:
				e.waitGroup.Add(1)
				*(e.goroutineRunning)++
				go e.doPipelinesHandlers(spider, item)
			case err := <-e.errorChan:
				e.waitGroup.Add(1)
				*(e.goroutineRunning)++
				go e.doError(err)
			case <-ctx.Done():
				break Loop
			default:
			}
		}

	}
	e.isClosed = true
	e.waitGroup.Done()
}
func (e *SpiderEnginer) Start(spiderName string) {
	defer func() {
		e.Close()
		enginerLog.Info("Spider enginer is closed!")
		if p := recover(); p != nil {
			enginerLog.Errorf("Close engier fail")
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
		go e.enginerScheduler(ctx, spider)
	}
	e.waitGroup.Wait()
	e.isDone = true
}

func (e *SpiderEnginer) checkTaskStatus() bool {
	return *e.goroutineRunning == 0
}
func (e *SpiderEnginer) checkChanStatus() bool {
	return (len(e.requestsChan) + len(e.requestResultChan) + len(e.respChan) + len(e.itemsChan) + len(e.errorChan)) == 0
}
func (e *SpiderEnginer) readyDone(ctx context.Context, cancel context.CancelFunc) {
	for {
		if e.startRequestFinish && e.checkTaskStatus() && e.checkChanStatus() {
			cancel()
			e.waitGroup.Done()
		}
		time.Sleep(time.Second)
	}

}
func (e *SpiderEnginer) StartSpiders(spiderName string) {
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
func (e *SpiderEnginer) doError(err error) {
	*(e.goroutineRunning)--
	e.waitGroup.Done()
}

func (e *SpiderEnginer) doDownload(request *Request) {
	defer func() {
		*(e.goroutineRunning)--
		e.waitGroup.Done()
	}()
	if e.doFilter(request) {
		e.Stats.RequestDownloaded++
		e.requestDownloader.Download(e.Ctx, request, e.requestResultChan)
	}
}
func (e *SpiderEnginer) doFilter(r *Request) bool {
	if e.filterDuplicateReq {
		// 不存在
		if !e.bloomFilter.TestOrAdd(r.Fingerprint()) {
			return true
		}
		enginerLog.Info("Request is not unique")
		return false
	}
	return true
}
func (e *SpiderEnginer) doRequestResult(result *RequestResult) {
	defer func() {
		*(e.goroutineRunning)--
		e.waitGroup.Done()
	}()
	err := result.Error
	if err != nil {
		e.errorChan <- err
		enginerLog.Errorf("Request is fail with error %s", err.Error())
	} else {
		if e.requestDownloader.CheckStatus(int64(result.Response.Status), e.allowStatusCode) {
			e.respChan <- result.Response
		} else {
			enginerLog.Warningf("Not allow handle status code %d", result.Response.Status)
			e.errorChan <- fmt.Errorf("%s %d", ErrNotAllowStatusCode.Error(), result.Response.Status)
		}

	}

}
func (e *SpiderEnginer) doParse(spider SpiderInterface, resp *Response) {
	defer func() {
		*(e.goroutineRunning)--
		e.waitGroup.Done()

	}()
	e.Stats.NetworkTraffic += int64(resp.ContentLength)
	spider.Parser(resp, e.itemsChan, e.requestsChan)
}

func (e *SpiderEnginer) doPipelinesHandlers(spider SpiderInterface, item ItemInterface) {
	defer func() {
		*(e.goroutineRunning)--
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
func (e *SpiderEnginer) Close() {
	defer func() {
		if p := recover(); p != nil {
			enginerLog.Errorf("Close engier fail")
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
func (e *SpiderEnginer) RegisterPipelines(pipeline PipelinesInterface) {
	e.pipelines = append(e.pipelines, pipeline)
	sort.Sort(e.pipelines)

}
func (e *SpiderEnginer) RegisterSpider(spider SpiderInterface) {
	e.spiders.Register(spider)
}

func WithSpidersContext(ctx context.Context) EnginerOption {
	return func(r *SpiderEnginer) {
		r.Ctx = ctx
	}
}

func WithSpidersTimeout(timeout time.Duration) EnginerOption {
	return func(r *SpiderEnginer) {
		r.DownloadTimeout = timeout
	}
}
func WithSpidersDownloader(downloader Downloader) EnginerOption {
	return func(r *SpiderEnginer) {
		r.requestDownloader = downloader
	}
}
func WithAllowStatusCode(allowStatusCode []int64) EnginerOption {
	return func(r *SpiderEnginer) {
		r.allowStatusCode = allowStatusCode
	}
}
func WithUniqueReq(uniqueReq bool) EnginerOption {
	return func(r *SpiderEnginer) {
		r.filterDuplicateReq = uniqueReq
	}
}

func WithSchedulerNum(schedulerNum uint) EnginerOption {
	return func(r *SpiderEnginer) {
		r.schedulerNum = schedulerNum
	}
}

func NewSpiderEnginer(opts ...EnginerOption) *SpiderEnginer {
	once.Do(func() {
		Enginer = &SpiderEnginer{
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
			enginerStatus:      0,
			waitGroup:          &sync.WaitGroup{},
			isDone:             false,
			isRunning:          false,
			schedulerNum:       3,
			Stats:              &SpiderStats{0, 0, 0.0},
		}
		for _, o := range opts {
			o(Enginer)
		}
	})
	return Enginer
}
