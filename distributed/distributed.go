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
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wetrycode/tegenaria"
)

// DistributedWorker 分布式组件，包含两个组件:
// request请求缓存队列，由各个节点上的引擎读队列消费，
// redis 队列缓存的是经过gob序列化之后的二进制数据
// 布隆过滤器主要是用于去重
// 该组件同时实现了RFPDupeFilterInterface 和CacheInterface
type DistributedWorker struct {
	// rdb redis客户端支持redis单机实例和redis cluster集群模式
	rdb redis.Cmdable

	// nodeID 节点id
	nodeID string
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
	// ip 本机地址
	ip string
}

// RdbNodes redis cluster 节点地址
type RdbNodes []string

// DistributeOptions 分布式组件的可选参数
type DistributeOptions func(w *DistributedWorkerConfig)

// InfluxdbConfig influxdb配置
type InfluxdbConfig struct {
	// influxdbServer influxdb服务链接
	influxdbServer string
	// influxdbToken influxdb api token
	influxdbToken string
	// influxdbBucket influxdb bucket 名称
	influxdbBucket string
	// influxdbOrg influxdb org 名称
	influxdbOrg string
}

// RedisConfig redis配置
type RedisConfig struct {
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
}

// DistributedWorkerConfig 分布式组件的配置参数
type DistributedWorkerConfig struct {
	redisConfig *RedisConfig

	// LimiterRate 限速大小
	LimiterRate int
	// bloomP 布隆过滤器的容错率
	bloomP float64
	// bloomN 数据规模，比如1024 * 1024
	bloomN int
	// isMaster 是否是主节点
	isMaster bool
	influxdb *InfluxdbConfig
	// getLimitKey 获取限速器redis key前缀
	getLimitKey GetRDBKey
	// getBloomFilterKey 布隆过滤器redis key前缀
	getBloomFilterKey GetRDBKey
	// getQueueKey 消息队列的key
	getQueueKey GetRDBKey
}

// WorkerConfigWithRdbCluster redis cluster 模式下的分布式组件配置参数
type WorkerConfigWithRdbCluster struct {
	*DistributedWorkerConfig
	RdbNodes
}

func NewRedisConfig(addr string, username string, passwd string, db uint32) *RedisConfig {
	return &RedisConfig{
		RedisUsername:      username,
		RedisPasswd:        passwd,
		RedisDB:            db,
		RdbConnectionsSize: 32,
		RdbTimeout:         10 * time.Second,
		RdbMaxRetry:        3,
		RedisAddr:          addr,
	}
}
func NewInfluxdbConfig(server string, token string, bucket string, org string) *InfluxdbConfig {
	return &InfluxdbConfig{
		influxdbServer: server,
		influxdbToken:  token,
		influxdbBucket: bucket,
		influxdbOrg:    org,
	}
}

