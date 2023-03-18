package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/wetrycode/tegenaria"
	"github.com/wetrycode/tegenaria/service/pb"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	pb.UnimplementedTegenariaServiceServer
	Engine *tegenaria.CrawlEngine
	Host   string
	Port   int
}

func NewServer(engine *tegenaria.CrawlEngine, host string, port int) *Server {
	return &Server{
		Engine: engine,
		Host:   host,
		Port:   port,
	}
}

func (s *Server) SetStatus(ctx context.Context, request *pb.StatusContorlRequest) (*pb.ResponseMessage, error) {
	status := request.Status
	spider := request.SpiderName
	if _, err := s.Engine.GetSpiders().GetSpider(spider); err != nil {
		return &pb.ResponseMessage{
			Code: int32(pb.ResponseStatus_NOT_FOUND_SPIDER.Number()),
			Msg:  fmt.Sprintf("%s spider not found", spider),
			Data: nil,
		}, err
	}
	engineStatus := s.Engine.GetRuntimeStatus().GetStatusOn()
	// 当前没有爬虫在运行并接收到启动指令
	rsp := &pb.ResponseMessage{
		Code: int32(pb.ResponseStatus_OK.Number()),
		Msg:  "ok",
		Data: nil,
	}
	var err error = nil
	// 第一次启动
	if engineStatus.GetTypeName() == tegenaria.ON_STOP.GetTypeName() && tegenaria.StatusType(status.Number()) == tegenaria.ON_START {
		go func() {
			defer func() {
				if p := recover(); p != nil {
					rsp.Code = int32(pb.ResponseStatus_UNKOWN.Number())
					rsp.Msg = "spider start err"
					rsp.Data = nil
					err = errors.New("UNKOWN ERROR")
				}
			}()
			s.Engine.Execute(spider)
		}()
		time.Sleep(time.Second * 5)
		return rsp, err
	}
	s.Engine.GetRuntimeStatus().SetStatus(tegenaria.StatusType(status))
	time.Sleep(time.Second * 1)
	return rsp, err
}
func (s *Server) GetStatus(ctx context.Context, _ *emptypb.Empty) (*pb.ResponseMessage, error) {
	runtimeStatus := s.Engine.GetRuntimeStatus()

	stats := s.Engine.GetStatic()

	info := &pb.TegenariaStatusMessage{
		Status:     runtimeStatus.GetStatusOn().GetTypeName(),
		StartAt:    time.Unix(runtimeStatus.GetStartAt(), 0).Format("2006-01-02 15:04:05"),
		StopAt:     time.Unix(runtimeStatus.GetStopAt(), 0).Format("2006-01-02 15:04:05"),
		Duration:   runtimeStatus.GetDuration(),
		Metrics:    stats.GetAllStats(),
		SpiderName: s.Engine.GetCurrentSpider().GetName(),
	}
	jsonInfo, _ := json.Marshal(info)

	return &pb.ResponseMessage{
		Code: int32(pb.ResponseStatus_OK.Number()),
		Msg:  "ok",
		Data: jsonInfo,
	}, nil
}

func (s *Server) Start() {
	// Create a listener on TCP port
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	// 创建一个gRPC server对象
	g := grpc.NewServer()
	pb.RegisterTegenariaServiceServer(g, s)

	// gRPC-Gateway mux
	gwmux := runtime.NewServeMux()
	dops := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = pb.RegisterTegenariaServiceHandlerFromEndpoint(context.Background(), gwmux, fmt.Sprintf("%s:%d", s.Host, s.Port), dops)
	if err != nil {
		log.Fatalln("Failed to register gwmux:", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", gwmux)

	// 定义HTTP server配置
	gwServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.Host, s.Port),
		Handler: s.grpcHandlerFunc(g, mux), // 请求的统一入口
	}
	log.Fatalln(gwServer.Serve(lis)) // 启动HTTP服务

}
func (s *Server) grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}
