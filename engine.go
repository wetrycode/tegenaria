// MIT License

// Copyright (c) 2023 wetrycode

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package tegenaria

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc"
)

var engineLog *logrus.Entry = GetLogger("engine") // engineLog engine runtime logger
// workerUnit context处理单元
type workerUnit func(c *Context) error

// EventsWatcher 事件监听器
type EventsWatcher func(ch chan EventType) error

// CheckMasterLive 检查所有的master节点是否都在线
type CheckMasterLive func() (bool, error)

// CrawlEngine 引擎是整个框架数据流调度核心
type CrawlEngine struct {
	// spiders 已经注册的spider
	// 引擎调度的spider实例从此处根据爬虫名获取
	spiders *Spiders

	// pipelines 处理item的pipelines
	pipelines ItemPipelines

	// middlewares 注册的下载中间件
	downloaderMiddlewares Middlewares

	// waitGroup 引擎调度任务协程控制器
	waitGroup *conc.WaitGroup
	// downloader 下载器
	downloader Downloader
	// cache request队列缓存
	// 可以自定义实现cache接口，例如通过redis实现缓存
	cache CacheInterface
	// limiter 限速器，用于并发控制
	limiter LimitInterface
	// checkMasterLive 检查所有的master是否都在线
	checkMasterLive CheckMasterLive

	// 整个系统产生的request对象都应该通过该channel
	// 提交到引擎
	requestsChan chan *Context
	// cacheChan 缓存Context数据数据流管道
	// 主要用于与cache交互
	cacheChan chan *Context
	// eventsChan 事件管道
	// 数据抓取过程中发起的事件都需要通过该channel与引擎进行交互
	eventsChan chan EventType
	// filterDuplicateReq 是否进行去重处理
	filterDuplicateReq bool
	// RFPDupeFilter 去重组件
	rfpDupeFilter RFPDupeFilterInterface
	// startSpiderFinish spider.StartRequest是否已经执行结束
	startSpiderFinish bool
	// statistic 数据统计组件
	statistic StatisticInterface
	// hooker 事件监听和处理组件
	hooker EventHooksInterface
	// isStop 当前的引擎是否已经停止处理数据
	// 引擎会每三秒种调用checkReadyDone检查所有的任务
	// 是否都已经结束，是则将isStop设置为true
	isStop bool
	// useDistributed 是否启用分布式年模式
	useDistributed bool
	// isMaster是否是master节点
	isMaster bool
	// mutex spider 启动锁
	// 同一进程下只能启动一个spider
	mutex sync.Mutex
	// interval 定时任务执行时间间隔
	interval time.Duration

	// statusOn 当前引擎的状态
	statusOn StatusType
	// ticker 计时器
	ticker *time.Ticker

	// currentSpider 当前正在运行的spider 实例
	currentSpider SpiderInterface

	// ctxCount
	ctxCount  int64
	StartAt   int64
	Duration  float64
	StopAt    int64
	restartAt int64
	onceStart sync.Once
}

// RegisterSpiders 将spider实例注册到引擎的 spiders
func (e *CrawlEngine) RegisterSpiders(spider SpiderInterface) {
	err := e.spiders.Register(spider)
	if err != nil {
		panic(err)
	}
	engineLog.Infof("Register %s spider success\n", spider.GetName())
}

// RegisterPipelines 注册pipelines到引擎
func (e *CrawlEngine) RegisterPipelines(pipeline PipelinesInterface) {
	e.pipelines = append(e.pipelines, pipeline)
	sort.Sort(e.pipelines)
	engineLog.Debugf("Register %v priority pipeline success\n", pipeline)

}

// RegisterDownloadMiddlewares 注册下载中间件到引擎
func (e *CrawlEngine) RegisterDownloadMiddlewares(middlewares MiddlewaresInterface) {
	e.downloaderMiddlewares = append(e.downloaderMiddlewares, middlewares)
	sort.Sort(e.downloaderMiddlewares)
}

