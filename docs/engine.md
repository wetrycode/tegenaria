#### Engine
引擎是tegenaria系统的调度核心，负责整个抓取调度流程的调度
##### 定义
```go
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
func NewEngine(opts ...EngineOption) *CrawlEngine {
	Engine := &CrawlEngine{
		waitGroup:             &conc.WaitGroup{},
		spiders:               NewSpiders(),
		pipelines:             make(ItemPipelines, 0),
		downloaderMiddlewares: make(Middlewares, 0),
		requestsChan:          make(chan *Context, 1024),
		eventsChan:            make(chan EventType, 16),
		filterDuplicateReq:    true,
		isStop:                false,
		downloader:            NewDownloader(),
		runtimeStatus:         NewRuntimeStatus(),
		currentSpider:         nil,
		ctxCount:              0,
		reqChannelSize:        1024,
		components:            NewDefaultComponents(),
		onceClose:             sync.Once{},

	}
	for _, o := range opts {
		o(Engine)
	}
	return Engine
}
```

#### 说明  

- 可以看到引擎包含了前文提及到的说有组件，引擎负责将这些组件进行组合构成一个完整的调度链路 

- 引擎提供了运行时状态控制和查询的组件`RuntimeStatus`,该组件可以控制和查询爬虫的运行状态，其定义如下

```go
// RuntimeStatus 引擎状态控制和查询
type RuntimeStatus struct {
	// StartAt 第一次启动时间
	StartAt int64
	// Duration 运行时长
	Duration float64
	// StopAt 停止时间
	StopAt int64
	// RestartAt 重启时间
	RestartAt int64
	// StatusOn 当前引擎的状态
	StatusOn StatusType
	// onceStart 启动状态只执行一次
	onceStart sync.Once
	// oncePause 暂停状态只触发一次
	oncePause *sync.Once
}

func NewRuntimeStatus() *RuntimeStatus {
	return &RuntimeStatus{
		StartAt:   0,
		Duration:  0,
		StopAt:    0,
		RestartAt: 0,
		StatusOn:  ON_STOP,
		onceStart: sync.Once{},
		oncePause: &sync.Once{},
	}
}
```