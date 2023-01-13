package tegenaria

import (
	"bytes"
	goContext "context"
	"encoding/gob"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/spaolacci/murmur3"
)

type GetRDBKey func(params ...interface{}) string

// DistributedWorker 分布式组件，包含两个组件:
// request请求缓存队列，由各个节点上的引擎读队列消费，
// redis 队列缓存的是经过gob序列化之后的二进制数据
// 布隆过滤器主要是用于去重
// 该组件同事实现了RFPDupeFilterInterface 和CacheInterface
type DistributedWorker struct {
	// rdb redis客户端支持redis单机实例和redis cluster集群模式
	rdb redis.Cmdable
	// getQueueKey 生成队列key的函数，允许用户自定义
	getQueueKey GetRDBKey
	// getBloomFilterKey 布隆过滤器对应的生成key的函数，允许用户自定义
	getBloomFilterKey GetRDBKey
	// spiders 所有的SpiderInterface实例
	spiders *Spiders
	// dupeFilter 去重组件
	dupeFilter *RFPDupeFilter
	// bloomP 布隆过滤器的容错率
	bloomP float64
	// bloomK hash函数个数
	bloomK uint
	// bloomN 数据规模，比如1024 * 1024
	bloomN uint
	// bloomM bitset 大小
	bloomM uint
	// 并发控制器
	limiter LimitInterface
}

// serialize 序列化组件
type serialize struct {
	buf bytes.Buffer
	val rdbCacheData
}

// RdbNodes redis cluster 节点地址
type RdbNodes []string

// DistributedWorkerConfig 分布式组件的配置参数
type DistributedWorkerConfig struct {
	// RedisAddr redis 地址
	RedisAddr string
	// RedisPasswd redis 密码
	RedisPasswd string
	// RedisUsername redis 用户名
	RedisUsername string
	// RedisDB redis 数据库索引 index
	RedisDB uint32
	// RdbConnectionsSize 连接池大小
	RdbConnectionsSize uint64
	// RdbTimeout redis 超时时间
	RdbTimeout time.Duration
	// RdbMaxRetry redis操作失败后的重试次数
	RdbMaxRetry int
	// BloomP 布隆过滤器的容错率
	BloomP float64
	// BloomN 数据规模，比如1024 * 1024
	BloomN uint
	// 并发量
	CurrentRequestNum int
	// GetqueueKey 生成队列key的函数，允许用户自定义
	GetqueueKey GetRDBKey
	// GetBFKey 布隆过滤器对应的生成key的函数，允许用户自定义
	GetBFKey GetRDBKey
	getLimitKey GetRDBKey

}

// rdbCacheData request 队列缓存的数据结构
type rdbCacheData map[string]interface{}

// WorkerConfigWithRdbCluster redis cluser 模式下的分布式组件配置参数
type WorkerConfigWithRdbCluster struct {
	*DistributedWorkerConfig
	RdbNodes
}

// NewDistributedWorker 构建redis单机模式下的分布式工作组件
func NewDistributedWorker(config *DistributedWorkerConfig) *DistributedWorker {
	// 获取最优bit 数组的大小
	m := OptimalNumOfBits(int64(config.BloomN), config.BloomP)
	// 获取最优的hash函数个数
	k := OptimalNumOfHashFunctions(int64(config.BloomN), m)
	rdb:=NewRdbClient(config)
	return &DistributedWorker{
		rdb:               rdb,
		getQueueKey:       config.GetqueueKey,
		getBloomFilterKey: config.GetBFKey,
		dupeFilter:        NewRFPDupeFilter(config.BloomP, uint(k)),
		bloomP:            config.BloomP,
		bloomK:            uint(k),
		bloomN:            config.BloomN,
		bloomM:            uint(m),
		limiter: NewLeakyBucketLimiterWithRdb(config.CurrentRequestNum, rdb, config.getLimitKey()),
	}
}

func getLimiterDefaultKey(params ...interface{}) string {
	return "tegenaria:v1:bf"
}
// getBloomFilterDefaultKey 自定义的布隆过滤器key生成函数
func getBloomFilterDefaultKey(params ...interface{}) string {
	return "tegenaria:v1:bf"
}

// getQueueDefaultKey 自定义的request缓存队列key生成函数
func getQueueDefaultKey(params ...interface{}) string {
	return "tegenaria:v1:request"

}

// NewWorkerWithRdbCluster redis cluster模式下的分布式工作组件
func NewWorkerWithRdbCluster(config *WorkerConfigWithRdbCluster) *DistributedWorker {
	w := NewDistributedWorker(config.DistributedWorkerConfig)
	// 替换为redis cluster客户端
	w.rdb = NewRdbClusterCLient(config)
	return w
}
func (w *DistributedWorker)GetLimter() LimitInterface{
	return w.limiter
}
// newRdbCache 构建待缓存的数据
func newRdbCache(request *Request, ctxId string, spiderName string) (rdbCacheData, error) {
	r, err := request.ToMap()
	if err != nil {
		return nil, err
	}
	r["ctxId"] = ctxId
	r["parser"] = GetFunctionName(request.Parser)
	if request.Proxy != nil {
		r["proxyUrl"] = request.Proxy.ProxyUrl

	}
	r["spiderName"] = spiderName
	return r, nil
}

// loads 从缓存队列中加载请求并反序列化
func (s *serialize) loads(request []byte) error {
	decoder := gob.NewDecoder(bytes.NewReader(request))
	return decoder.Decode(&s.val)

}

// dumps 序列化操作
func (s *serialize) dumps() error {
	enc := gob.NewEncoder(&s.buf)
	return enc.Encode(s.val)
}

