#### 事件监听组件
- tegenaria 提供了如下的事件定义:  

     - START 启动事件,启动爬虫是触发 

     - HEARTBEAT 心跳由调度器主动触发  

     - PAUSE 暂停事件，在暂停时触发  

     - ERROR 错误事件,捕获错误时触发  

     - EXIT 引擎退出前触发

- 事件监听接口定义如下:
```go
// EventHooksInterface 事件处理函数接口
type EventHooksInterface interface {
	// Start 处理引擎启动事件
	Start(params ...interface{}) error
	// Stop 处理引擎停止事件
	Pause(params ...interface{}) error
	// Error处理错误事件
	Error(params ...interface{}) error
	// Exit 退出引擎事件
	Exit(params ...interface{}) error
	// Heartbeat 心跳检查事件
	Heartbeat(params ...interface{}) error
	// EventsWatcher 事件监听器
	EventsWatcher(ch chan EventType) error

	SetCurrentSpider(spider SpiderInterface)
}
```

#### 默认的事件监听器

```go
type DefaultHooks struct {
	spider SpiderInterface
}

// NewDefaultHooks 构建新的默认事件监听器
func NewDefaultHooks() *DefaultHooks {
	return &DefaultHooks{}
}
```
目前自带的事件监听器没有实现对事件的处理，用户可以根据需求实现需要的事件监听模块用于处理相关的事件