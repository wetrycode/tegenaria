#### 请求队列

请求队列主要是用于缓存请求对象，为什么不直接使用 channel 进行缓存？主要是考虑到以下的几点:

- channel 无法实时获取队列的大小

- channel 扩展性较差，无法满足一些自定义的业务需求，例如分布式缓存

#### 接口说明

- 请求队列的接口定义如下:

```go
type CacheInterface interface {
	// enqueue ctx写入缓存
	Enqueue(ctx *Context) error
	// dequeue ctx 从缓存出队列
	Dequeue() (interface{}, error)
	// isEmpty 缓存是否为空
	IsEmpty() bool
	// getSize 缓存大小
	GetSize() uint64
	// close 关闭缓存
	Close() error
	// SetCurrentSpider 设置当前的spider
	SetCurrentSpider(spider SpiderInterface)
}
```

- 接口实现了队列的一些常用操作包括进出队列及队列大小的查询能力

- 默认的请求队列是通过[无锁队列](https://pkg.go.dev/github.com/yireyun/go-queue)实现

```go
type DefaultQueue struct {
	queue  *queue.EsQueue
	spider SpiderInterface
}
```

- 其构造函数需要指定队列的大小size,默认值1024

```go
// NewDefaultQueue get a new DefaultQueue
func NewDefaultQueue(size int) *DefaultQueue {
	return &DefaultQueue{
		queue: queue.NewQueue(uint32(size)),
	}
}
```
- 如何指定队列大小?  
在构造```DefaultComponents```是传入参数```DefaultComponentsWithDefaultQueue(NewDefaultQueue(1024))```即可
