package main

import (
	_ "net/http/pprof"

	"github.com/wetrycode/tegenaria"
	"github.com/wetrycode/tegenaria/example/quotes"
)

func main() {
	opts:= []tegenaria.EngineOption{tegenaria.EngineWithUniqueReq(true), tegenaria.EngineWithUniqueReq(false), tegenaria.EngineWithLimiter(tegenaria.NewDefaultLimiter(128))}
	engine:=quotes.NewQuotesEngine(opts...)
	engine.Start("example")
}
