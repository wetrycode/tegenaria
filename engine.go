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
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc"
)

var engineLog *logrus.Entry = GetLogger("engine") // engineLog engine runtime logger
// workerUnit context处理单元
type workerUnit func(c *Context) error

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

	// requestsChan 整个系统产生的request对象都应该通过该channel
	// 提交到引擎
	requestsChan chan *Context
	// reqChannelSize 请求channel的大小
	reqChannelSize int

	// eventsChan 事件管道
	// 数据抓取过程中发起的事件都需要通过该channel与引擎进行交互
	eventsChan chan EventType
	// filterDuplicateReq 是否进行去重处理
	filterDuplicateReq bool
	// // startSpiderFinish spider.StartRequest是否已经执行结束
	startSpiderFinish bool
	// isStop 当前的引擎是否已经停止处理数据
	// 引擎会每三秒种调用checkReadyDone检查所有的任务
	// 是否都已经结束，是则将isStop设置为true
	isStop bool
	// mutex spider 启动锁
	// 同一进程下只能启动一个spider
	mutex sync.Mutex
	// currentSpider 当前正在运行的spider 实例
	currentSpider SpiderInterface
	// ctxCount 请求计数器
	ctxCount int64
	// runtimeStatus 运行时的一些状态，包括运行开始时间，持续时间，引擎的状态
	runtimeStatus *RuntimeStatus
	// components 引擎核心组件,包括去重、请求队列、限速器、指标统计组件、时间监听器
	components ComponentInterface
	// onceClose 引擎关闭动作只执行一次
	onceClose sync.Once
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
func (e *CrawlEngine) startSpider(spider SpiderInterface) {
	defer func() {
		if p := recover(); p != nil {
			if strings.Contains(fmt.Sprintf("%s", p), "send on closed channel") {
				engineLog.Warningf("%s", p)
			} else {
				panic(p)
			}
		}
		e.startSpiderFinish = true

	}()
	spider.StartRequest(e.requestsChan)
}

// stop 引擎停止时的动作,
// 汇总stats并设置状态
func (e *CrawlEngine) stop() StatisticInterface {
	defer e.reInit()
	stats := e.components.GetStats().GetAllStats()
	s := Map2String(stats)
	engineLog.Infof(s)
	e.startSpiderFinish = false
	e.isStop = false
	e.runtimeStatus.SetStatus(ON_STOP)
	e.close()
	engineLog.Warning("关闭引擎")
	e.runtimeStatus.SetStopAt(time.Now().Unix())
	return e.components.GetStats()
}

// Start 爬虫启动器
func (e *CrawlEngine) start(spiderName string) {
	e.mutex.Lock()
	e.reInit()
	defer func() {
		if p := recover(); p != nil {
			e.mutex.Unlock()
			panic(p)
		}
		e.mutex.Unlock()
	}()
	spider, err := e.spiders.GetSpider(spiderName)
	if err != nil {
		panic(err.Error())
	}
	e.setCurrentSpider(spider)
	err = e.components.SpiderBeforeStart(e, spider)
	if err != nil {
		engineLog.Errorf("SpiderBeforeStart ERROR %s", err.Error())
		return
	}
	e.runtimeStatus.SetStartAt(time.Now().Unix())
	e.runtimeStatus.SetRestartAt(time.Now().Unix())
	// 引入引擎所有的组件
	e.eventsChan <- START
	tasks := []GoFunc{e.recvRequest, e.Scheduler}
	e.runtimeStatus.SetStatus(ON_START)
	wg := &conc.WaitGroup{}
	// eventTasks 事件监听器
	eventTasks := []GoFunc{e.EventsWatcherRunner}
	GoRunner(wg, eventTasks...)
	GoRunner(e.waitGroup, tasks...)
	go e.startSpider(spider)

	p := e.waitGroup.WaitAndRecover()
	e.eventsChan <- EXIT
	engineLog.Infof("Wating engine to stop...")
	wg.Wait()
	if p != nil {
		panic(p)
	}

}
func (e *CrawlEngine) Execute(spiderName string) StatisticInterface {
	e.start(spiderName)
	return e.stop()
}

// setCurrentSpider 对相关组件设置当前的spider
func (e *CrawlEngine) setCurrentSpider(spider SpiderInterface) {
	e.components.GetStats().SetCurrentSpider(spider)
	e.components.GetQueue().SetCurrentSpider(spider)
	e.components.GetDupefilter().SetCurrentSpider(spider)
	e.components.GetEventHooks().SetCurrentSpider(spider)
	e.components.GetLimiter().SetCurrentSpider(spider)
	e.components.SetCurrentSpider(spider)
	e.currentSpider = spider
}

