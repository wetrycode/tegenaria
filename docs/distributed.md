## 分布式模式

### 中间件部署  

分布式模式的爬虫需要依赖redis和influxdb这两个中间件，两者的部署在开发环境下可以直接使用docker：
- [redis](https://hub.docker.com/_/redis)   
- [influxdb2](https://hub.docker.com/r/bitnami/influxdb)

### 配置

在项目目录下添加如下的配置```settings.yml```  
```yaml
redis:
  addr: "127.0.0.1:6379"
  username: ""
  password: ""

influxdb:
  host: "127.0.0.1"
  port: 9086
  bucket: "tegenaria"
  org: "wetrycode"
  token: "9kCeULToVdZVdUGboG5qhiWaMiUUV1XKjKMlrFwPisNMKSENDAK3EBsuh_M-GFwGnNpQZkcZ6kPgltrDOHZo5g=="

log:
  path: "/var/logs/Tegenaria"
  level: "info"
```
### 引入分布式组件  
- 代码  

```go
package main

import (
	"github.com/wetrycode/tegenaria"
	"github.com/wetrycode/tegenaria/distributed"
	"github.com/wetrycode/tegenaria/example/quotes"
)

func main() {
	// 工作组件
	components := distributed.NewDefaultDistributedComponents()
	// 分布式组件加入到引擎
	opts := []tegenaria.EngineOption{tegenaria.EngineWithUniqueReq(false), tegenaria.EngineWithComponents(components)}
	engine := quotes.NewQuotesEngine(opts...)
	engine.Execute("example")
}

```  

- 启动  
```shell
go run main.go
```

### 从节点  
- 分布式组件默认启动的是主节点，会同时生产和消费请求，若需要启动从节点则需要在初始分布式组件时指定节点的角色:

```go
distributed.NewDefaultDistributedComponents(DistributedWithSlave())
```

- 请注意，从节点不会调用```StartRequest```函数只会从请求队列中获取队列