// NewDistributedWorkerConfig 新建分布式组件的配置
func NewDistributedWorkerConfig(rdbConfig *RedisConfig, influxdbConfig *InfluxdbConfig, opts ...DistributeOptions) *DistributedWorkerConfig {
	config := &DistributedWorkerConfig{
		redisConfig:       rdbConfig,
		influxdb:          influxdbConfig,
		LimiterRate:       32,
		bloomP:            0.001,
		bloomN:            1024 * 256,
		isMaster:          true,
		getLimitKey:       getLimiterDefaultKey,
		getBloomFilterKey: getBloomFilterKey,
		getQueueKey:       getQueueKey,
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
func NewDistributedWorker(config *DistributedWorkerConfig) *DistributedWorker {
	rdb := NewRdbClient(config)
	ip, err := tegenaria.GetMachineIP()
	if err != nil {
		panic(fmt.Sprintf("get machine ip error %s", err.Error()))
	}
	d := &DistributedWorker{
		rdb:            rdb,
		nodeID:         tegenaria.GetEngineID(),
		ip:             ip,
		nodesSetPrefix: "tegenaria:v1:nodes",
		nodePrefix:     "tegenaria:v1:node",
		masterNodesKey: "tegenaria:v1:master",
		isMaster:       true,
	}
	return d
}

// NewWorkerWithRdbCluster redis cluster模式下的分布式工作组件
func NewWorkerWithRdbCluster(config *WorkerConfigWithRdbCluster) *DistributedWorker {
	w := NewDistributedWorker(config.DistributedWorkerConfig)
	// 替换为redis cluster客户端
	w.rdb = NewRdbClusterCLient(config)
	return w
}

// getLimiterDefaultKey 限速器默认的key
func getLimiterDefaultKey() (string, time.Duration) {
	return "tegenaria:v1:limiter", 0 * time.Second
}

// SetMaster 设置当前的节点是否为master
func (w *DistributedWorker) SetMaster(flag bool) {
	w.isMaster = flag
}

// CheckAllNodesStop 检查所有的节点是否都已经停止
func (w *DistributedWorker) CheckAllNodesStop() (bool, error) {
	members := w.rdb.SMembers(context.TODO(), w.getNodesSetKey()).Val()
	pipe := w.rdb.Pipeline()
	result := []*redis.StringCmd{}
	count := len(members)
	// 检查所有节点的状态
	for _, member := range members {
		key := fmt.Sprintf("%s:%s:%s", w.nodePrefix, w.currentSpider, member)
		result = append(result, pipe.Get(context.TODO(), key))
	}
	count, err := w.executeCheck(pipe, result, count)
	return count == 0, err

}

// getNodeKey 获取当前节点的缓存key
// 格式:{nodePrefix}:{spiderName}:{ip}:{nodeID}
func (w *DistributedWorker) getNodeKey() string {
	return fmt.Sprintf("%s:%s:%s:%s", w.nodePrefix, w.currentSpider, w.ip, w.nodeID)
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
	key := w.getNodeKey()
	// 注册节点信息
	status := w.rdb.SetEx(context.TODO(), key, 1, 10*time.Second)
	err := status.Err()
	defer func() {
		logger.Infof("注册节点:%s", key)
	}()
	if err != nil {
		return err
	}
	// 将节点加入节点池
	member := []interface{}{fmt.Sprintf("%s:%s", w.ip, w.nodeID)}
	nodesKey := w.getNodesSetKey()
	err = w.rdb.SAdd(context.TODO(), nodesKey, member...).Err()
	if err != nil {
		return fmt.Errorf("add node to nodes set error:%s", err.Error())
	}
	if w.isMaster {
		// 添加master节点
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
	err := w.rdb.SAdd(context.TODO(), key, member...).Err()
	return err
}

// delMaster 删除master节点
func (w *DistributedWorker) delMaster() error {
	key := w.getMaterSetKey()
	nodeKey := w.getNodeKey()
	member := []interface{}{nodeKey}
	err := w.rdb.SRem(context.TODO(), key, member...).Err()
	return err
}

// CheckMasterLive 检查所有master 节点是否都在线
func (w *DistributedWorker) CheckMasterLive() (bool, error) {
	members := w.rdb.SMembers(context.TODO(), w.getMaterSetKey()).Val()
	count := len(members)
	pipe := w.rdb.Pipeline()
	result := []*redis.StringCmd{}

	for _, member := range members {
		result = append(result, pipe.Get(context.TODO(), member))
	}
	count, err := w.executeCheck(pipe, result, count)
	logger.Infof("主节点个数:%d", count)
	return count != 0, err

}

// DelNode 删除当前节点
func (w *DistributedWorker) DelNode() error {
	key := w.getNodeKey()
	err := w.rdb.Del(context.TODO(), key).Err()
	if err != nil {
		return err
	}
	member := []interface{}{fmt.Sprintf("%s:%s", w.ip, w.nodeID)}
	err = w.rdb.SRem(context.TODO(), w.getNodesSetKey(), member...).Err()
	if err != nil && w.isMaster {
		err = w.delMaster()
	}
	return err
}

// StopNode 停止当前节点的活动
func (w *DistributedWorker) StopNode() error {
	key := w.getNodeKey()
	err := w.rdb.SetEx(context.TODO(), key, 0, 1*time.Second).Err()
	return err
}

// Heartbeat 心跳包
func (w *DistributedWorker) Heartbeat() error {
	key := w.getNodeKey()
	err := w.rdb.SetEx(context.TODO(), key, 1, 1*time.Second).Err()
	return err
}

// executeCheck 检查所有的节点状态
func (w *DistributedWorker) executeCheck(pipe redis.Pipeliner, result []*redis.StringCmd, count int) (int, error) {
	_, err := pipe.Exec(context.TODO())
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

func (w *DistributedWorker) close() error {
	return w.DelNode()
}
func (w *DistributedWorker) GetRDB() redis.Cmdable {
	return w.rdb
}

func (w *DistributedWorker) GetWorkerID() string {
	return w.nodeID
}
func (w *DistributedWorker) SetCurrentSpider(spider tegenaria.SpiderInterface) {
	w.currentSpider = spider.GetName()
}

func (w *DistributedWorker) IsMaster() bool {
	return w.isMaster
}
