package tegenaria

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestSerialize(t *testing.T) {

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
	if err != nil {
		t.Errorf("request serialize fail %s", err.Error())
	}
	s := newSerialize(rd)
	err = s.dumps()
	if err != nil {
		t.Errorf("request serialize fail %s", err.Error())
	}
	s2 := newSerialize(rdbCacheData{})
	err = s2.loads(s.buf.Bytes())
	if err != nil {
		t.Errorf("request deserialize fail %s", err.Error())
	}
	t.Logf("url:%s,parser:%v", s2.val["url"], s2.val["parser"])

}
func TestDistributedWorker(t *testing.T) {
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
	// proxy := Proxy{
	// 	ProxyUrl: "http://127.0.0.1",
	// }
	request := NewRequest("http://www.example.com", GET, spider1.Parser, RequestWithRequestBody(body))
	ctx := NewContext(request, spider1)
	err = worker.enqueue(ctx)
	if err != nil {
		t.Errorf("context push into redis cache error %s", err.Error())
	}
	var c interface{}
	c, err = worker.dequeue()
	if err != nil {
		t.Errorf("context pop from redis cache error %s", err.Error())
	}
	newCtx := c.(*Context)
	t.Logf("get the context with id %s", newCtx.CtxId)

}

func TestDistributedBloomFilter(t *testing.T) {
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
	isFilter, err := worker.DoDupeFilter(ctx1.Request)
	if err != nil {
		t.Errorf("dupfilter error %s", err.Error())
	}
	if isFilter {
		t.Errorf("except filter result is false,but get true")
	}
	err = worker.enqueue(ctx1)
	if err != nil {
		t.Errorf("context push into redis cache error %s", err.Error())

	}

	request2 := NewRequest("http://www.example.com", GET, spider1.Parser, RequestWithRequestBody(body))
	ctx2 := NewContext(request2, spider1)
	isFilter, err = worker.DoDupeFilter(ctx2.Request)
	if err != nil {
		t.Errorf("dupfilter error %s", err.Error())
	}
	if !isFilter {
		t.Errorf("except filter result is true,but get false")
	}
	worker.enqueue(ctx2)

	request3 := NewRequest("http://www.example123.com", GET, spider1.Parser, RequestWithRequestBody(body))
	isFilter, err = worker.DoDupeFilter(request3)
	if err != nil {
		t.Errorf("dupfilter error %s", err.Error())
	}
	if isFilter {
		t.Errorf("except filter result is false,but get true")
	}

}
