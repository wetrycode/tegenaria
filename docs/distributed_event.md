#### 分布式事件监听器

tegenaria 提供了分布式模式下的事件监听组件，主要是用于监听节点的状态，该过模块实现了`EventHooksInterface`[接口](events.md)   
其定义如下:
```go
type DistributedHooks struct {
	worker tegenaria.DistributedWorkerInterface
	spider tegenaria.SpiderInterface
}
// DistributedHooks 构建新的分布式监听器组件对象
func NewDistributedHooks(worker tegenaria.DistributedWorkerInterface) *DistributedHooks {
	return &DistributedHooks{
		worker: worker,
	}

}
```

#### 说明  

##### 关键参数

- `worker`是`DistributedWorkerInterface`实例对象主要是控制节点的状态，关于`DistributedWorkerInterface`接口的说明见[文档](worker.md)  

- `spider` 当前在运行的爬虫实例对象

##### 事件对应的节点worker动作

- `START`->`worker.AddNode`  

- `PAUSE`->`worker.PauseNode`

- `Error`->`什么也不做`

- `Exit`->`worker.DelNode()`