// doSerialize 对request 进行序列化操作方便缓存
// 返回的是二进制数组
func doSerialize(ctx *Context) ([]byte, error) {
	// 先构建需要缓存的对象
	data, err := newRdbCache(ctx.Request, ctx.CtxId, ctx.Spider.GetName())
	if err != nil {
		return nil, err
	}
	s := &serialize{
		buf: bytes.Buffer{},
		val: data,
	}
	err = s.dumps()
	if err != nil {
		return nil, err
	}
	return s.buf.Bytes(), nil
}

// unserialize 对从rdb中读取到的二进制数据进行反序列化
// 返回一个rdbCacheData对象
func unserialize(data []byte) (rdbCacheData, error) {
	s := &serialize{
		buf: bytes.Buffer{},
		val: make(rdbCacheData),
	}
	err := s.loads(data)
	if err != nil {
		return nil, err
	}
	return s.val, nil
}
func newSerialize(r rdbCacheData) *serialize {
	return &serialize{
		val: r,
	}
}
func (w *DistributedWorker) SetSpiders(spiders *Spiders) {
	w.spiders = spiders
}

// enqueue request对象缓存入rdb队列
// 先将request 对象进行序列化再push入指定的队列
func (w *DistributedWorker) enqueue(ctx *Context) error {
	defer ctx.Close()
	// It will wait to put request until queue is not full
	if ctx == nil || ctx.Request == nil {
		return nil
	}
	key := w.getQueueKey()
	bytes, err := doSerialize(ctx)
	if err != nil {
		return err
	}
	w.rdb.LPush(goContext.TODO(), key, bytes)
	return nil

}

// dequeue 从缓存队列中读取二进制对象并序列化为rdbCacheData
// 随后构建context
func (w *DistributedWorker) dequeue() (interface{}, error) {
	data, err := w.rdb.RPop(goContext.TODO(), w.getQueueKey()).Bytes()
	if err != nil {
		return nil, err
	}
	req, err := unserialize(data)
	if err != nil {
		return nil, err
	}
	spider := w.spiders.SpidersModules[req["spiderName"].(string)]

	opts := []RequestOption{}
	opts = append(opts, RequestWithParser(GetParserByName(spider, req["parser"].(string))))
	opts = append(opts, RequestWithRequestProxy(Proxy{ProxyUrl: req["proxyUrl"].(string)}))
	request := RequestFromMap(req, opts...)
	return NewContext(request, spider, WithContextId(req["ctxId"].(string))), nil

}

// getSize 获取队列大小
func (w *DistributedWorker) getSize() int64 {
	length, _ := w.rdb.LLen(goContext.TODO(), w.getQueueKey()).Uint64()
	return int64(length)
}

// Fingerprint 生成request 对象的指纹
func (w *DistributedWorker) Fingerprint(request *Request) ([]byte, error) {
	fp, err := w.dupeFilter.Fingerprint(request)
	if err != nil {
		return nil, err
	}
	return fp, nil
}

// baseHashes 生成hash值
func baseHashes(data []byte) [4]uint64 {
	a1 := []byte{1} // to grab another bit of data
	hasher := murmur3.New128()
	hasher.Write(data) // #nosec
	v1, v2 := hasher.Sum128()
	hasher.Write(a1) // #nosec
	v3, v4 := hasher.Sum128()
	return [4]uint64{
		v1, v2, v3, v4,
	}
}

// location 获取第i个的hash值
func location(h [4]uint64, i uint) uint64 {
	ii := uint64(i)
	return h[ii%2] + ii*h[2+(((ii+(ii%2))%4)/2)]
}

// getOffset 计算偏移量
func (w *DistributedWorker) getOffset(hash [4]uint64, index uint) uint {
	return uint(location(hash, index) % uint64(w.bloomM))
}

// DoDupeFilter request去重处理,如果指纹已经存在则返回True,否则为False
// 指纹不存在的情况下会将指纹添加到缓存
func (w *DistributedWorker) DoDupeFilter(request *Request) (bool, error) {
	fp, err := w.Fingerprint(request)
	if err != nil {
		return false, err
	}
	return w.TestOrAdd(fp)
}

// TestOrAdd 如果指纹已经存在则返回True,否则为False
// 指纹不存在的情况下会将指纹添加到缓存
func (w *DistributedWorker) TestOrAdd(fingerprint []byte) (bool, error) {
	isExists, err := w.isExists(fingerprint)
	if err != nil {
		return false, err
	}
	if isExists {
		return true, nil
	}
	err = w.Add(fingerprint)
	return false, err
}

// Add 添加指纹到布隆过滤器
func (w *DistributedWorker) Add(fingerprint []byte) error {
	h := baseHashes(fingerprint)
	pipe := w.rdb.Pipeline()
	for i := uint(0); i < w.bloomK; i++ {
		value := w.getOffset(h, i)
		pipe.SetBit(goContext.TODO(), w.getBloomFilterKey(), int64(value), 1)
	}
	_, err := pipe.Exec(goContext.TODO())
	if err != nil {
		return err
	}
	return nil
}

// isExists 判断指纹是否存在
func (w *DistributedWorker) isExists(fingerprint []byte) (bool, error) {
	h := baseHashes(fingerprint)
	pipe := w.rdb.Pipeline()
	result := []*redis.IntCmd{}
	for i := uint(0); i < w.bloomK; i++ {
		value := w.getOffset(h, i)
		result = append(result, pipe.GetBit(goContext.TODO(), w.getBloomFilterKey(), int64(value)))
	}
	_, err := pipe.Exec(goContext.TODO())
	if err != nil {
		return false, err
	}
	for _, val := range result {
		r, err := val.Result()
		if err != nil {
			return false, err
		}
		if r == 0 {
			return false, nil
		}
	}
	return true, nil
}
