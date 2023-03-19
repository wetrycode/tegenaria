#### 消息队列

tegenaria 提供了基于redis实现的消息队列用于缓存`Request`对象，该消息队列实现了`CacheInterface`[接口](queue.md)  
##### 定义
```go
type DistributedQueue struct {
	rdb redis.Cmdable
	// getQueueKey 生成队列key的函数，允许用户自定义
	getQueueKey GetRDBKey
	// currentSpider 当前的spider
	currentSpider string
}
func NewDistributedQueue(rdb redis.Cmdable, queueKey GetRDBKey) *DistributedQueue {
	return &DistributedQueue{
		rdb:         rdb,
		getQueueKey: queueKey,
	}
}
```
##### 参数说明  
- `rdb` redis客户端实例
 - `getQueueKey` 消息队列key前缀生成器，[详细说明](distributed_components.md)

#### 说明

##### `Enqueue`流程

- 通过`Request.ToMap`方法将`Request`对象序列化为`map[string]interface{}`  
- 通过`gob.Encode`将上述的map对象序列化为而进行数据
- 将二进制推入队列

##### `Dequeue`流程  

- 从redis拉取一条二进制记录
- 
- 通过`gob.Decode`将二进制进行解码得到一个`map[string]interface{}`对象  
- 
- 调用`Request.RequestFromMap` 将`map[string]interface{}`对象 反序列化为`Request`对象

