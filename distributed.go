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

package tegenaria

import (
	"bytes"
	goContext "context"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/spaolacci/murmur3"
)

// GetRDBKey 获取缓存rdb key和ttl
type GetRDBKey func(params ...interface{}) (string, time.Duration)

// DistributedWorkerInterface 分布式组件接口
type DistributedWorkerInterface interface {
	CacheInterface
	RFPDupeFilterInterface
	// AddNode 新增一个节点
	AddNode() error
	// DelNode 删除当前的节点
	DelNode() error
	// StopNode 停止当前的节点
	StopNode() error
	// Heartbeat 心跳
	Heartbeat() error
	// CheckAllNodesStop 检查所有的节点是否都已经停止
	CheckAllNodesStop() (bool, error)
	// GetLimter 获取限速器
	GetLimter() LimitInterface
	// CheckMasterLive 检测主节点是否还在线
	CheckMasterLive() (bool, error)
	// SetMaster 是否将当前的节点设置为主节点
	SetMaster(flag bool)
	// SetSpiders 设置已经注册的所有的spider
	SetSpiders(spiders *Spiders)
}

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
	// limiter 并发控制器
	limiter LimitInterface
	// nodeId 节点id
	nodeId string
	// nodesSetPrefix 节点池的key前缀
	// 默认为"tegenaria:v1:nodes"
	nodesSetPrefix string
	// masterNodesKey master节点池的key前缀
	// 默认值为"tegenaria:v1:master"
	masterNodesKey string
	// nodePrefix 节点前缀
	nodePrefix string
	// currentSpider 当前的spider
	currentSpider string
	// isMaster 是否是master节点
	isMaster bool
}

// serialize 序列化组件
type serialize struct {
	buf bytes.Buffer
	val rdbCacheData
}

// RdbNodes redis cluster 节点地址
type RdbNodes []string
type DistributeOptions func(w *DistributedWorkerConfig)

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
	LimiterRate int
	// GetqueueKey 生成队列key的函数，允许用户自定义
	GetqueueKey GetRDBKey
	// GetBFKey 布隆过滤器对应的生成key的函数，允许用户自定义
	GetBFKey GetRDBKey
	// getLimitKey 获取限速器对应的redis key
	getLimitKey GetRDBKey
}

// rdbCacheData request 队列缓存的数据结构
type rdbCacheData map[string]interface{}

// WorkerConfigWithRdbCluster redis cluser 模式下的分布式组件配置参数
type WorkerConfigWithRdbCluster struct {
	*DistributedWorkerConfig
	RdbNodes
}