// startSpider 通过爬虫名启动爬虫
func (e *CrawlEngine) startSpider(spiderName string) GoFunc {
	_spiderName := spiderName
	return func() error {
		defer func() {
			e.startSpiderFinish = true

		}()
		spider, ok := e.spiders.SpidersModules[_spiderName]

		if !ok {
			panic(ErrSpiderNotExist)
		}
		// 对相关组件设置当前的spider
		e.statistic.setCurrentSpider(spiderName)
		e.cache.setCurrentSpider(spiderName)
		e.currentSpider = spider
		if e.useDistributed && !e.isMaster {
			// 分布式模式下非master组件不启动StartRequest
			start := time.Now()
			for {
				time.Sleep(3 * time.Second)
				live, err := e.checkMasterLive()
				engineLog.Infof("有主节点存在:%v", live)
				if (!live) || (err != nil) {
					if err != nil {
						panic(fmt.Sprintf("check master nodes status error %s", err.Error()))
					}
					engineLog.Warnf("正在等待主节点上线")
					if time.Since(start) > 30*time.Second {
						panic(ErrNoMaterNodeLive)
					}
					continue
				}
				break
			}
		} else {
			spider.StartRequest(e.requestsChan)
		}

		return nil
	}

}

// Execute 通过命令行启动spider
func (e *CrawlEngine) Execute() {
	ExecuteCmd(e)
	defer rootCmd.ResetCommands()

}

// Start 爬虫启动器
func (e *CrawlEngine) Start(spiderName string) StatisticInterface {
	e.mutex.Lock()
	defer func() {
		if p := recover(); p != nil {
			e.mutex.Unlock()
			panic(p)
		}
		e.mutex.Unlock()
	}()
	e.onceStart.Do(func() {
		e.StartAt = time.Now().Unix()

	})
	e.restartAt = time.Now().Unix()
	// 引入引擎所有的组件，并通过协程执行
	e.eventsChan <- START

	tasks := []GoFunc{e.startSpider(spiderName), e.recvRequest, e.Scheduler}
	e.statusOn = ON_START
	wg := &conc.WaitGroup{}
	// 事件监听器，通过协程执行
	hookTasks := []GoFunc{e.EventsWatcherRunner}
	// AddGo(wg, hookTasks...)
	GoRunner(context.Background(), wg, hookTasks...)
	GoRunner(context.Background(), e.waitGroup, tasks...)

	// AddGo(e.waitGroup, tasks...)
	p := e.waitGroup.WaitAndRecover()
	e.eventsChan <- EXIT
	engineLog.Infof("Wating engine to stop...")
	wg.Wait()
	stats := e.statistic.GetAllStats()
	s := Map2String(stats)
	engineLog.Infof(s)
	e.startSpiderFinish = false
	e.isStop = false
	e.statusOn = ON_STOP
	if p != nil {
		panic(p)
	}
	return e.statistic
}

// StartWithTicker 以定时任务的方式启动
func (e *CrawlEngine) StartWithTicker(spiderName string) {
	e.ticker.Reset(e.interval)
	e.Start(spiderName)
	for range e.ticker.C {
		if e.statusOn == ON_PAUSE {
			continue
		}
		e.Start(spiderName)
		e.statusOn = ON_PAUSE
	}

}

// EventsWatcherRunner 事件监听器运行组件
func (e *CrawlEngine) EventsWatcherRunner() error {
	err := e.hooker.EventsWatcher(e.eventsChan)
	if err != nil {
		return fmt.Errorf("events watcher task execution error %s", err.Error())
	}
	return nil
}

// Scheduler 调度器
func (e *CrawlEngine) Scheduler() error {
	for {
		if e.isStop {
			e.StopAt = time.Now().Unix()
			return nil
		}
		if e.statusOn == ON_PAUSE {
			e.eventsChan <- HEARTBEAT
			runtime.Gosched()
			continue
		}
		// 从缓存队列中读取请求对象
		req, err := e.cache.dequeue()
		// engineLog.Infof("读取对象:%s",err)
		if err != nil {
			runtime.Gosched()
			continue
		}
		atomic.AddInt64(&e.ctxCount, 1)
		request := req.(*Context)
		// 对request进行处理
		f := []GoFunc{e.worker(request)}
		// AddGo(e.waitGroup, f...)
		duration := time.Since(time.Unix(e.restartAt, 0))
		e.Duration = duration.Seconds() + e.Duration
		engineLog.Infof("%d:运行时长:%d", e.restartAt, duration)
		GoRunner(context.TODO(), e.waitGroup, f...)

	}
}

