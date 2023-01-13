package main

import (
	"log"
	"net/http"
	"runtime"

	_ "net/http/pprof"

	"github.com/wetrycode/tegenaria/example/quotes"
)

func main() {
	runtime.SetMutexProfileFraction(1) // 开启对锁调用的跟踪
	runtime.SetBlockProfileRate(1)     // 开启对阻塞操作的跟踪

	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:8005", nil))
	}()
	quotes.Start()
}