// NewDistributedWorkerConfig 新建分布式组件的配置
func NewDistributedWorkerConfig(username string, passwd string, db uint32, opts ...DistributeOptions) *DistributedWorkerConfig {
	config := &DistributedWorkerConfig{
		RedisUsername:      username,
		RedisPasswd:        passwd,
		RedisDB:            db,
		RdbConnectionsSize: 32,
		RdbTimeout:         10 * time.Second,
		RdbMaxRetry:        3,
		BloomP:             0.001,
		BloomN:             1024 * 1024,
		LimiterRate:        32,
		GetqueueKey:        getQueueDefaultKey,
		GetBFKey:           getBloomFilterDefaultKey,
		getLimitKey:        getLimiterDefaultKey,
	}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// NewWorkerConfigWithRdbCluster redis cluster模式的分布式组件
func NewWorkerConfigWithRdbCluster(config *DistributedWorkerConfig, nodes RdbNodes) *WorkerConfigWithRdbCluster {
	newConfig := &WorkerConfigWithRdbCluster{
		DistributedWorkerConfig: config,
		RdbNodes:                nodes,
	}
	return newConfig
}

// NewDistributedWorker 构建redis单机模式下的分布式工作组件
func NewDistributedWorker(addr string, config *DistributedWorkerConfig) *DistributedWorker {
	// 获取最优bit 数组的大小
	m := OptimalNumOfBits(int64(config.BloomN), config.BloomP)
	// 获取最优的hash函数个数
	k := OptimalNumOfHashFunctions(int64(config.BloomN), m)
	config.RedisAddr = addr
	rdb := NewRdbClient(config)

	d := &DistributedWorker{
		rdb:               rdb,
		getQueueKey:       config.GetqueueKey,
		getBloomFilterKey: config.GetBFKey,
		dupeFilter:        NewRFPDupeFilter(config.BloomP, uint(k)),
		bloomP:            config.BloomP,
		bloomK:            uint(k),
		bloomN:            config.BloomN,
		bloomM:            uint(m),
		nodeId:            GetUUID(),
		nodesSetPrefix:    "tegenaria:v1:nodes",
		nodePrefix:        "tegenaria:v1:node",
		masterNodesKey:    "tegenaria:v1:master",
		isMaster:          true,
		limiter:           NewLeakyBucketLimiterWithRdb(config.LimiterRate, rdb, config.getLimitKey),
	}
	return d
}

// setCurrentSpider 设置当前的spider
func (w *DistributedWorker) setCurrentSpider(spider string) {
	w.currentSpider = spider
	funcs := []GetRDBKey{}
	funcs = append(funcs, func(params ...interface{}) (string, time.Duration) {
		return w.queueKey()
	})
	funcs = append(funcs, func(params ...interface{}) (string, time.Duration) {
		return w.bfKey()
	})
	for _, f := range funcs {
		key, ttl := f()
		if ttl > 0 {
			w.rdb.Expire(goContext.TODO(), key, ttl)
		}

	}
}

// NewWorkerWithRdbCluster redis cluster模式下的分布式工作组件
func NewWorkerWithRdbCluster(config *WorkerConfigWithRdbCluster) *DistributedWorker {
	w := NewDistributedWorker("", config.DistributedWorkerConfig)
	// 替换为redis cluster客户端
	w.rdb = NewRdbClusterCLient(config)
	return w
}

// getLimiterDefaultKey 限速器默认的key
func getLimiterDefaultKey(params ...interface{}) (string, time.Duration) {
	return "tegenaria:v1:limiter", 0 * time.Second
}

// getBloomFilterDefaultKey 自定义的布隆过滤器key生成函数
func getBloomFilterDefaultKey(params ...interface{}) (string, time.Duration) {
	return "tegenaria:v1:bf", 0 * time.Second
}

// getQueueDefaultKey 自定义的request缓存队列key生成函数
func getQueueDefaultKey(params ...interface{}) (string, time.Duration) {
	return "tegenaria:v1:request", 0 * time.Second

}

// GetLimter 获取限速器
func (w *DistributedWorker) GetLimter() LimitInterface {
	return w.limiter
}

// SetMaster 设置当前的节点是否为master
func (w *DistributedWorker) SetMaster(flag bool) {
	w.isMaster = flag
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

// newSerialize 获取序列化组件
func newSerialize(r rdbCacheData) *serialize {
	return &serialize{
		val: r,
	}
}

// SetSpiders 设置所有的已注册的spider
func (w *DistributedWorker) SetSpiders(spiders *Spiders) {
	w.spiders = spiders
}

// enqueue request对象缓存入rdb队列
// 先将request 对象进行序列化再push入指定的队列
func (w *DistributedWorker) enqueue(ctx *Context) error {
	// It will wait to put request until queue is not full
	if ctx == nil || ctx.Request == nil {
		return nil
	}
	key, _ := w.queueKey()

	bytes, err := doSerialize(ctx)
	if err != nil {
		return err
	}
	_, err = w.rdb.LPush(goContext.TODO(), key, bytes).Uint64()

	return err

}

// dequeue 从缓存队列中读取二进制对象并序列化为rdbCacheData
// 随后构建context
func (w *DistributedWorker) dequeue() (interface{}, error) {
	key, _ := w.queueKey()
	data, err := w.rdb.RPop(goContext.TODO(), key).Bytes()
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
	if val, ok := req["proxyUrl"]; ok {
		opts = append(opts, RequestWithRequestProxy(Proxy{ProxyUrl: val.(string)}))
	}
	if val, ok := req["body"]; ok && val != nil {
		decodeBytes, _ := base64.StdEncoding.DecodeString(val.(string))
		opts = append(opts, RequestWithBodyReader(strings.NewReader(string(decodeBytes))))
	}
	request := RequestFromMap(req, opts...)
	return NewContext(request, spider, WithContextId(req["ctxId"].(string))), nil

}
func (w *DistributedWorker) queueKey() (string, time.Duration) {
	key, ttl := w.getQueueKey()
	return fmt.Sprintf("%s:%s", key, w.currentSpider), ttl

}

// getSize 获取队列大小
func (w *DistributedWorker) isEmpty() bool {
	key, _ := w.queueKey()
	length, err := w.rdb.LLen(goContext.TODO(), key).Uint64()
	if err != nil {
		length = 0
		engineLog.Errorf("get queue len error %s", err.Error())

	}
	stop, err := w.CheckAllNodesStop()
	if err != nil {
		engineLog.Warnf("check all nodes status error %s", err.Error())

	}
	return int64(length) == 0 && stop
}

// getSize request 队列大小
func (w *DistributedWorker) getSize() uint64 {
	key, _ := w.queueKey()
	length, _ := w.rdb.LLen(goContext.TODO(), key).Uint64()
	return uint64(length)
}

// Fingerprint 生成request 对象的指纹
func (w *DistributedWorker) Fingerprint(ctx *Context) ([]byte, error) {
	fp, err := w.dupeFilter.Fingerprint(ctx)
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
func (w *DistributedWorker) DoDupeFilter(ctx *Context) (bool, error) {
	fp, err := w.Fingerprint(ctx)
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
func (w *DistributedWorker) bfKey() (string, time.Duration) {
	key, ttl := w.getBloomFilterKey()
	return fmt.Sprintf("%s:%s", key, w.currentSpider), ttl
}

// Add 添加指纹到布隆过滤器
func (w *DistributedWorker) Add(fingerprint []byte) error {
	h := baseHashes(fingerprint)
	pipe := w.rdb.Pipeline()
	key, _ := w.bfKey()
	for i := uint(0); i < w.bloomK; i++ {
		value := w.getOffset(h, i)
		pipe.SetBit(goContext.TODO(), key, int64(value), 1)
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
	key, _ := w.bfKey()
	for i := uint(0); i < w.bloomK; i++ {
		value := w.getOffset(h, i)
		result = append(result, pipe.GetBit(goContext.TODO(), key, int64(value)))
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

// getNodeKey 获取当前节点的缓存key
// 格式:{nodePrefix}:{spiderName}:{ip}:{nodeId}
func (w *DistributedWorker) getNodeKey() string {
	ip := GetMachineIp()

	return fmt.Sprintf("%s:%s:%s:%s", w.nodePrefix, w.currentSpider, ip, w.nodeId)
}

// getNodesSetKey 节点池缓存key
// 格式:{nodesSetPrefix}:{spiderName}
func (w *DistributedWorker) getNodesSetKey() string {
	return fmt.Sprintf("%s:%s", w.nodesSetPrefix, w.currentSpider)
}

// getMaterSetKey master 节点池缓存key
// 格式:{masterNodesKey}:{spiderName}
func (w *DistributedWorker) getMaterSetKey() string {
	return fmt.Sprintf("%s:%s", w.masterNodesKey, w.currentSpider)
}

// AddNode 新增节点
func (w *DistributedWorker) AddNode() error {
	ip := GetMachineIp()
	key := w.getNodeKey()
	status := w.rdb.SetEX(goContext.TODO(), key, 1, 10*time.Second)
	err := status.Err()
	if err != nil {
		return err
	}
	member := []interface{}{fmt.Sprintf("%s:%s", ip, w.nodeId)}
	nodesKey := w.getNodesSetKey()
	err = w.rdb.SAdd(goContext.TODO(), nodesKey, member...).Err()
	if err != nil {
		return fmt.Errorf("add node to nodes set error:%s", err.Error())
	}
	if w.isMaster {
		err = w.addMaster()
		if err != nil {
			return fmt.Errorf("add node to master set error:%s", err.Error())
		}
	}
	return err
}

// addMaster 新增master节点
func (w *DistributedWorker) addMaster() error {
	key := w.getMaterSetKey()
	nodeKey := w.getNodeKey()
	member := []interface{}{nodeKey}
	err := w.rdb.SAdd(goContext.TODO(), key, member...).Err()
	return err
}

// delMaster 删除master节点
func (w *DistributedWorker) delMaster() error {
	key := w.getMaterSetKey()
	nodeKey := w.getNodeKey()
	member := []interface{}{nodeKey}
	err := w.rdb.SRem(goContext.TODO(), key, member...).Err()
	return err
}

// CheckMasterLive 检查所有master 节点是否都在线
func (w *DistributedWorker) CheckMasterLive() (bool, error) {
	members := w.rdb.SMembers(goContext.TODO(), w.getMaterSetKey()).Val()
	count := len(members)
	pipe := w.rdb.Pipeline()
	result := []*redis.StringCmd{}

	for _, member := range members {
		result = append(result, pipe.Get(goContext.TODO(), member))
	}
	count, err := w.executeCheck(pipe, result, count)
	return count != 0, err

}

// DelNode 删除当前节点
func (w *DistributedWorker) DelNode() error {
	ip := GetMachineIp()
	key := w.getNodeKey()
	err := w.rdb.Del(goContext.TODO(), key).Err()
	if err != nil {
		return err
	}
	member := []interface{}{fmt.Sprintf("%s:%s", ip, w.nodeId)}
	err = w.rdb.SRem(goContext.TODO(), w.getNodesSetKey(), member...).Err()
	if w.isMaster {
		w.delMaster()
	}
	return err
}

// StopNode 停止当前节点的活动
func (w *DistributedWorker) StopNode() error {
	key := w.getNodeKey()
	err := w.rdb.SetEX(goContext.TODO(), key, 0, 1*time.Second).Err()
	if err != nil {
		return err
	}
	return nil
}

// Heartbeat 心跳包
func (w *DistributedWorker) Heartbeat() error {
	key := w.getNodeKey()
	err := w.rdb.SetEX(goContext.TODO(), key, 1, 1*time.Second).Err()
	if err != nil {
		return err
	}
	return nil
}

// executeCheck 检查所有的节点状态
func (w *DistributedWorker) executeCheck(pipe redis.Pipeliner, result []*redis.StringCmd, count int) (int, error) {
	_, err := pipe.Exec(goContext.TODO())
	if err != nil {
		if strings.Contains(err.Error(), "nil") {
			err = nil
		} else {
			return 0, err

		}
	}

	// 遍历所有的节点检查是否任务已经终止
	for _, r := range result {
		val, readErr := r.Int()
		if readErr != nil && !strings.Contains(readErr.Error(), "nil") {
			err = readErr
		}
		if val < 1 {
			// 删除节点
			count--
		}

	}
	return count, err
}

// CheckAllNodesStop 检查所有的节点是否都已经停止
func (w *DistributedWorker) CheckAllNodesStop() (bool, error) {
	members := w.rdb.SMembers(goContext.TODO(), w.getNodesSetKey()).Val()
	pipe := w.rdb.Pipeline()
	result := []*redis.StringCmd{}
	count := len(members)
	for _, member := range members {
		key := fmt.Sprintf("%s:%s:%s", w.nodePrefix, w.currentSpider, member)
		result = append(result, pipe.Get(goContext.TODO(), key))
	}
	count, err := w.executeCheck(pipe, result, count)
	return count == 0, err

}
func (w *DistributedWorker) close() error {
	return w.DelNode()
}
