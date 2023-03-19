package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime/debug"
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
	"google.golang.org/protobuf/types/known/structpb"
)

var logger = tegenaria.GetLogger("service")

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
	defer func() {
		if p := recover(); p != nil {
			logger.Errorf("get engine status fail %s %s", p, debug.Stack())
		}
	}()
	runtimeStatus := s.Engine.GetRuntimeStatus()

	stats := s.Engine.GetStatic()
	current_spider := ""
	if s.Engine.GetCurrentSpider() != nil {
		current_spider = s.Engine.GetCurrentSpider().GetName()
	}
	start_at := ""
	if runtimeStatus.GetStartAt() != 0 {
		start_at = time.Unix(runtimeStatus.GetStartAt(), 0).Format("2006-01-02 15:04:05")
	}
	stop_at := ""
	if runtimeStatus.GetStopAt() != 0 {
		start_at = time.Unix(runtimeStatus.GetStartAt(), 0).Format("2006-01-02 15:04:05")
	}
	info := &pb.TegenariaStatusMessage{
		Status:     runtimeStatus.GetStatusOn().GetTypeName(),
		StartAt:    start_at,
		StopAt:     stop_at,
		Duration:   runtimeStatus.GetDuration(),
		Metrics:    stats.GetAllStats(),
		SpiderName: current_spider,
	}
	jsonInfo, _ := json.Marshal(info)
	var m map[string]interface{}
	_ = json.Unmarshal(jsonInfo, &m)
	value, err := structpb.NewValue(m)
	if err != nil {
		logger.Errorf("get status error %s", err.Error())
	}
	rspStruct := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"data": value,
		},
	}
	return &pb.ResponseMessage{
		Code: int32(pb.ResponseStatus_OK.Number()),
		Msg:  "ok",
		Data: rspStruct,
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
	IP, _ := tegenaria.GetMachineIP()
	fmt.Printf("Server listen on:http://%s:%d\n", s.Host, s.Port)
	fmt.Printf("Server listen on:http://%s:%d\n", IP, s.Port)

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
