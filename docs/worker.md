#### 分布式woker接口
tegenaria提供了用于检查和监听分布式节点状态的接口```DistributedWorkerInterface```  
接口定义如下：
```go
type DistributedWorkerInterface interface {

	// AddNode 新增一个节点
	AddNode() error
	// DelNode 删除当前的节点
	DelNode() error
	// PauseNode 停止当前的节点
	PauseNode() error
	// Heartbeat 心跳
	Heartbeat() error
	// CheckAllNodesStop 检查所有的节点是否都已经停止
	CheckAllNodesStop() (bool, error)
	// CheckMasterLive 检测主节点是否还在线
	CheckMasterLive() (bool, error)
	// SetMaster 是否将当前的节点设置为主节点
	SetMaster(flag bool)
	// SetCurrentSpider 设置当前的spider
	SetCurrentSpider(spider SpiderInterface)
	// GetWorkerID 当前工作节点的id
	GetWorkerID() string
	// IsMaster 是否是主节点
	IsMaster() bool
}
```

#### 说明

- ```AddNode() error``` 新增提个节点

- ```DelNode() error``` 删除一个节点

- ```PauseNode() error``` 节点任务暂停处理

- ```Heartbeat() error``` 心跳检测

- ```CheckAllNodesStop() (bool, error)``` 检查所有的节点是否都已经停止

- ```CheckMasterLive() (bool, error)```   是否有检查主节点在线

- ```SetMaster(flag bool)``` 设置节点角色，如果是主节点则设置为`true`,从节点设置为`false`  

- ```SetCurrentSpider(spider SpiderInterface)``` 设置当前运行的爬虫实例

- ```GetWorkerID() string```当前节点的标识  

- ```IsMaster() bool```判断节点的角色

#### 默认的分布式worker实现

tegenaria 提供了一个分布式场景下的woker接口实现

- ##### 定义:
```go
type DistributedWorker struct {
	// rdb redis客户端支持redis单机实例和redis cluster集群模式
	rdb redis.Cmdable

	// nodeID 节点id
	nodeID string
	// nodesSetPrefix 节点池的key前缀
	// 默认为"tegenaria:v1:nodes"
	nodesSetPrefix string
	// masterNodesKey master节点池的key前缀
	// 默认值为"tegenaria:v1:master"
	masterNodesKey string
	// nodePrefix 节点前缀
	nodePrefix string
	// currentSpider 当前的spider
	currentSpider string
	// isMaster 是否是master节点
	isMaster bool
	// ip 本机地址
	ip string
}
// NewDistributedWorker 构建redis单机模式下的分布式工作组件
func NewDistributedWorker(config *DistributedWorkerConfig) *DistributedWorker {
	rdb := NewRdbClient(config)
	ip, err := tegenaria.GetMachineIP()
	if err != nil {
		panic(fmt.Sprintf("get machine ip error %s", err.Error()))
	}
	d := &DistributedWorker{
		rdb:            rdb,
		nodeID:         tegenaria.GetEngineID(),
		ip:             ip,
		nodesSetPrefix: "tegenaria:v1:nodes",
		nodePrefix:     "tegenaria:v1:node",
		masterNodesKey: "tegenaria:v1:master",
		isMaster:       true,
	}
	return d
}
// NewWorkerWithRdbCluster redis cluster模式下的分布式工作组件
func NewWorkerWithRdbCluster(config *WorkerConfigWithRdbCluster) *DistributedWorker {
	w := NewDistributedWorker(config.DistributedWorkerConfig)
	// 替换为redis cluster客户端
	w.rdb = NewRdbClusterCLient(config)
	return w
}
```

##### 参数说明：

 - `rdb` redis客户端实例，目前支持redis单机和redis cluster模式下的实例，默认采用单机模式  
 - `nodeID`当前节点的id，生成方式如下:
 ```go
 	engineID = MD5(GetUUID())[0:16]
 ```
 - `nodesSetPrefix` 节点池前缀，用于缓存所有节点的id,默认值为`tegenaria:v1:nodes`,节点池key的结构为`{prefix}:{spider_name}`,例如`tegenaria:v1:nodes:example`
 - `masterNodesKey` master节点池前缀，用于缓存所有的master节点id,默认值为`tegenaria:v1:master`,master节点池的结构为`{prefix}:{spider_name}`, 例如`tegenaria:v1:master:example`
 - `nodePrefix` 节点状态key前缀，用于缓存节点的状态,默认值为`tegenaria:v1:node`,节点状态key的结构为`{nodePrefix}:{spiderName}:{ip}:{nodeID}`,例如`tegenaria:v1:node:example:127.0.0.1:532553557ca33be4`
 - `currentSpider` 当前正在运行的spider实例
 - `ip` 当前节点的ip
 - `isMaster`是否是主节点，用于标记节点角色
 
##### 应用

- 该模块和分布式事件监听器一起使用，可以实现自动化的新增、删除、暂停节点及触发心跳

- 包含如下的几个动作：

	- `AddNode` 响应节点池添加一个节点 

	- `DelNode`从节点池(包含master节点池)中删除某一个节点

	- `PauseNode`将节点状态设置为暂停,维护其状态的key为`{node_key}:pause`,例如`tegenaria:v1:node:example:127.0.0.1:532553557ca33be4:pause`

	- `Heartbeat`维护节点的存活状态

#### redis cluster模式

- woker实例对象支持redis cluster 模式的实例，需要通过`NewWorkerWithRdbCluster`配合`WorkerConfigWithRdbCluster`创建实例

- `WorkerConfigWithRdbCluster`定义:

```go
// WorkerConfigWithRdbCluster redis cluster 模式下的分布式组件配置参数
type WorkerConfigWithRdbCluster struct {
	*DistributedWorkerConfig
	RdbNodes
}
// NewWorkerConfigWithRdbCluster redis cluster模式的分布式组件
func NewWorkerConfigWithRdbCluster(config *DistributedWorkerConfig, nodes RdbNodes) *WorkerConfigWithRdbCluster {
	newConfig := &WorkerConfigWithRdbCluster{
		DistributedWorkerConfig: config,
		RdbNodes:                nodes,
	}
	return newConfig
}
```

- `WorkerConfigWithRdbCluster` 继承了`DistributedWorkerConfig`，新增了一个redis clutser节点列表的配置`RdbNodes`

- 在构建`WorkerConfigWithRdbCluster`实例对象时传入额外的redis cluster 节点列表即可

- 示例
```go
	config := NewDistributedWorkerConfig(NewRedisConfig("", "", "", 0), NewInfluxdbConfig("http://127.0.0.1:9086", "xxxx", "test", "distributed"))
	worker := NewWorkerWithRdbCluster(NewWorkerConfigWithRdbCluster(config, nodes))
```
