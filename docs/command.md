## 命令行控制  

- 使用命令行启动爬虫的方式非常简单，只需引入[command](../command/command.go)即可，参照如下的代码：

```go
package main

import (
	_ "net/http/pprof"

	"github.com/wetrycode/tegenaria"
	"github.com/wetrycode/tegenaria/command"
	"github.com/wetrycode/tegenaria/example/quotes"
)

func main() {
	opts := []tegenaria.EngineOption{tegenaria.EngineWithUniqueReq(false)}
	engine := quotes.NewQuotesEngine(opts...)
	command.ExecuteCmd(engine)
}
```

- 启动example爬虫  
```shell
go run main.go crawl example
```
