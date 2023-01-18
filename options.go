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


// EngineOption the options params of NewDownloader
type EngineOption func(r *CrawlEngine)
func EngineWithCache(cache CacheInterface)EngineOption{
	return func (r *CrawlEngine){
		r.cache = cache
	}
}

// EngineWithDownloader set spider engine downloader
func EngineWithDownloader(downloader Downloader) EngineOption {
	return func(r *CrawlEngine) {
		r.downloader = downloader

	}
}
func EngineWithFilter(filter RFPDupeFilterInterface) EngineOption {
	return func(r *CrawlEngine) {
		r.RFPDupeFilter = filter

	}
}

// EngineWithUniqueReq set request unique flag
func EngineWithUniqueReq(uniqueReq bool) EngineOption {
	return func(r *CrawlEngine) {
		r.filterDuplicateReq = uniqueReq

	}
}
func EngineWithLimiter(limiter LimitInterface) EngineOption {
	return func(r *CrawlEngine) {
		r.limiter = limiter
	}
}

func EngineWithDistributedWorker(woker DistributedWorkerInterface)EngineOption{
	return func(r *CrawlEngine) {
		r.cache = woker
		r.limiter = woker.GetLimter()
		r.RFPDupeFilter = woker
		r.useDistributed = true
		r.checkMasterLive = woker.CheckMasterLive
		woker.SetMaster(r.isMaster)
		r.hooker = NewDistributedHooks(woker)
		woker.SetSpiders(r.GetSpiders())
	}
}

func DistributedWithConnectionsSize(size int)DistributeOptions{
	return func(w *DistributedWorkerConfig) {
		w.RdbConnectionsSize = uint64(size)
	}
}
func DistributedWithRdbTimeout(timeout time.Duration)DistributeOptions{
	return func(w *DistributedWorkerConfig) {
		w.RdbTimeout = timeout
	}
}
func DistributedWithRdbMaxRetry(retry int)DistributeOptions{
	return func(w *DistributedWorkerConfig) {
		w.RdbMaxRetry = retry
	}
}

func DistributedWithBloomP(bloomP float64)DistributeOptions{
	return func(w *DistributedWorkerConfig) {
		w.BloomP = bloomP
	}
}

func DistributedWithBloomN(bloomN uint)DistributeOptions{
	return func(w *DistributedWorkerConfig) {
		w.BloomN = bloomN
	}
}

func DistributedWithLimiterRate(rate int)DistributeOptions{
	return func(w *DistributedWorkerConfig) {
		w.LimiterRate = rate
	}
}

func DistributedWithGetqueueKey(keyFunc GetRDBKey)DistributeOptions{
	return func(w *DistributedWorkerConfig) {
		w.GetqueueKey = keyFunc
	}
}

func DistributedWithGetBFKey(keyFunc GetRDBKey)DistributeOptions{
	return func(w *DistributedWorkerConfig) {
		w.GetBFKey = keyFunc
	}
}

func DistributedWithGetLimitKey(keyFunc GetRDBKey)DistributeOptions{
	return func(w *DistributedWorkerConfig) {
		w.getLimitKey = keyFunc
	}
}