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

// EngineOption 引擎构造过程中的可选参数
type EngineOption func(r *CrawlEngine)

// EngineWithCache 引擎使用的缓存组件
// func EngineWithCache(cache CacheInterface) EngineOption {
// 	return func(r *CrawlEngine) {
// 		r.cache = cache
// 	}
// }

// EngineWithDownloader 引擎使用的下载器组件
func EngineWithDownloader(downloader Downloader) EngineOption {
	return func(r *CrawlEngine) {
		r.downloader = downloader

	}
}
func EngineWithComponents(components ComponentInterface) EngineOption {
	return func(r *CrawlEngine) {
		r.components = components

	}
}

// EngineWithFilter 引擎使用的过滤去重组件
// func EngineWithFilter(filter RFPDupeFilterInterface) EngineOption {
// 	return func(r *CrawlEngine) {
// 		r.rfpDupeFilter = filter

// 	}
// }

// EngineWithUniqueReq 是否进行去重处理
func EngineWithUniqueReq(uniqueReq bool) EngineOption {
	return func(r *CrawlEngine) {
		r.filterDuplicateReq = uniqueReq

	}
}

// EngineWithInterval 定时执行时间
// func EngineWithInterval(interval time.Duration) EngineOption {
// 	return func(r *CrawlEngine) {
// 		r.SetInterval(interval)
// 	}
// }

// EngineWithLimiter 引擎使用的限速器
// func EngineWithLimiter(limiter LimitInterface) EngineOption {
// 	return func(r *CrawlEngine) {
// 		r.limiter = limiter
// 	}
// }

// EngineWithDistributedWorker 引擎使用的的分布式组件
// func EngineWithDistributedWorker(woker DistributedWorkerInterface) EngineOption {
// 	return func(r *CrawlEngine) {
// 		r.cache = woker
// 		r.limiter = woker.GetLimiter()
// 		r.rfpDupeFilter = woker
// 		r.useDistributed = true
// 		r.checkMasterLive = woker.CheckMasterLive
// 		woker.SetMaster(r.isMaster)
// 		r.hooker = NewDistributedHooks(woker)
// 		// woker.SetSpiders(r.GetSpiders())
// 	}
// }
// func EngineWithStastCollector(stats StatisticInterface) EngineOption {
// 	return func(r *CrawlEngine) {
// 		r.statistic = stats
// 	}
// }
