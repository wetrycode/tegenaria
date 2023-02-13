package main

import (
	"github.com/wetrycode/tegenaria"
	"github.com/wetrycode/tegenaria/api"
	"github.com/wetrycode/tegenaria/example/quotes"
)

func main() {
	opts := []tegenaria.EngineOption{tegenaria.EngineWithUniqueReq(false), tegenaria.EngineWithLimiter(tegenaria.NewDefaultLimiter(64))}
	engine := quotes.NewQuotesEngine(opts...)
	api := api.NewAPI(engine)
	api.Server()
}
