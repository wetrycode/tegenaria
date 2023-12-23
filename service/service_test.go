package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"github.com/wetrycode/tegenaria"
	"github.com/wetrycode/tegenaria/service/pb"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

func NewTestClient() pb.TegenariaServiceClient {
	conn, err := grpc.Dial("127.0.0.1:19527", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	// defer conn.Close()
	c := pb.NewTegenariaServiceClient(conn)
	return c
}
func TestSetStatusWithRPC(t *testing.T) {
	convey.Convey("test set status RPC  api", t, func() {
		engine := tegenaria.NewTestEngine("example")

		server := NewServer(engine, "127.0.0.1", 19527)
		patch := gomonkey.ApplyFunc((*tegenaria.DefaultComponents).CheckWorkersStop, func(_ *tegenaria.DefaultComponents) bool {
			return false

		})
		defer patch.Reset()
		go func() {
			server.Start()
		}()
		// 启动example爬虫
		time.Sleep(time.Second * 2)
		client := NewTestClient()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		r, err := client.SetStatus(ctx, &pb.StatusContorlRequest{
			Status:     pb.TegenariaStatus_ON_START,
			SpiderName: "example",
		})
		time.Sleep(time.Second * 6)
		convey.So(err, convey.ShouldBeNil)
		convey.So(r.Code, convey.ShouldAlmostEqual, pb.ResponseStatus_OK)
		// 查询状态
		r, err = client.GetStatus(context.TODO(), &emptypb.Empty{})
		convey.So(err, convey.ShouldBeNil)
		convey.So(r.Code, convey.ShouldAlmostEqual, pb.ResponseStatus_OK)
		// 获取状态
		convey.So(r.Data, convey.ShouldNotBeNil)
		var status = &pb.TegenariaStatusMessage{}
		data, err := r.Data.GetFields()["data"].MarshalJSON()
		convey.So(err, convey.ShouldBeNil)

		err = json.Unmarshal(data, status)
		convey.So(err, convey.ShouldBeNil)
		convey.So(status.Metrics[tegenaria.RequestStats], convey.ShouldAlmostEqual, 1)
		convey.So(status.Status, convey.ShouldContainSubstring, tegenaria.ON_START.GetTypeName())
		// 暂停
		r, err = client.SetStatus(context.TODO(), &pb.StatusContorlRequest{
			Status:     pb.TegenariaStatus_ON_PAUSE,
			SpiderName: "example",
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(r.Code, convey.ShouldAlmostEqual, pb.ResponseStatus_OK)

		r, err = client.GetStatus(context.TODO(), &emptypb.Empty{})
		convey.So(err, convey.ShouldBeNil)
		convey.So(r.Code, convey.ShouldAlmostEqual, pb.ResponseStatus_OK)
		// 获取状态
		convey.So(r.Data, convey.ShouldNotBeNil)
		var statusPause = &pb.TegenariaStatusMessage{}
		data, err = r.Data.GetFields()["data"].MarshalJSON()
		convey.So(err, convey.ShouldBeNil)
		err = json.Unmarshal(data, statusPause)

		convey.So(err, convey.ShouldBeNil)
		convey.So(statusPause.Metrics["requests"], convey.ShouldAlmostEqual, 1)
		convey.So(statusPause.Status, convey.ShouldContainSubstring, tegenaria.ON_PAUSE.GetTypeName())

		ctx, cancel = context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		r, err = client.SetStatus(ctx, &pb.StatusContorlRequest{
			Status:     pb.TegenariaStatus_ON_START,
			SpiderName: "example_2",
		})
		time.Sleep(time.Second * 5)
		convey.So(err, convey.ShouldBeNil)
		convey.So(r, convey.ShouldNotBeNil)
		convey.So(r.Code, convey.ShouldAlmostEqual, pb.ResponseStatus_NOT_FOUND_SPIDER)

	})

}

func TestSetStatusWithHTTP(t *testing.T) {
	convey.Convey("test set status HTTP api", t, func() {
		engine := tegenaria.NewTestEngine("example2")

		server := NewServer(engine, "127.0.0.1", 12138)

		go func() {
			server.Start()
		}()
		time.Sleep(time.Second * 2)
		reqBody := map[string]interface{}{
			"Status":     0,
			"SpiderName": "example2",
		}
		transport := &http2.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			AllowHTTP:       true,
			DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(netw, addr)
			},
		}
		client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

		// 启动爬虫
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		var jsonStr, _ = json.Marshal(reqBody)
		req, err := http.NewRequestWithContext(ctx, "POST", "http://127.0.0.1:12138/api/v1/tegenaria/status", bytes.NewReader(jsonStr))
		req.Header.Set("Accept", "application/json")

		convey.So(err, convey.ShouldBeNil)
		rsp, err := client.Do(req)
		convey.So(err, convey.ShouldBeNil)
		convey.So(rsp.StatusCode, convey.ShouldAlmostEqual, 200)

		defer rsp.Body.Close()
		body, err := io.ReadAll(rsp.Body)
		convey.So(err, convey.ShouldBeNil)
		var message = &pb.ResponseMessage{}
		err = json.Unmarshal(body, message)
		convey.So(err, convey.ShouldBeNil)
		convey.So(message.Code, convey.ShouldAlmostEqual, pb.ResponseStatus_OK)
		time.Sleep(time.Second * 5)

		// 获取爬虫状态
		statusRequest, _ := http.NewRequest("GET", "http://127.0.0.1:12138/api/v1/tegenaria/status", nil)
		statusRequest.Header.Set("Accept", "application/json")

		statusRsp, err := client.Do(statusRequest)
		convey.So(statusRsp.StatusCode, convey.ShouldAlmostEqual, 200)

		convey.So(err, convey.ShouldBeNil)
		var statusMessage = &pb.ResponseMessage{}
		body, err = io.ReadAll(statusRsp.Body)
		convey.So(err, convey.ShouldBeNil)

		err = json.Unmarshal(body, statusMessage)
		convey.So(err, convey.ShouldBeNil)
		convey.So(statusMessage.Code, convey.ShouldAlmostEqual, pb.ResponseStatus_OK)
		convey.So(err, convey.ShouldBeNil)
		var engineStatus = &pb.TegenariaStatusMessage{}
		data, err := statusMessage.Data.GetFields()["data"].MarshalJSON()
		convey.So(err, convey.ShouldBeNil)
		err = json.Unmarshal(data, engineStatus)

		convey.So(err, convey.ShouldBeNil)
		convey.So(engineStatus.Metrics["requests"], convey.ShouldAlmostEqual, 1)
		convey.So(engineStatus.Status, convey.ShouldContainSubstring, tegenaria.ON_STOP.GetTypeName())
		defer statusRsp.Body.Close()
	})

}

func TestStartError(t *testing.T) {
	convey.Convey("test spider not found HTTP api", t, func() {
		engine := tegenaria.NewTestEngine("example3")

		server := NewServer(engine, "127.0.0.1", 12139)

		go func() {
			server.Start()
		}()
		time.Sleep(time.Second * 2)
		reqBody := map[string]interface{}{
			"Status":     0,
			"SpiderName": "example4",
		}
		transport := &http2.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			AllowHTTP:       true,
			DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(netw, addr)
			},
		}
		client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

		// 启动爬虫
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		var jsonStr, _ = json.Marshal(reqBody)
		req, err := http.NewRequestWithContext(ctx, "POST", "http://127.0.0.1:12139/api/v1/tegenaria/status", bytes.NewReader(jsonStr))
		req.Header.Set("Accept", "application/json")

		convey.So(err, convey.ShouldBeNil)
		rsp, err := client.Do(req)
		convey.So(err, convey.ShouldBeNil)
		convey.So(rsp.StatusCode, convey.ShouldAlmostEqual, 200)
		var statusMessage = &pb.ResponseMessage{}
		body, err := io.ReadAll(rsp.Body)
		convey.So(err, convey.ShouldBeNil)
		err = json.Unmarshal(body, statusMessage)
		convey.So(err, convey.ShouldBeNil)
		convey.So(statusMessage.Code, convey.ShouldAlmostEqual, pb.ResponseStatus_NOT_FOUND_SPIDER)
	})

	convey.Convey("test spider start error", t, func() {
		engine := tegenaria.NewTestEngine("example5")

		server := NewServer(engine, "127.0.0.1", 12119)

		go func() {
			server.Start()
		}()
		time.Sleep(time.Second * 2)
		reqBody := map[string]interface{}{
			"Status":     0,
			"SpiderName": "example5",
		}
		transport := &http2.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			AllowHTTP:       true,
			DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(netw, addr)
			},
		}
		client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

		// 启动爬虫
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		patch := gomonkey.ApplyFunc((*tegenaria.DefaultComponents).SpiderBeforeStart, func(_ *tegenaria.DefaultComponents, _ *tegenaria.CrawlEngine, _ tegenaria.SpiderInterface) error {
			panic("SpiderBeforeStart panic")

		})
		defer patch.Reset()
		var jsonStr, _ = json.Marshal(reqBody)
		req, err := http.NewRequestWithContext(ctx, "POST", "http://127.0.0.1:12119/api/v1/tegenaria/status", bytes.NewReader(jsonStr))
		req.Header.Set("Accept", "application/json")

		convey.So(err, convey.ShouldBeNil)
		rsp, err := client.Do(req)
		convey.So(err, convey.ShouldBeNil)
		convey.So(rsp.StatusCode, convey.ShouldAlmostEqual, 200)
		var statusMessage = &pb.ResponseMessage{}
		body, err := io.ReadAll(rsp.Body)
		convey.So(err, convey.ShouldBeNil)
		err = json.Unmarshal(body, statusMessage)
		convey.So(err, convey.ShouldBeNil)

		convey.So(statusMessage.Code, convey.ShouldAlmostEqual, pb.ResponseStatus_UNKNOWN)
		convey.So(statusMessage.Msg, convey.ShouldContainSubstring, "start err")

	})

}