// EventsWatcherRunner 事件监听器运行组件
func (e *CrawlEngine) EventsWatcherRunner() error {
	err := e.components.GetEventHooks().EventsWatcher(e.eventsChan)
	if err != nil {
		return fmt.Errorf("events watcher task execution error %s", err.Error())
	}
	return nil
}

// Scheduler 调度器
func (e *CrawlEngine) Scheduler() error {
	for {
		if e.isStop || e.GetRuntimeStatus().GetStatusOn() == ON_STOP {
			return nil
		}
		if e.runtimeStatus.GetStatusOn() == ON_PAUSE {
			e.runtimeStatus.oncePause.Do(func() {
				e.eventsChan <- PAUSE
			})
			e.eventsChan <- HEARTBEAT
			time.Sleep(time.Second * 3)
			runtime.Gosched()
			continue
		}
		// 从缓存队列中读取请求对象
		req, err := e.components.GetQueue().Dequeue()
		if err != nil {
			time.Sleep(time.Second * 3)
			runtime.Gosched()
			continue
		}
		atomic.AddInt64(&e.ctxCount, 1)
		request := req.(*Context)
		// 对request进行处理
		f := []GoFunc{e.worker(request)}
		duration := time.Since(time.Unix(e.runtimeStatus.GetRestartAt(), 0)) / time.Second
		e.runtimeStatus.SetDuration(float64(duration))
		engineLog.Infof("%d:运行时长:%d", e.runtimeStatus.GetRestartAt(), duration)
		GoRunner(e.waitGroup, f...)

	}
}

// doWorkerUnit 依次执行工作单元
func (e *CrawlEngine) doWorkerUnit(ctx *Context, units ...workerUnit) {
	for _, unit := range units {
		err := unit(ctx)
		// 出现异常
		if err != nil || ctx.Error != nil {
			if ctx.Error == nil {
				// 若上下文的error没有构造则在此处构造该异常
				ctx.Error = NewError(ctx, err)
			}
			break
		}
	}
}

// worker request处理器，单个context生成周期都基于此
// 该函数包含了整个request和context的处理过程
func (e *CrawlEngine) worker(ctx *Context) GoFunc {
	c := ctx
	return func() error {
		defer func() {
			atomic.AddInt64(&e.ctxCount, -1)

			if c.Error != nil {
				// 新增一个错误
				e.components.GetStats().Incr(ErrorStats)
				// 提交一个错误事件
				e.eventsChan <- ERROR
				// 将错误交给自定义的spider错误处理函数
				c.Spider.ErrorHandler(c, e.requestsChan)
			}
			
			c.Close()
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
		GoRunner(wg, funcs...)
		// 处理request的所有工作单元
		units := []workerUnit{e.tryLimiterUint, e.doDownload, e.doHandleResponse, e.doParse}
		e.doWorkerUnit(c, units...)
		close(ctx.Items)
		wg.Wait()
		return nil
	}
}

// recvRequest 从requestsChan读取context对象
func (e *CrawlEngine) recvRequest() error {
	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()

	for {
		select {
		case req := <-e.requestsChan:
			if req == nil {
				continue
			}
			runtimeStatus := e.GetRuntimeStatus().GetStatusOn()
			if runtimeStatus == ON_STOP {
				logger.Warnf("准备停止爬虫")
				e.isStop = true
				return nil
			}
			// 新增一条请求
			atomic.AddInt64(&e.ctxCount, 1)
			err := e.writeCache(req)
			if err != nil {
				engineLog.WithField("request_id", req.CtxID).Errorf("请求入队列失败")
			}
		case <-ticker.C:
			// 被动等待爬虫停止或主动停止爬虫
			engineLog.Infof("当前运行状态:%s", e.runtimeStatus.GetStatusOn().GetTypeName())
			if e.checkReadyDone() || e.runtimeStatus.GetStatusOn() == ON_STOP {
				e.isStop = true
				engineLog.Warningf("停止接收请求")
				return nil
			}
		}
	}
}

// checkReadyDone 检查任务状态
// 主要检查spider.StartRequest是否已经执行完成
// 所有的context是否都已经关闭
// 队列是否为空
func (e *CrawlEngine) checkReadyDone() bool {
	// fmt.Printf("当前请求计数:%d\n", atomic.LoadInt64(&e.ctxCount))
	// fmt.Printf("当前队列大小:%d\n", e.components.GetQueue().GetSize())
	// fmt.Printf("当前运行状态:%s\n", e.startSpiderFinish)
	return e.startSpiderFinish && atomic.LoadInt64(&e.ctxCount) == 0 && e.components.CheckWorkersStop()
}

// writeCache 将Context 写入缓存
func (e *CrawlEngine) writeCache(ctx *Context) error {
	defer func() {
		if err := recover(); err != nil {
			ctx.setError(fmt.Sprintf("write cache error %s", err), string(debug.Stack()))
		}
		// 写入缓存后该ctx 生命周期结束，计数器-1
		atomic.AddInt64(&e.ctxCount, -1)
	}()
	var err error = nil
	// 是否进入去重流程
	if e.filterDuplicateReq && !ctx.Request.DoNotFilter {
		ret, err := e.components.GetDupefilter().DoDupeFilter(ctx)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxID).Errorf("request unique error %s", err.Error())
			return err
		}
		// request重复则直接返回
		if ret {
			return nil
		}
	}
	// ctx进入缓存队列
	err = e.components.GetQueue().Enqueue(ctx)
	if err != nil {
		engineLog.Errorf("进入队列失败:%s", err.Error())
		engineLog.WithField("request_id", ctx.CtxID).Warnf("Cache enqueue error %s", err.Error())
		time.Sleep(time.Second)
		return err
	}
	return nil

}