// worker request处理器，单个context生成周期都基于此
// 该函数包含了整个request和context的处理过程
func (e *CrawlEngine) worker(ctx *Context) GoFunc {
	c := ctx
	return func() error {
		defer func() {
			if c.Error != nil {
				// 新增一个错误
				e.statistic.Incr("errors")
				// 提交一个错误事件
				e.eventsChan <- ERROR
				// 将错误交给自定义的spider错误处理函数
				c.Spider.ErrorHandler(c, e.requestsChan)
			}
			// 处理结束回收context
			c.Close()
			atomic.AddInt64(&e.ctxCount, -1)
			// c.Cancel()
		}()
		// 发起心跳检查
		e.eventsChan <- HEARTBEAT
		// item启用新的协程进行处理
		wg := &conc.WaitGroup{}
		funcs := []GoFunc{func() error {
			err := e.doPipelinesHandlers(c)
			if err != nil {
				c.Error = NewError(c, errors.New(err.Error()))
			}
			return err
		},
		}
		GoRunner(ctx, wg, funcs...)
		// AddGo(wg, funcs...)
		// 处理request的所有工作单元
		units := []workerUnit{e.doDownload, e.doHandleResponse, e.doParse}
		// 依次执行工作单元
		for _, unit := range units {
			err := unit(c)
			if err != nil || c.Error != nil {
				if c.Error == nil {
					c.Error = NewError(c, err)
				}
				break
			}
		}
		close(ctx.Items)
		wg.Wait()
		return nil
	}
}

// recvRequest 从requestsChan读取context对象
func (e *CrawlEngine) recvRequest() error {
	for {
		select {
		case req := <-e.requestsChan:
			atomic.AddInt64(&e.ctxCount, 1)
			e.writeCache(req)
		// 每三秒钟检查一次所有的任务是否都已经结束
		case <-time.After(time.Second * 3):
			if e.checkReadyDone() || e.statusOn == ON_STOP {
				e.isStop = true
				return nil
			}

		}
		runtime.Gosched()
	}
}

// checkReadyDone 检查任务状态
// 主要检查spider.StartRequest是否已经执行完成
// 所有的context是否都已经关闭
// 队列是否为空
func (e *CrawlEngine) checkReadyDone() bool {
	return e.startSpiderFinish && atomic.LoadInt64(&e.ctxCount) == 0 && e.cache.isEmpty()
}

// writeCache 将Context 写入缓存
func (e *CrawlEngine) writeCache(ctx *Context) error {
	// var isDuplicated bool = false
	defer func() {
		if err := recover(); err != nil {
			ctx.setError(fmt.Sprintf("write cache error %s", err), string(debug.Stack()))
		}
		// // 写入分布式组件后主动删除
		// if e.useDistributed || isDuplicated {
		// 	ctx.Close()
		// }
		atomic.AddInt64(&e.ctxCount, -1)

	}()
	var err error = nil
	// 是否进入去重流程
	if e.filterDuplicateReq && !ctx.Request.DoNotFilter {
		ret, err := e.rfpDupeFilter.DoDupeFilter(ctx)
		if err != nil {
			// isDuplicated = true
			engineLog.WithField("request_id", ctx.CtxID).Errorf("request unique error %s", err.Error())
			return err
		}
		// request重复则直接返回
		if ret {
			// isDuplicated = true
			return nil
		}
	}
	// ctx进入缓存队列
	err = e.cache.enqueue(ctx)
	if err != nil {
		engineLog.Errorf("进入队列失败:%s", err.Error())
		engineLog.WithField("request_id", ctx.CtxID).Warnf("Cache enqueue error %s", err.Error())
		time.Sleep(time.Second)
		e.cacheChan <- ctx
	}
	err = nil
	ctx.Error = err
	return nil

}

