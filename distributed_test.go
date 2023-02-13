package tegenaria

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/smartystreets/goconvey/convey"
)

func TestSerialize(t *testing.T) {
	convey.Convey("test serialize", t, func() {
		body := make(map[string]interface{})
		body["test"] = "test"
		proxy := Proxy{
			ProxyUrl: "http://127.0.0.1",
		}
		spider1 := &TestSpider{
			NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
		}
		request := NewRequest("http://www.example.com", GET, spider1.Parser, RequestWithMaxRedirects(3), RequestWithRequestBody(body), RequestWithRequestProxy(proxy))
		convey.So(request.AllowRedirects, convey.ShouldBeTrue)
		convey.So(request.MaxRedirects, convey.ShouldAlmostEqual, 3)
		// GetAllParserMethod(spider1)
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
		mockRedis, err := miniredis.Run()
		pServer := newTestProxyServer()
		tServer := newTestServer()
		if err != nil {
			panic(err)
		}
		opts := []DistributeOptions{DistributedWithConnectionsSize(10)}
		opts = append(opts, DistributedWithRdbTimeout(5*time.Second))
		opts = append(opts, DistributedWithRdbMaxRetry(3))
		opts = append(opts, DistributedWithGetLimitKey(getLimiterDefaultKey))
		opts = append(opts, DistributedWithGetBFKey(getBloomFilterDefaultKey))
		opts = append(opts, DistributedWithGetqueueKey(getQueueDefaultKey))
		opts = append(opts, DistributedWithBloomN(1024*1024))
		opts = append(opts, DistributedWithBloomP(0.001))
		config := NewDistributedWorkerConfig("", "", 0, opts...)
		spiders := map[string]SpiderInterface{}
		spider1 := &TestSpider{
			NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
		}
		proxy := Proxy{
			ProxyUrl: pServer.URL,
		}
		spiders[spider1.GetName()] = spider1
		worker := NewDistributedWorker(mockRedis.Addr(), config)
		worker.SetSpiders(&Spiders{
			SpidersModules: spiders,
		})
		defer mockRedis.Close()

		body := make(map[string]interface{})
		body["test"] = "test"
		meta := make(map[string]interface{})
		meta["key"] = "value"
		requestOptions := []RequestOption{}
		requestOptions = append(requestOptions, RequestWithRequestBody(body))
		requestOptions = append(requestOptions, RequestWithRequestProxy(proxy))
		requestOptions = append(requestOptions, RequestWithRequestMeta(meta))
		requestOptions = append(requestOptions, RequestWithMaxConnsPerHost(16))
		requestOptions = append(requestOptions, RequestWithMaxRedirects(-1))
		urlReq := fmt.Sprintf("%s/testPOST", tServer.URL)
		request := NewRequest(urlReq, POST, spider1.Parser, requestOptions...)
		ctx := NewContext(request, spider1)
		ctxId := ctx.CtxID
		err = worker.enqueue(ctx)
		convey.So(err, convey.ShouldBeNil)
		var c interface{}
		c, err = worker.dequeue()
		convey.So(err, convey.ShouldBeNil)
		newCtx := c.(*Context)
		downloader := NewDownloader()
		resp, err := downloader.Download(newCtx)
		content, _ := resp.String()
		convey.So(err, convey.ShouldBeNil)
		convey.So(content, convey.ShouldContainSubstring, "test")
		convey.So(newCtx.GetCtxID(), convey.ShouldContainSubstring, ctxId)
		convey.So(newCtx.Request.Meta, convey.ShouldContainKey, "key")
		convey.So(newCtx.Request.MaxConnsPerHost, convey.ShouldAlmostEqual, 16)
		convey.So(newCtx.Request.AllowRedirects, convey.ShouldBeFalse)
		convey.So(newCtx.Request.MaxRedirects, convey.ShouldAlmostEqual, 3)
		convey.So(newCtx.Request.Proxy.ProxyUrl, convey.ShouldContainSubstring, pServer.URL)

	})
	convey.Convey("test RequestWithPostForm", t, func() {
		tServer := newTestServer()
		mockRedis := miniredis.RunT(t)
		defer mockRedis.Close()
		config := NewDistributedWorkerConfig("", "", 0)
		spiders := map[string]SpiderInterface{}
		spider1 := &TestSpider{
			NewBaseSpider("testspider2", []string{"https://www.baidu.com"}),
		}
		worker := NewDistributedWorker(mockRedis.Addr(), config)
		worker.SetSpiders(&Spiders{
			SpidersModules: spiders,
		})
		urlReq := fmt.Sprintf("%s/testForm", tServer.URL)
		form := url.Values{}
		form.Set("key", "form data")
		header := make(map[string]string)
		header["Content-Type"] = "application/x-www-form-urlencoded"
		request := NewRequest(urlReq, POST, spider1.Parser, RequestWithPostForm(form), RequestWithRequestHeader(header))
		ctx := NewContext(request, spider1)
		err := worker.enqueue(ctx)
		convey.So(err, convey.ShouldBeNil)
		var c interface{}
		c, err = worker.dequeue()
		convey.So(err, convey.ShouldBeNil)
		newCtx := c.(*Context)
		downloader := NewDownloader()
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
		config := NewDistributedWorkerConfig("", "", 0)
		spider1 := &TestSpider{
			NewBaseSpider("testAddNoeErrorSpider", []string{"https://www.baidu.com"}),
		}
		spiders := map[string]SpiderInterface{}
		spiders[spider1.GetName()] = spider1
		worker := NewDistributedWorker(mockRedis.Addr(), config)
		worker.SetSpiders(&Spiders{
			SpidersModules: spiders,
		})
		patch := gomonkey.ApplyFunc((*redis.Client).SetEX, func(_ *redis.Client, ctx context.Context, _ string, _ interface{}, _ time.Duration) *redis.StatusCmd {
			s := redis.NewStatusCmd(ctx)
			s.SetErr(errors.New("set ex error"))
			return s
		})
		err := worker.AddNode()
		convey.So(err, convey.ShouldBeError, errors.New("set ex error"))
		patch.Reset()

		patch = gomonkey.ApplyFunc((*redis.Client).SAdd, func(_ *redis.Client, ctx context.Context, _ string, _ ...interface{}) *redis.IntCmd {
			s := redis.NewIntCmd(ctx)
			s.SetErr(errors.New("sadd add node error"))
			return s
		})
		err = worker.AddNode()
		convey.So(err.Error(), convey.ShouldContainSubstring, "sadd add node error")
		patch.Reset()

		patch = gomonkey.ApplyFunc((*DistributedWorker).addMaster, func(_ *DistributedWorker) error {
			return errors.New("add master error")

		})
		err = worker.AddNode()
		convey.So(err.Error(), convey.ShouldContainSubstring, "add master error")
		patch.Reset()

	})

}
func TestDistributedWorkerNodeStatus(t *testing.T) {
	convey.Convey("test distribute worker node status", t, func() {
		mockRedis := miniredis.RunT(t)
		defer mockRedis.Close()
		config := NewDistributedWorkerConfig("", "", 0)
		spider1 := &TestSpider{
			NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
		}
		spiders := map[string]SpiderInterface{}
		spiders[spider1.GetName()] = spider1
		// worker := NewDistributedWorker(mockRedis.Addr(), config)
		nodes := RdbNodes{mockRedis.Addr()}
		time.Sleep(500 * time.Millisecond)
		worker := NewWorkerWithRdbCluster(NewWorkerConfigWithRdbCluster(config, nodes))
		worker.SetSpiders(&Spiders{
			SpidersModules: spiders,
		})
		err := worker.AddNode()
		convey.So(err, convey.ShouldBeNil)
		err = worker.Heartbeat()
		convey.So(err, convey.ShouldBeNil)

		r, err := worker.CheckMasterLive()
		convey.So(err, convey.ShouldBeNil)
		convey.So(r, convey.ShouldBeTrue)
		err = worker.StopNode()
		convey.So(err, convey.ShouldBeNil)

		r, err = worker.CheckAllNodesStop()
		convey.So(err, convey.ShouldBeNil)
		convey.So(r, convey.ShouldBeTrue)
		r, err = worker.CheckMasterLive()
		convey.So(err, convey.ShouldBeNil)
		convey.So(r, convey.ShouldBeFalse)
		ip, err := GetMachineIp()
		convey.So(err, convey.ShouldBeNil)
		member := fmt.Sprintf("%s:%s", ip, worker.nodeID)
		key := fmt.Sprintf("%s:%s:%s", worker.nodePrefix, worker.currentSpider, member)
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
		mockRedis, err := miniredis.Run()
		if err != nil {
			panic(err)
		}
		config := NewDistributedWorkerConfig("", "", 0)
		spider1 := &TestSpider{
			NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
		}
		spiders := map[string]SpiderInterface{}
		spiders[spider1.GetName()] = spider1
		worker := NewDistributedWorker(mockRedis.Addr(), config)
		worker.SetSpiders(&Spiders{
			SpidersModules: spiders,
		})
		defer mockRedis.Close()
		body := make(map[string]interface{})
		body["test"] = "test"
		request1 := NewRequest("http://www.example.com", GET, spider1.Parser, RequestWithRequestBody(body))
		ctx1 := NewContext(request1, spider1)
		isFilter, err := worker.DoDupeFilter(ctx1)
		convey.So(err, convey.ShouldBeNil)
		convey.So(isFilter, convey.ShouldBeFalse)
		err = worker.enqueue(ctx1)
		convey.So(err, convey.ShouldBeNil)
		request2 := NewRequest("http://www.example.com", GET, spider1.Parser, RequestWithRequestBody(body))
		ctx2 := NewContext(request2, spider1)
		isFilter, err = worker.DoDupeFilter(ctx2)
		convey.So(err, convey.ShouldBeNil)
		convey.So(isFilter, convey.ShouldBeTrue)
		worker.enqueue(ctx2)
		request3 := NewRequest("http://www.example123.com", GET, spider1.Parser, RequestWithRequestBody(body))
		ctx3 := NewContext(request3, spider1)

		isFilter, err = worker.DoDupeFilter(ctx3)
		convey.So(err, convey.ShouldBeNil)
		convey.So(isFilter, convey.ShouldBeFalse)
	})
}
