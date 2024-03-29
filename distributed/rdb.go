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
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRdbConfig redis 配置构造函数
func NewRdbConfig(config *DistributedWorkerConfig) *redis.Options {
	return &redis.Options{
		Password: config.redisConfig.RedisPasswd,   // 密码
		Username: config.redisConfig.RedisUsername, //用户名
		DB:       int(config.redisConfig.RedisDB),  // redis数据库index

		//连接池容量及闲置连接数量
		PoolSize:     int(config.redisConfig.RdbConnectionsSize), // 连接池最大socket连接数，默认为4倍CPU数， 4 * runtime.NumCPU
		MinIdleConns: 10,                                         //在启动阶段创建指定数量的Idle连接，并长期维持idle状态的连接数不少于指定数量；。

		//超时
		DialTimeout:  config.redisConfig.RdbTimeout, //连接建立超时时间，默认5秒。
		ReadTimeout:  config.redisConfig.RdbTimeout, //读超时，默认3秒， -1表示取消读超时
		WriteTimeout: config.redisConfig.RdbTimeout, //写超时，默认等于读超时
		PoolTimeout:  config.redisConfig.RdbTimeout, //当所有连接都处在繁忙状态时，客户端等待可用连接的最大等待时长，默认为读超时+1秒。

		//闲置连接检查包括IdleTimeout，MaxConnAge
		// I: 60 * time.Second, //闲置连接检查的周期，默认为1分钟，-1表示不做周期性检查，只在客户端获取连接时对闲置连接进行处理。
		// IdleTimeout:        5 * time.Minute,  //闲置超时，默认5分钟，-1表示取消闲置超时检查
		// MaxConnAge:         0 * time.Second,  //连接存活时长，从创建开始计时，超过指定时长则关闭连接，默认为0，即不关闭存活时长较长的连接

		//命令执行失败时的重试策略
		MaxRetries:      config.redisConfig.RdbMaxRetry, // 命令执行失败时，最多重试多少次，默认为0即不重试
		MinRetryBackoff: 8 * time.Millisecond,           //每次计算重试间隔时间的下限，默认8毫秒，-1表示取消间隔
		MaxRetryBackoff: 512 * time.Millisecond,         //每次计算重试间隔时间的上限，默认512毫秒，-1表示取消间隔

	}
}
func NewRdbClient(config *DistributedWorkerConfig) *redis.Client {
	options := NewRdbConfig(config)
	options.Addr = config.redisConfig.RedisAddr
	rdb := redis.NewClient(options)
	err := rdb.Ping(context.TODO()).Err()
	// RedisAddr 为空说明处于集群模式
	if err != nil && config.redisConfig.RedisAddr != "" {
		panic(err)
	}
	return rdb
}

func NewRdbClusterCLient(config *WorkerConfigWithRdbCluster) *redis.ClusterClient {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: config.RdbNodes,
		// MaxRetries: config.DistributedWorkerConfig.RdbMaxRetry,

		//连接池容量及闲置连接数量
		NewClient: func(opt *redis.Options) *redis.Client {
			addr := opt.Addr
			opt = NewRdbConfig(config.DistributedWorkerConfig)
			opt.Addr = addr
			return redis.NewClient(opt)
		},
		// To route commands by latency or randomly, enable one of the following.
		RouteByLatency: true,
		RouteRandomly:  true,
	})
	err := client.Ping(context.TODO()).Err()
	if err != nil {
		panic(err)
	}
	return client
}

// GetRDBKey 获取缓存rdb key和ttl
type GetRDBKey func() (string, time.Duration)