// doDownload 对context执行下载操作
func (e *CrawlEngine) doDownload(ctx *Context) (err error) {
	defer func() {
		if p := recover(); p != nil {
			ctx.setError(fmt.Sprintf("Download error %s", p), string(debug.Stack()))
			err = fmt.Errorf("Download error %s", p)
		}
		if err != nil || ctx.Err() != nil {
			e.statistic.Incr("download_fail")
		}
	}()
	// 通过下载中间件对request进行处理
	// 按优先级调用ProcessRequest
	for _, middleware := range e.downloaderMiddlewares {
		err := middleware.ProcessRequest(ctx)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxID).Errorf("Middleware %s handle request error %s", middleware.GetName(), err.Error())
			return err
		}
	}
	// 执行限速器，速率过高则等待
	err = e.limiter.checkAndWaitLimiterPass()
	if err != nil {
		return err
	}
	// 增加请求发送量
	e.statistic.Incr(RequestStats)
	engineLog.WithField("request_id", ctx.CtxID).Infof("%s request ready to download", ctx.CtxID)
	rsp, err := e.downloader.Download(ctx)
	if rsp != nil {
		e.statistic.Incr(strconv.Itoa(rsp.Status))
	}
	if err != nil {
		return err
	}
	ctx.setResponse(rsp)
	return nil

}

// doHandleResponse 处理请求响应
func (e *CrawlEngine) doHandleResponse(ctx *Context) error {
	defer func() {
		if p := recover(); p != nil {
			ctx.setError(fmt.Sprintf("handle response error %s", p), string(debug.Stack()))
		}
	}()
	if ctx.Response == nil {
		err := fmt.Errorf("response is nil")
		engineLog.WithField("request_id", ctx.CtxID).Errorf("Request is fail with error %s", err.Error())
		return err
	}
	// 检查状态码是否合法
	if !e.downloader.CheckStatus(uint64(ctx.Response.Status), ctx.Request.AllowStatusCode) {
		err := fmt.Errorf("%s %d", ErrNotAllowStatusCode.Error(), ctx.Response.Status)
		engineLog.WithField("request_id", ctx.CtxID).Errorf("Request is fail with error %s", err.Error())
		return err
	}

	if len(e.downloaderMiddlewares) == 0 {
		return nil
	}
	// 逆优先级调用ProcessResponse
	for index := range e.downloaderMiddlewares {
		middleware := e.downloaderMiddlewares[len(e.downloaderMiddlewares)-index-1]
		err := middleware.ProcessResponse(ctx, e.requestsChan)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxID).Errorf("Middleware %s handle response error %s", middleware.GetName(), err.Error())
			return err
		}
	}
	return nil

}

// doParse 调用解析逻辑
func (e *CrawlEngine) doParse(ctx *Context) error {
	defer func() {
		if err := recover(); err != nil {
			ctx.setError(fmt.Sprintf("parse error %s", err), string(debug.Stack()))
		}

	}()
	if ctx.Response == nil {
		return nil
	}
	engineLog.WithField("request_id", ctx.CtxID).Infof("%s request response ready to parse", ctx.CtxID)
	args := make([]reflect.Value, 2)
	args[0] = reflect.ValueOf(ctx)
	args[1] = reflect.ValueOf(e.requestsChan)
	rets := GetParserByName(ctx.Spider, ctx.Request.Parser).Call(args)
	var parserErr error = nil
	if !rets[0].IsNil() {
		// return nil
		parserErr = rets[0].Interface().(error)
	}
	// parserErr := ctx.Request.Parser(ctx, e.requestsChan)
	if parserErr != nil {
		engineLog.Errorf("%s", parserErr.Error())
	}
	return parserErr

}

