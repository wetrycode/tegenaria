// MIT License

// Copyright (c) 2023 wetrycode

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package distributed

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/smartystreets/goconvey/convey"
	"github.com/wetrycode/tegenaria"
)

var onceInfluxDBServer sync.Once
var ts *httptest.Server

type PointMocker struct {
	Value float64
	Time  int64
}
type InfluxdbServerMocker struct {
	Points map[string]*PointMocker
}

func (s *InfluxdbServerMocker) WriterInflux(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	bodyString := string(body)
	lines := strings.Split(bodyString, "\n")
	logger.Info(bodyString)
	for _, line := range lines[:len(lines)-1] {
		fields := strings.Split(line, " ")
		logger.Info(line)
		field := strings.Split(fields[1], "=")
		time, _ := strconv.Atoi(fields[2])
		v, _ := strconv.Atoi(strings.Replace(field[1], "i", "", -1))
		s.Points[field[0]] = &PointMocker{
			Value: float64(v),
			Time:  int64(time),
		}

	}
	// Replace the request body so that it can be read again
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	c.Status(204)
}
func (s *InfluxdbServerMocker) QueryInflux(c *gin.Context) {
	// #datatype,string,long,dateTime:RFC3339,dateTime:RFC3339,long,string,string
	// #group,false,false,true,true,false,true,true
	// #default,_result,,,,,,
	// ,result,table,_start,_stop,_value,_field,_measurement
	// ,,0,1970-01-01T00:00:00Z,2023-03-17T09:52:16.339615725Z,9000,items,example
	fields := []string{
		tegenaria.RequestStats,
		tegenaria.ErrorStats,
		tegenaria.DownloadFailStats,
		tegenaria.ItemsStats,
	}
	body, _ := io.ReadAll(c.Request.Body)
	bodyString := string(body)
	rsp := []string{
		"#datatype,string,long,dateTime:RFC3339,dateTime:RFC3339,long,string,string",
		"#group,false,false,true,true,false,true,true",
		"#default,_result,,,,,,",
		",result,table,_start,_stop,_value,_field,_measurement",
	}
	isOk := false
	for _, field := range fields {
		if strings.Contains(bodyString, field) {
			logger.Infof("查询指标:%s", field)
			isOk = true

			if point, ok := s.Points[field]; ok {
				row := []string{
					"", "", "0", "1923-03-17T09:35:02.305429904Z", "2023-03-17T09:35:02.305429904Z",
					strconv.FormatInt(int64(point.Value), 10),
					field,
					"example",
				}
				logger.Infof("查询指标:%s", field)
				rsp = append(rsp, strings.Join(row, ","))
			}

		}
	}
	if !isOk {
		for _, field := range fields {
			if point, ok := s.Points[field]; ok {
				row := []string{
					"", "", "0", "1923-03-17T09:35:02.305429904Z", "2023-03-17T09:35:02.305429904Z",
					strconv.FormatInt(int64(point.Value), 10),
					field,
					"example",
				}
				isOk = true
				rsp = append(rsp, strings.Join(row, ","))
			}

		}
	}
	if !isOk {
		rsp = []string{}
	}
	c.Writer.Header().Set("Content-Type", "text/csv; charset=utf-8")
	c.String(200, strings.Join(rsp, "\n")+"\n\n")

}
func NewInfluxServer() *httptest.Server {
	onceInfluxDBServer.Do(func() {
		gin.SetMode(gin.DebugMode)
		router := gin.Default()
		mocker := &InfluxdbServerMocker{
			Points: make(map[string]*PointMocker),
		}
		router.POST("/api/v2/write", mocker.WriterInflux)
		router.POST("/api/v2/query", mocker.QueryInflux)
		router.NoRoute(func(c *gin.Context) {
			// 实现内部重定向
			c.String(404, c.FullPath())
		})
		ts = httptest.NewServer(router)
	})
	return ts

}
func newTestDistributedComponents(rdb *miniredis.Miniredis, opts ...DistributeOptions) *DistributedComponents {
	ts = NewInfluxServer()
	config := NewDistributedWorkerConfig(NewRedisConfig(rdb.Addr(), "", "", 0), NewInfluxdbConfig(ts.URL, "xxxx", "test", "distributed"), opts...)
	woker := NewDistributedWorker(config)
	components := NewDistributedComponents(config, woker, woker.rdb)
	return components
}
func TestSerialize(t *testing.T) {
	convey.Convey("test serialize", t, func() {
		body := make(map[string]interface{})
		body["test"] = "test"
		proxy := tegenaria.Proxy{
			ProxyUrl: "http://127.0.0.1",
		}
		spider1 := &tegenaria.TestSpider{
			BaseSpider: tegenaria.NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
		}
		request := tegenaria.NewRequest("http://www.example.com", tegenaria.GET, spider1.Parser, tegenaria.RequestWithMaxRedirects(3), tegenaria.RequestWithRequestBody(body), tegenaria.RequestWithRequestProxy(proxy))
		convey.So(request.AllowRedirects, convey.ShouldBeTrue)
		convey.So(request.MaxRedirects, convey.ShouldAlmostEqual, 3)
		spiderName := spider1.GetName()
		rd, err := newRdbCache(request, "xxxxxxx", spiderName)
		convey.So(err, convey.ShouldBeNil)
		s := newSerialize(rd)
		err = s.dumps()
		convey.So(err, convey.ShouldBeNil)

		s2 := newSerialize(rdbCacheData{})
		err = s2.loads(s.buf.Bytes())
		convey.So(err, convey.ShouldBeNil)
	})

}
func TestDistributedWorker(t *testing.T) {
	convey.Convey("test distribute worker", t, func() {
		mockRedis := miniredis.RunT(t)
		tServer := tegenaria.NewTestServer()
		pServer := tegenaria.NewTestProxyServer()
		// 分布式组件可选参数
		// redis连接池大小
		opts := []DistributeOptions{DistributedWithConnectionsSize(10)}
		// redis超时时间设置
		opts = append(opts, DistributedWithRdbTimeout(5*time.Second))
		// 重试次数
		opts = append(opts, DistributedWithRdbMaxRetry(3))
		opts = append(opts, DistributedWithGetLimitKey(getLimiterDefaultKey))

		components := newTestDistributedComponents(mockRedis, opts...)
		engine := tegenaria.NewTestEngine("distributeWorkerSpider")
		spider1, _ := engine.GetSpiders().GetSpider("distributeWorkerSpider")

		proxy := tegenaria.Proxy{
			ProxyUrl: pServer.URL,
		}
		defer mockRedis.Close()

		body := make(map[string]interface{})
		body["test"] = "test"
		meta := make(map[string]interface{})
		meta["key"] = "value"
		header := make(map[string]string)
		header["Content-Type"] = "application/json"
		requestOptions := []tegenaria.RequestOption{}
		// 添加请求的可选参数
		requestOptions = append(requestOptions, tegenaria.RequestWithRequestHeader(header))
		requestOptions = append(requestOptions, tegenaria.RequestWithRequestBody(body))
		requestOptions = append(requestOptions, tegenaria.RequestWithRequestProxy(proxy))
		requestOptions = append(requestOptions, tegenaria.RequestWithRequestMeta(meta))
		requestOptions = append(requestOptions, tegenaria.RequestWithMaxConnsPerHost(16))
		requestOptions = append(requestOptions, tegenaria.RequestWithMaxRedirects(-1))

		urlReq := fmt.Sprintf("%s/testPOST", tServer.URL)
		request := tegenaria.NewRequest(urlReq, tegenaria.POST, spider1.Parser, requestOptions...)
		ctx := tegenaria.NewContext(request, spider1)
		ctxId := ctx.CtxID
		// 设置请求队列
		queue := components.GetQueue()
		queue.SetCurrentSpider(spider1)
		// 请求入队列
		err := queue.Enqueue(ctx)
		convey.So(err, convey.ShouldBeNil)
		var c interface{}
		// 请求出队列
		c, err = queue.Dequeue()
		convey.So(err, convey.ShouldBeNil)
		newCtx := c.(*tegenaria.Context)
		// 开始处理请求
		downloader := tegenaria.NewDownloader()
		resp, err := downloader.Download(newCtx)
		content, _ := resp.String()
		// 断言处理
		convey.So(err, convey.ShouldBeNil)
		convey.So(content, convey.ShouldContainSubstring, "test")
		convey.So(newCtx.GetCtxID(), convey.ShouldContainSubstring, ctxId)
		convey.So(newCtx.Request.Meta, convey.ShouldContainKey, "key")
		// 校验请求对象参数
		convey.So(newCtx.Request.MaxConnsPerHost, convey.ShouldAlmostEqual, 16)
		convey.So(newCtx.Request.AllowRedirects, convey.ShouldBeFalse)
		convey.So(newCtx.Request.MaxRedirects, convey.ShouldAlmostEqual, 0)
		convey.So(newCtx.Request.Proxy.ProxyUrl, convey.ShouldContainSubstring, pServer.URL)

	})
	convey.Convey("test tegenaria.RequestWithPostForm", t, func() {
		tServer := tegenaria.NewTestServer()
		mockRedis := miniredis.RunT(t)
		defer mockRedis.Close()
		engine := tegenaria.NewTestEngine("RequestWithPostFormDistributeWorkerSpider")
		spider1, _ := engine.GetSpiders().GetSpider("RequestWithPostFormDistributeWorkerSpider")
		components := newTestDistributedComponents(mockRedis)
		components.SetCurrentSpider(spider1)

		urlReq := fmt.Sprintf("%s/testForm", tServer.URL)
		form := url.Values{}
		form.Set("key", "form data")
		header := make(map[string]string)
		header["Content-Type"] = "application/x-www-form-urlencoded"
		request := tegenaria.NewRequest(urlReq, tegenaria.POST, spider1.Parser, tegenaria.RequestWithPostForm(form), tegenaria.RequestWithRequestHeader(header))
		ctx := tegenaria.NewContext(request, spider1)
		queue := components.GetQueue()
		queue.SetCurrentSpider(spider1)
		err := queue.Enqueue(ctx)
		convey.So(err, convey.ShouldBeNil)
		var c interface{}
		c, err = queue.Dequeue()
		convey.So(err, convey.ShouldBeNil)
		newCtx := c.(*tegenaria.Context)
		downloader := tegenaria.NewDownloader()
		resp, err := downloader.Download(newCtx)
		content, _ := resp.String()
		convey.So(err, convey.ShouldBeNil)
		convey.So(content, convey.ShouldContainSubstring, "form data")
	})
}
func TestAddNodeError(t *testing.T) {
	convey.Convey("test add node with error", t, func() {
		mockRedis := miniredis.RunT(t)
		defer mockRedis.Close()
		engine := tegenaria.NewTestEngine("AddNodeErrorSpider")
		spider1, _ := engine.GetSpiders().GetSpider("AddNodeErrorSpider")
		components := newTestDistributedComponents(mockRedis)
		components.SetCurrentSpider(spider1)
		patch := gomonkey.ApplyFunc((*redis.Client).SetEx, func(_ *redis.Client, ctx context.Context, _ string, _ interface{}, _ time.Duration) *redis.StatusCmd {
			s := redis.NewStatusCmd(ctx)
			s.SetErr(errors.New("set ex error"))
			return s
		})
		err := components.worker.AddNode()
		convey.So(err, convey.ShouldBeError, errors.New("set ex error"))
		patch.Reset()

		patch = gomonkey.ApplyFunc((*redis.Client).SAdd, func(_ *redis.Client, ctx context.Context, _ string, _ ...interface{}) *redis.IntCmd {
			s := redis.NewIntCmd(ctx)
			s.SetErr(errors.New("sadd add node error"))
			return s
		})
		err = components.worker.AddNode()
		convey.So(err.Error(), convey.ShouldContainSubstring, "sadd add node error")
		patch.Reset()

		patch = gomonkey.ApplyFunc((*DistributedWorker).addMaster, func(_ *DistributedWorker) error {
			return errors.New("add master error")

		})
		err = components.worker.AddNode()
		convey.So(err.Error(), convey.ShouldContainSubstring, "add master error")
		patch.Reset()

	})

}
func TestDistributedWorkerNodeStatus(t *testing.T) {
	convey.Convey("test distribute worker node status", t, func() {
		// 测试节点状态
		mockRedis := miniredis.RunT(t)
		defer mockRedis.Close()
		engine := tegenaria.NewTestEngine("WorkerNodeStatusSpider")
		spider1, _ := engine.GetSpiders().GetSpider("WorkerNodeStatusSpider")
		nodes := RdbNodes{mockRedis.Addr()}

		time.Sleep(500 * time.Millisecond)
		ts = NewInfluxServer()
		config := NewDistributedWorkerConfig(NewRedisConfig(mockRedis.Addr(), "", "", 0), NewInfluxdbConfig(ts.URL, "xxxx", "test", "distributed"))
		worker := NewWorkerWithRdbCluster(NewWorkerConfigWithRdbCluster(config, nodes))
		components := NewDistributedComponents(config, worker, worker.rdb)
		components.SetCurrentSpider(spider1)
		err := components.worker.AddNode()
		convey.So(err, convey.ShouldBeNil)
		err = components.worker.Heartbeat()
		convey.So(err, convey.ShouldBeNil)

		r, err := components.worker.CheckMasterLive()
		convey.So(err, convey.ShouldBeNil)
		convey.So(r, convey.ShouldBeTrue)
		err = components.worker.StopNode()
		convey.So(err, convey.ShouldBeNil)

		r, err = components.worker.CheckAllNodesStop()
		convey.So(err, convey.ShouldBeNil)
		convey.So(r, convey.ShouldBeTrue)
		r, err = components.worker.CheckMasterLive()
		convey.So(err, convey.ShouldBeNil)
		convey.So(r, convey.ShouldBeFalse)
		ip, err := tegenaria.GetMachineIP()
		convey.So(err, convey.ShouldBeNil)
		member := fmt.Sprintf("%s:%s", ip, components.worker.GetWorkerID())
		key := fmt.Sprintf("%s:%s:%s", "tegenaria:v1:node", spider1.GetName(), member)
		worker.rdb.Del(context.TODO(), key)
		r, err = worker.CheckAllNodesStop()
		convey.So(err, convey.ShouldBeNil)
		convey.So(r, convey.ShouldBeTrue)
		err = worker.close()
		convey.So(err, convey.ShouldBeNil)

	})
}

