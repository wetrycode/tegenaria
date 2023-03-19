package main

import (
	_ "net/http/pprof"

	"github.com/wetrycode/tegenaria"
	"github.com/wetrycode/tegenaria/example/quotes"
	"github.com/wetrycode/tegenaria/service"
)

func main() {
	opts := []tegenaria.EngineOption{tegenaria.EngineWithUniqueReq(false)}
	engine := quotes.NewQuotesEngine(opts...)
	gRPCService := service.NewServer(engine, "0.0.0.0", 9527)
	gRPCService.Start()
}
