package tegenaria

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// NewRdbConfig redis 配置构造函数
func NewRdbConfig(config *DistributedWorkerConfig) *redis.Options {
	return &redis.Options{
		Password: config.RedisPasswd, // 密码
		Username: config.RedisUsername, //用户名
		DB:       int(config.RedisDB),  // redis数据库index

		//连接池容量及闲置连接数量
		PoolSize:     int(config.RdbConnectionsSize), // 连接池最大socket连接数，默认为4倍CPU数， 4 * runtime.NumCPU
		MinIdleConns: 10,                             //在启动阶段创建指定数量的Idle连接，并长期维持idle状态的连接数不少于指定数量；。

		//超时
		DialTimeout:  config.RdbTimeout, //连接建立超时时间，默认5秒。
		ReadTimeout:  config.RdbTimeout, //读超时，默认3秒， -1表示取消读超时
		WriteTimeout: config.RdbTimeout, //写超时，默认等于读超时
		PoolTimeout:  config.RdbTimeout, //当所有连接都处在繁忙状态时，客户端等待可用连接的最大等待时长，默认为读超时+1秒。

		//闲置连接检查包括IdleTimeout，MaxConnAge
		IdleCheckFrequency: 60 * time.Second, //闲置连接检查的周期，默认为1分钟，-1表示不做周期性检查，只在客户端获取连接时对闲置连接进行处理。
		IdleTimeout:        5 * time.Minute,  //闲置超时，默认5分钟，-1表示取消闲置超时检查
		MaxConnAge:         0 * time.Second,  //连接存活时长，从创建开始计时，超过指定时长则关闭连接，默认为0，即不关闭存活时长较长的连接

		//命令执行失败时的重试策略
		MaxRetries:      config.RdbMaxRetry,     // 命令执行失败时，最多重试多少次，默认为0即不重试
		MinRetryBackoff: 8 * time.Millisecond,   //每次计算重试间隔时间的下限，默认8毫秒，-1表示取消间隔
		MaxRetryBackoff: 512 * time.Millisecond, //每次计算重试间隔时间的上限，默认512毫秒，-1表示取消间隔

	}
}
func NewRdbClient(config *DistributedWorkerConfig) *redis.Client {
	options := NewRdbConfig(config)
	options.Addr = config.RedisAddr
	rdb := redis.NewClient(options)
	err:=rdb.Ping(context.TODO()).Err()
	if err!=nil{
		panic(err)
	}
	return rdb
}

func NewRdbClusterCLient(config *WorkerConfigWithRdbCluster) *redis.ClusterClient {
	client:=redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.RdbNodes,
		// MaxRetries: config.DistributedWorkerConfig.RdbMaxRetry,

		//连接池容量及闲置连接数量
		NewClient: func(opt *redis.Options) *redis.Client {
			addr :=opt.Addr
			opt = NewRdbConfig(config.DistributedWorkerConfig)
			opt.Addr = addr
			return redis.NewClient(opt)
		},
		// To route commands by latency or randomly, enable one of the following.
		RouteByLatency: true,
		RouteRandomly:  true,
	})
	err:=client.Ping(context.TODO()).Err()
	if err!=nil{
		panic(err)
	}
	return client
}