func TestDistributedBloomFilter(t *testing.T) {
	convey.Convey("test distributed bloom filter", t, func() {
		mockRedis := miniredis.RunT(t)
		engine := tegenaria.NewTestEngine("DistributedBloomFilterSpider")
		spider1, _ := engine.GetSpiders().GetSpider("DistributedBloomFilterSpider")
		components := newTestDistributedComponents(mockRedis)
		defer mockRedis.Close()
		body := make(map[string]interface{})
		body["test"] = "test"
		request1 := tegenaria.NewRequest("http://www.example.com", tegenaria.GET, spider1.Parser, tegenaria.RequestWithRequestBody(body))
		ctx1 := tegenaria.NewContext(request1, spider1)
		dupefilter := components.GetDupefilter()
		dupefilter.SetCurrentSpider(spider1)
		isFilter, err := dupefilter.DoDupeFilter(ctx1)
		convey.So(err, convey.ShouldBeNil)
		convey.So(isFilter, convey.ShouldBeFalse)
		queue := components.GetQueue()
		queue.SetCurrentSpider(spider1)
		err = queue.Enqueue(ctx1)
		convey.So(err, convey.ShouldBeNil)
		request2 := tegenaria.NewRequest("http://www.example.com", tegenaria.GET, spider1.Parser, tegenaria.RequestWithRequestBody(body))
		ctx2 := tegenaria.NewContext(request2, spider1)
		isFilter, err = dupefilter.DoDupeFilter(ctx2)
		convey.So(err, convey.ShouldBeNil)
		convey.So(isFilter, convey.ShouldBeTrue)
		err = queue.Enqueue(ctx2)
		convey.So(err, convey.ShouldBeNil)

		request3 := tegenaria.NewRequest("http://www.example123.com", tegenaria.GET, spider1.Parser, tegenaria.RequestWithRequestBody(body))
		ctx3 := tegenaria.NewContext(request3, spider1)

		isFilter, err = dupefilter.DoDupeFilter(ctx3)
		convey.So(err, convey.ShouldBeNil)
		convey.So(isFilter, convey.ShouldBeFalse)
	})
}
func TestEngineStartWithDistributed(t *testing.T) {
	convey.Convey("engine start with distributed", t, func() {

		mockRedis := miniredis.RunT(t)

		defer mockRedis.Close()
		options := make([]DistributeOptions, 0)
		options = append(options, DistributedWithBloomN(1024))
		options = append(options, DistributedWithBloomP(0.001))
		options = append(options, DistributedWithRdbTimeout(3*time.Second))
		options = append(options, DistributedWithGetLimitKey(getLimiterDefaultKey))
		options = append(options, DistributedWithLimiterRate(16))
		options = append(options, DistributedWithGetBFKey(getBloomFilterKey))
		options = append(options, DistributedWithGetQueueKey(getQueueKey))

		components := newTestDistributedComponents(mockRedis, options...)
		engine := tegenaria.NewTestEngine("testDistributedSpider9", tegenaria.EngineWithComponents(components))
		go func() {
			for range time.Tick(1 * time.Second) {
				mockRedis.FastForward(1 * time.Second)
			}
		}()
		stats := engine.Execute("testDistributedSpider9")
		s := stats.GetAllStats()
		t.Logf("%v", s)
		convey.So(stats.Get(tegenaria.DownloadFailStats), convey.ShouldAlmostEqual, 0)
		convey.So(stats.Get(tegenaria.RequestStats), convey.ShouldAlmostEqual, 1)
		convey.So(stats.Get(tegenaria.ItemsStats), convey.ShouldAlmostEqual, 1)
		convey.So(stats.Get(tegenaria.ErrorStats), convey.ShouldAlmostEqual, 0)
	})

}
func TestEngineStartWithDistributedSlave(t *testing.T) {
	convey.Convey("engine start with distributed", t, func() {

		mockRedis := miniredis.RunT(t)
		defer mockRedis.Close()
		components := newTestDistributedComponents(mockRedis, DistributedWithSlave())
		engine := tegenaria.NewTestEngine("testDistributedSlaveSpider", tegenaria.EngineWithComponents(components))
		patch := gomonkey.ApplyFunc(time.Since, func(_ time.Time) time.Duration {
			return 90 * time.Second
		})
		defer patch.Reset()
		go func() {
			for range time.Tick(1 * time.Second) {
				mockRedis.FastForward(1 * time.Second)
			}
		}()
		convey.So(func() { engine.Execute("testDistributedSlaveSpider") }, convey.ShouldPanic)
	})

}
