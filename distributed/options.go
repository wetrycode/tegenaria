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

import "time"

// DistributedWithConnectionsSize rdb 连接池最大连接数
func DistributedWithConnectionsSize(size int) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.redisConfig.RdbConnectionsSize = uint64(size)
	}
}

// DistributedWithRdbTimeout rdb超时时间设置
func DistributedWithRdbTimeout(timeout time.Duration) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.redisConfig.RdbTimeout = timeout
	}
}

// DistributedWithRdbMaxRetry rdb失败重试次数
func DistributedWithRdbMaxRetry(retry int) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.redisConfig.RdbMaxRetry = retry
	}
}

// DistributedWithLimiterRate 分布式组件下限速器的限速值
func DistributedWithLimiterRate(rate int) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.LimiterRate = rate
	}
}

// DistributedWithGetQueueKey 队列key生成函数
func DistributedWithGetQueueKey(keyFunc GetRDBKey) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.getQueueKey = keyFunc
	}
}

// DistributedWithGetBFKey 布隆过滤器的key生成函数
func DistributedWithGetBFKey(keyFunc GetRDBKey) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.getBloomFilterKey = keyFunc
	}
}

// DistributedWithGetLimitKey 限速器key的生成函数
func DistributedWithGetLimitKey(keyFunc GetRDBKey) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.getLimitKey = keyFunc
	}
}

// DistributedWithBloomP 布隆过滤器容错率
func DistributedWithBloomP(bloomP float64) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.bloomP = bloomP
	}
}

// DistributedWithBloomN 布隆过滤器数据规模
func DistributedWithBloomN(bloomN int) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.bloomN = bloomN
	}
}

func DistributedWithSlave() DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.isMaster = false
	}
}