// doPipelinesHandlers 通过pipeline 处理item
func (e *CrawlEngine) doPipelinesHandlers(ctx *Context) error {
	defer func() {
		if err := recover(); err != nil {
			ctx.setError(fmt.Sprintf("pipeline error %s", err), string(debug.Stack()))
		}
		engineLog.WithField("request_id", ctx.CtxID).Infof("pipelines pass")
	}()
	for item := range ctx.Items {
		e.statistic.Incr(ItemsStats)
		engineLog.WithField("request_id", ctx.CtxID).Infof("%s item is scraped", item.CtxID)
		for _, pipeline := range e.pipelines {
			err := pipeline.ProcessItem(ctx.Spider, item)
			if err != nil {
				engineLog.WithField("request_id", ctx.CtxID).Errorf("Pipeline %d handle item error %s", pipeline.GetPriority(), err.Error())
				return err
			}
		}
	}
	return nil
}

// GetSpiders 获取所有的已经注册到引擎的spider实例
func (e *CrawlEngine) GetSpiders() *Spiders {
	return e.spiders
}

// Close 关闭引擎
func (e *CrawlEngine) Close() {
	close(e.requestsChan)
	close(e.cacheChan)
	e.ticker.Stop()

}

// StopTicker 关闭定时任务的计时器
func (e *CrawlEngine) StopTicker() {
	e.ticker.Stop()
}

// SetInterval 设置时间间隔
func (e *CrawlEngine) SetInterval(interval time.Duration) {
	e.interval = interval
	e.ticker.Reset(interval)
}

// SetMaster 设置是否是主节点
func (e *CrawlEngine) SetMaster(isMaster bool) {
	e.isMaster = isMaster
}

// SetStatus 设置引擎状态
// 用于控制引擎的启停
func (e *CrawlEngine) SetStatus(status StatusType) {
	e.statusOn = status
}

// GetStatic 获取StatisticInterface 统计指标
func (e *CrawlEngine) GetStatic() StatisticInterface {
	return e.statistic
}

// GetStatusOn 获取引擎的状态
func (e *CrawlEngine) GetStatusOn() StatusType {
	return e.statusOn
}

// GetCurrentSpider 获取当前正在运行的spider
func (e *CrawlEngine) GetCurrentSpider() SpiderInterface {
	return e.currentSpider
}
func (e *CrawlEngine) GetStartAt() int64 {
	return e.StartAt
}
func (e *CrawlEngine) GetStopAt() int64 {
	return e.StopAt
}
func (e *CrawlEngine) GetDuration() float64 {
	return decimal.NewFromFloat(e.Duration).Round(2).InexactFloat64()
}
func (e *CrawlEngine) GetRole() string {
	if e.isMaster {
		return "master"
	}
	return "solve"
}

// NewEngine 构建新的引擎
func NewEngine(opts ...EngineOption) *CrawlEngine {
	Engine := &CrawlEngine{
		waitGroup:             &conc.WaitGroup{},
		spiders:               NewSpiders(),
		requestsChan:          make(chan *Context, 1024),
		pipelines:             make(ItemPipelines, 0),
		downloaderMiddlewares: make(Middlewares, 0),
		cache:                 NewRequestCache(),
		cacheChan:             make(chan *Context, 512),
		eventsChan:            make(chan EventType, 16),
		statistic:             NewStatistic(),
		filterDuplicateReq:    true,
		rfpDupeFilter:         NewRFPDupeFilter(0.001, 1024*4),
		isStop:                false,
		useDistributed:        false,
		isMaster:              true,
		checkMasterLive:       func() (bool, error) { return true, nil },
		limiter:               NewDefaultLimiter(32),
		downloader:            NewDownloader(),
		hooker:                NewDefaultHooks(),
		interval:              -1 * time.Second,
		statusOn:              ON_STOP,
		ticker:                time.NewTicker(1),
		currentSpider:         nil,
		ctxCount:              0,
		onceStart:             sync.Once{},
		StartAt:               0,
		StopAt:                0,
		restartAt:             0,
		Duration:              0,
	}
	for _, o := range opts {
		o(Engine)
	}
	return Engine
}
