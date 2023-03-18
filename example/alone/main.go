package main

// import (
// 	_ "net/http/pprof"

// 	"github.com/wetrycode/tegenaria"
// 	"github.com/wetrycode/tegenaria/example/quotes"
// )

// func main() {
// 	opts := []tegenaria.EngineOption{tegenaria.EngineWithUniqueReq(false)}
// 	engine := quotes.NewQuotesEngine(opts...)
// 	engine.Execute("example")
// }

import (
	"fmt"
	"net/url"
)

func main() {
	// Example Flux query with parameters
	query := "from(bucket:\"mybucket\") |> range(start: 0, stop: 10m)"

	// Parse the query into a URL object
	u, err := url.Parse("http://localhost/query?" + query)
	if err != nil {
		fmt.Println("Error parsing query:", err)
		return
	}

	// Get the values of the "start" and "stop" parameters
	start := u.Query().Get("start")
	stop := u.Query().Get("stop")

	fmt.Println("start:", start)
	fmt.Println("stop:", stop)
}
