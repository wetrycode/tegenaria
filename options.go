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

// import (
// 	"context"
// 	"time"
// )

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