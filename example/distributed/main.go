package main

import (
	"github.com/wetrycode/tegenaria"
	"github.com/wetrycode/tegenaria/example/quotes"
)

func main() {
	// 构造分布式组件
	config := tegenaria.NewDistributedWorkerConfig("", "", 0)
	woker := tegenaria.NewDistributedWorker("127.0.0.1:6379", config)
	// 分布式组件加入到引擎
	opts := []tegenaria.EngineOption{tegenaria.EngineWithUniqueReq(false), tegenaria.EngineWithDistributedWorker(woker)}
	engine := quotes.NewQuotesEngine(opts...)
	engine.Execute()
}
