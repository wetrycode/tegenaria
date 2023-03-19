#### RPC服务

Tegenaria同时提供了http和gRPC协议的远程控制接口，该服务基于gRPC和[grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway)实现,   
可以同时在同一端口上提供http和gRPC服务

#### 引入rpc服务  
```go
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
```  

#### HTTP请求接口

- 启动爬虫  

```shell
curl --location 'http://127.0.0.1:9527/api/v1/tegenaria/status' \
--header 'Content-Type: application/json' \
--data '{
    "status":0,
    "spider_name":"example"
}'
```

- 获取爬虫状态

```shell
curl --location 'http://127.0.0.1:9527/api/v1/tegenaria/status'
```

- 暂停爬虫  

```shell
curl --location 'http://127.0.0.1:9527/api/v1/tegenaria/status' \
--header 'Content-Type: application/json' \
--data '{
    "status":2,
    "spider_name":"example"
}'
```  

- 停止爬虫  

```shell
curl --location 'http://127.0.0.1:9527/api/v1/tegenaria/status' \
--header 'Content-Type: application/json' \
--data '{
    "status":1,
    "spider_name":"example"
}'
```

#### gRPC请求

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/wetrycode/tegenaria/service/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

func main() {
	conn, err := grpc.Dial("127.0.0.1:9527", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	// 启动爬虫
	client := pb.NewTegenariaServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	r, err := client.SetStatus(ctx, &pb.StatusContorlRequest{
		Status:     pb.TegenariaStatus_ON_START,
		SpiderName: "example",
	})
	if err != nil {
		fmt.Printf("爬虫启动失败:%s\n", err.Error())
	}
	fmt.Printf("请求状态码:%d", r.Code)
	time.Sleep(time.Second * 5)

	// 查询状态
	r, err = client.GetStatus(context.TODO(), &emptypb.Empty{})
	if err != nil {
		fmt.Printf("爬虫状态查询失败:%s\n", err.Error())
	}
	fmt.Printf("爬虫状态:%v", r.Data.AsMap())

}
```

#### 注意

启动爬虫的接口请求需要设置至少5s的超时时间
