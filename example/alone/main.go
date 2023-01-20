package main

import (
	_ "net/http/pprof"

	"github.com/wetrycode/tegenaria"
	"github.com/wetrycode/tegenaria/example/quotes"
)

func main() {
	opts := []tegenaria.EngineOption{tegenaria.EngineWithUniqueReq(false), tegenaria.EngineWithLimiter(tegenaria.NewDefaultLimiter(64))}
	engine := quotes.NewQuotesEngine(opts...)
	engine.Execute()
}
