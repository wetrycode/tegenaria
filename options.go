// Copyright 2022 geebytes
// Licensed under the Apache License, Version 2.0 (the 'License');
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an 'AS IS' BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tegenaria

import "time"

// EngineOption 引擎构造过程中的可选参数
type EngineOption func(r *CrawlEngine)

// EngineWithCache 引擎使用的缓存组件
func EngineWithCache(cache CacheInterface) EngineOption {
	return func(r *CrawlEngine) {
		r.cache = cache
	}
}

// EngineWithDownloader 引擎使用的下载器组件
func EngineWithDownloader(downloader Downloader) EngineOption {
	return func(r *CrawlEngine) {
		r.downloader = downloader

	}
}

// EngineWithFilter 引擎使用的过滤去重组件
func EngineWithFilter(filter RFPDupeFilterInterface) EngineOption {
	return func(r *CrawlEngine) {
		r.rfpDupeFilter = filter

	}
}

// EngineWithUniqueReq 是否进行去重处理
func EngineWithUniqueReq(uniqueReq bool) EngineOption {
	return func(r *CrawlEngine) {
		r.filterDuplicateReq = uniqueReq

	}
}

// EngineWithLimiter 引擎使用的限速器
func EngineWithLimiter(limiter LimitInterface) EngineOption {
	return func(r *CrawlEngine) {
		r.limiter = limiter
	}
}

// EngineWithDistributedWorker 引擎使用的的分布式组件
func EngineWithDistributedWorker(woker DistributedWorkerInterface) EngineOption {
	return func(r *CrawlEngine) {
		r.cache = woker
		r.limiter = woker.GetLimter()
		r.rfpDupeFilter = woker
		r.useDistributed = true
		r.checkMasterLive = woker.CheckMasterLive
		woker.SetMaster(r.isMaster)
		r.hooker = NewDistributedHooks(woker)
		woker.SetSpiders(r.GetSpiders())
	}
}

// DistributedWithConnectionsSize rdb 连接池最大连接数
func DistributedWithConnectionsSize(size int) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.RdbConnectionsSize = uint64(size)
	}
}

// DistributedWithRdbTimeout rdb超时时间设置
func DistributedWithRdbTimeout(timeout time.Duration) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.RdbTimeout = timeout
	}
}

// DistributedWithRdbMaxRetry rdb失败重试次数
func DistributedWithRdbMaxRetry(retry int) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.RdbMaxRetry = retry
	}
}

// DistributedWithBloomP 布隆过滤器容错率
func DistributedWithBloomP(bloomP float64) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.BloomP = bloomP
	}
}

// DistributedWithBloomN 布隆过滤器数据规模
func DistributedWithBloomN(bloomN uint) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.BloomN = bloomN
	}
}

// DistributedWithLimiterRate 分布式组件下限速器的限速值
func DistributedWithLimiterRate(rate int) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.LimiterRate = rate
	}
}

// DistributedWithGetqueueKey 队列key生成函数
func DistributedWithGetqueueKey(keyFunc GetRDBKey) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.GetqueueKey = keyFunc
	}
}

// DistributedWithGetBFKey 布隆过滤器的key生成函数
func DistributedWithGetBFKey(keyFunc GetRDBKey) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.GetBFKey = keyFunc
	}
}

// DistributedWithGetLimitKey 限速器key的生成函数
func DistributedWithGetLimitKey(keyFunc GetRDBKey) DistributeOptions {
	return func(w *DistributedWorkerConfig) {
		w.getLimitKey = keyFunc
	}
}
