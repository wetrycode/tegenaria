package main

import (
	"github.com/wetrycode/tegenaria"
	"github.com/wetrycode/tegenaria/example/quotes"
)

func main() {
	config:=tegenaria.NewDistributedWorkerConfig("","",0)
	woker:=tegenaria.NewDistributedWorker("127.0.0.1:6379", config)
	opts := []tegenaria.EngineOption{tegenaria.EngineWithUniqueReq(false), tegenaria.EngineWithDistributedWorker(woker)}
	engine := quotes.NewQuotesEngine(opts...)
	engine.Start("example")
}
