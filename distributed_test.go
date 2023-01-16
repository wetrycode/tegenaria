package tegenaria

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
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
		request := NewRequest("http://www.example.com", GET, spider1.Parser, RequestWithRequestBody(body), RequestWithRequestProxy(proxy))
		GetAllParserMethod(spider1)
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
		spiders[spider1.GetName()] = spider1
		worker := NewDistributedWorker(mockRedis.Addr(), config)
		worker.SetSpiders(&Spiders{
			SpidersModules: spiders,
		})
		defer mockRedis.Close()

		body := make(map[string]interface{})
		body["test"] = "test"
		request := NewRequest("http://www.example.com", GET, spider1.Parser, RequestWithRequestBody(body))
		ctx := NewContext(request, spider1)
		ctxId := ctx.CtxId
		err = worker.enqueue(ctx)
		convey.So(err, convey.ShouldBeNil)
		var c interface{}
		c, err = worker.dequeue()
		convey.So(err, convey.ShouldBeNil)
		newCtx := c.(*Context)
		convey.So(newCtx.CtxId, convey.ShouldContainSubstring, ctxId)
	})
}
func TestDistributedWorkerNodeStatus(t *testing.T) {
	convey.Convey("test distribute worker node status", t, func() {
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
		err = worker.AddNode()
		convey.So(err, convey.ShouldBeNil)
		err = worker.Heartbeat()
		convey.So(err, convey.ShouldBeNil)
		err=worker.StopNode()
		convey.So(err, convey.ShouldBeNil)

		// time.Sleep(1 * time.Second)
		// mockRedis.FastForward(2 * time.Second)
		r, err := worker.CheckAllNodesStop()
		convey.So(err, convey.ShouldBeNil)
		convey.So(r, convey.ShouldBeTrue)
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
