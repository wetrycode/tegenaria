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
