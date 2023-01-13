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
func TestDistributedWorker(t *testing.T){
	mockRedis, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	worker:=NewDistributedWorker(&DistributedWorkerConfig{
		RedisAddr: mockRedis.Addr(),
		RedisPasswd: "",
		RedisUsername: "",
		RedisDB: 0,
		RdbConnectionsSize: 15,
		RdbTimeout: 5 * time.Second,
		RdbMaxRetry: 3,
		getLimitKey: getLimiterDefaultKey,
		GetqueueKey: getQueueDefaultKey,
		GetBFKey: getBloomFilterDefaultKey,
	})
	defer mockRedis.Close()
	spider1 := &TestSpider{
		NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
	}
	spiders:=map[string]SpiderInterface{}
	spiders[spider1.GetName()] = spider1
	worker.SetSpiders(&Spiders{
		SpidersModules: spiders,
	})
	body := make(map[string]interface{})
	body["test"] = "test"
	proxy := Proxy{
		ProxyUrl: "http://127.0.0.1",
	}
	request := NewRequest("http://www.example.com", GET, spider1.Parser, RequestWithRequestBody(body), RequestWithRequestProxy(proxy))
	ctx:=NewContext(request, spider1)
	err=worker.enqueue(ctx)
	if err!=nil{
		t.Errorf("context push into redis cache error %s", err.Error())
	}
	var c interface{}
	c, err=worker.dequeue()
	if err!=nil{
		t.Errorf("context pop from redis cache error %s", err.Error())
	}
	newCtx := c.(*Context)
	t.Logf("get the context with id %s", newCtx.CtxId)

}

func TestDistributedBloomFilter(t *testing.T){
	mockRedis, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	worker:=NewDistributedWorker(&DistributedWorkerConfig{
		RedisAddr: mockRedis.Addr(),
		RedisPasswd: "",
		RedisUsername: "",
		RedisDB: 0,
		RdbConnectionsSize: 15,
		RdbTimeout: 5 * time.Second,
		RdbMaxRetry: 3,
		GetqueueKey: getQueueDefaultKey,
		GetBFKey: getBloomFilterDefaultKey,
		BloomP: 0.001,
		BloomN: 1024 * 1024 * 4,
		getLimitKey: getLimiterDefaultKey,


	})
	defer mockRedis.Close()


	spider1 := &TestSpider{
		NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
	}
	spiders:=map[string]SpiderInterface{}
	spiders[spider1.GetName()] = spider1
	worker.SetSpiders(&Spiders{
		SpidersModules: spiders,
	})
	body := make(map[string]interface{})
	body["test"] = "test"
	request1 := NewRequest("http://www.example.com", GET, spider1.Parser, RequestWithRequestBody(body))
	ctx1:=NewContext(request1, spider1)
	isFilter, err:=worker.DoDupeFilter(ctx1.Request)
	if err!=nil{
		t.Errorf("dupfilter error %s", err.Error())
	}
	if isFilter{
		t.Errorf("except filter result is false,but get true")
	}
	err=worker.enqueue(ctx1)
	if err!=nil{
		t.Errorf("context push into redis cache error %s", err.Error())

	}

	request2 := NewRequest("http://www.example.com", GET, spider1.Parser, RequestWithRequestBody(body))
	ctx2:=NewContext(request2, spider1)
	isFilter, err=worker.DoDupeFilter(ctx2.Request)
	if err!=nil{
		t.Errorf("dupfilter error %s", err.Error())
	}
	if !isFilter{
		t.Errorf("except filter result is true,but get false")
	}
	worker.enqueue(ctx2)

	request3 := NewRequest("http://www.example123.com", GET, spider1.Parser, RequestWithRequestBody(body))
	isFilter, err=worker.DoDupeFilter(request3)
	if err!=nil{
		t.Errorf("dupfilter error %s", err.Error())
	}
	if isFilter{
		t.Errorf("except filter result is false,but get true")
	}

}