// doDownloaderMiddlewares 通过下载中间件对request进行处理,
// 按优先级调用ProcessRequest
func (e *CrawlEngine) downloaderMiddlewaresUint(ctx *Context) error {

	for _, middleware := range e.downloaderMiddlewares {
		err := middleware.ProcessRequest(ctx)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxID).Errorf("Middleware %s handle request error %s", middleware.GetName(), err.Error())
			return err
		}
	}
	return nil
}
func (e *CrawlEngine) tryLimiterUint(ctx *Context) error {
	// 执行限速器，速率过高则等待
	return e.components.GetLimiter().CheckAndWaitLimiterPass()
}
func (e *CrawlEngine) downloadUnit(ctx *Context) error {
	if ctx.Request.Skip {
		return nil
	}
	// 增加请求发送量
	e.components.GetStats().Incr(RequestStats)
	engineLog.WithField("request_id", ctx.CtxID).Infof("%s request ready to download", ctx.CtxID)
	rsp, err := e.downloader.Download(ctx)
	if rsp != nil {
		e.components.GetStats().Incr(strconv.Itoa(rsp.Status))
	}
	if err != nil {
		return err
	}
	ctx.setResponse(rsp)
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
			e.components.GetStats().Incr("download_fail")
		}
	}()
	// 处理request的所有工作单元
	units := []workerUnit{e.downloaderMiddlewaresUint, e.downloadUnit}
	e.doWorkerUnit(ctx, units...)
	err = ctx.Error
	return err
}

// doHandleResponse 处理请求响应
func (e *CrawlEngine) doHandleResponse(ctx *Context) error {
	defer func() {
		if p := recover(); p != nil {
			ctx.setError(fmt.Sprintf("handle response error %s", p), string(debug.Stack()))
		}
	}()
	if ctx.Request.Skip {
		return nil
	}
	// 检查response的值
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
	// 不需要进入下载中间件
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

// doParse 基于spider实例的解析函数名称通过反射调用解析逻辑
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
		parserErr = rets[0].Interface().(error)
	}
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
		e.components.GetStats().Incr(ItemsStats)
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

// close 关闭引擎
func (e *CrawlEngine) close() {
	e.onceClose.Do(func() {
		// 保证channel只关闭一次
		close(e.requestsChan)
		close(e.eventsChan)
	})
}

// GetStatic 获取StatisticInterface 统计指标
func (e *CrawlEngine) GetStatic() StatisticInterface {
	return e.components.GetStats()
}

// GetCurrentSpider 获取当前正在运行的spider
func (e *CrawlEngine) GetCurrentSpider() SpiderInterface {
	return e.currentSpider
}

func (e *CrawlEngine) GetRuntimeStatus() *RuntimeStatus {
	return e.runtimeStatus
}
func (e *CrawlEngine) GetComponents() ComponentInterface {
	return e.components
}
func (e *CrawlEngine) reInit() {
	e.requestsChan = make(chan *Context, e.reqChannelSize)
	e.eventsChan = make(chan EventType, 16)
	e.ctxCount = 0
}

// NewEngine 构建新的引擎
func NewEngine(opts ...EngineOption) *CrawlEngine {
	Engine := &CrawlEngine{
		waitGroup:             &conc.WaitGroup{},
		spiders:               NewSpiders(),
		pipelines:             make(ItemPipelines, 0),
		downloaderMiddlewares: make(Middlewares, 0),
		requestsChan:          make(chan *Context, 1024),
		eventsChan:            make(chan EventType),
		filterDuplicateReq:    true,
		isStop:                false,
		downloader:            NewDownloader(),
		runtimeStatus:         NewRuntimeStatus(),
		currentSpider:         nil,
		ctxCount:              0,
		reqChannelSize:        1024,
		onceClose:             sync.Once{},
		components:            NewDefaultComponents(),
	}
	for _, o := range opts {
		o(Engine)
	}
	return Engine
}
