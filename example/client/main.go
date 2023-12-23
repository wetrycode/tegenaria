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
