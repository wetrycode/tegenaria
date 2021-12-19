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

type SpiderEnginer struct {
	spiders            *Spiders
	requestsChan       chan *Request
	spidersChan        chan SpiderInterface
	itemsChan          chan ItemInterface
	respChan           chan *Response
	requestResultChan  chan *RequestResult
	errorChan          chan error
	startRequestFinish bool
	requestRunning     *uint
	pipelinesRunning   *uint
	parseRunning       *uint
	requestResultCount *uint
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
}

var Enginer *SpiderEnginer
var once sync.Once
var enginerLog *logrus.Entry = GetLogger("enginer")
var (
	requestRunning     uint = 0
	pipelinesRunning   uint = 0
	parseRunning       uint = 0
	requestResultCount uint = 0
)

type EnginerOption func(r *SpiderEnginer)

func (e *SpiderEnginer) eninerScheduler(spider SpiderInterface) {
Loop:
	for {
		if e.isRunning {
			select {
			case request := <-e.requestsChan:
				*(e.requestRunning)++
				e.waitGroup.Add(1)
				go e.doDownload(request)
			case requestResult := <-e.requestResultChan:
				*(e.requestResultCount)++
				e.waitGroup.Add(1)

				go e.doRequestResult(requestResult)
			case response := <-e.respChan:
				*(e.parseRunning)++
				e.waitGroup.Add(1)

				go e.doParse(spider, response)
			case item := <-e.itemsChan:
				e.waitGroup.Add(1)
				*(e.pipelinesRunning)++
				go e.doPipelinesHandlers(spider, item)
			default:
				if e.readyDone() {
					break Loop
				}
			}
		}

	}
	e.isClosed = true
	e.waitGroup.Done()
}
func (e *SpiderEnginer) Start(spiderName string) {
	defer func() {
		e.Close()
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
	e.waitGroup.Add(1)
	go e.StartSpiders(spiderName)
	for i := 0; i < int(e.schedulerNum); i++ {
		e.waitGroup.Add(1)
		go e.eninerScheduler(spider)
	}
	e.waitGroup.Wait()
	e.isDone = true
	e.Close()

}

func (e *SpiderEnginer) checkTaskStatus() bool {
	return (*e.requestRunning + *e.requestResultCount + *e.parseRunning + *e.pipelinesRunning) == 0
}
func (e *SpiderEnginer) checkChanStatus() bool {
	return (len(e.requestsChan) + len(e.requestResultChan) + len(e.respChan) + len(e.itemsChan)) == 0
}
func (e *SpiderEnginer) readyDone() bool {
	if e.startRequestFinish && e.checkTaskStatus() && e.checkChanStatus() {
		return true
	} else {
		return false
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
func (e *SpiderEnginer) ProcessInput(ch chan interface{}) {}
func (e *SpiderEnginer) ProcessOutput(ch chan interface{}) {
}
func (e *SpiderEnginer) ProcessException(ch chan interface{}) {}

func (e *SpiderEnginer) doDownload(request *Request) {
	defer func() {
		*(e.requestRunning)--
		e.waitGroup.Done()
	}()
	if e.doFilter(request) {
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
		*(e.requestResultCount)--
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
		*(e.parseRunning)--
		e.waitGroup.Done()

	}()
	spider.Parser(resp, e.itemsChan, e.requestsChan)
}

func (e *SpiderEnginer) doPipelinesHandlers(spider SpiderInterface, item ItemInterface) {
	defer func() {
		*(e.pipelinesRunning)--
		e.waitGroup.Done()

	}()
	for _, pipeline := range e.pipelines {
		err := pipeline.ProcessItem(spider, item)
		if err != nil {
			e.errorChan <- err
		}
	}

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
			startRequestFinish: false,
			requestRunning:     &requestRunning,
			pipelinesRunning:   &pipelinesRunning,
			parseRunning:       &parseRunning,
			requestResultCount: &requestResultCount,
			pipelines:          []PipelinesInterface{},
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
		}
		for _, o := range opts {
			o(Enginer)
		}
	})
	return Enginer
}
