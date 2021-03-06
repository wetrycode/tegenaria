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

import (
	"context"
	"time"
)

// EngineOption the options params of NewDownloader
type EngineOption func(r *SpiderEngine)

// EngineWithContext set engine context
func EngineWithContext(ctx context.Context) EngineOption {
	return func(r *SpiderEngine) {
		r.Ctx = ctx
		engineLog.Infoln("Set engine context to ", ctx)
	}
}

// EngineWithTimeout set request download timeout
func EngineWithTimeout(timeout time.Duration) EngineOption {
	return func(r *SpiderEngine) {
		r.DownloadTimeout = timeout
		engineLog.Infoln("Set download timeout to ", timeout)

	}
}

// EngineWithDownloader set spider engine downloader
func EngineWithDownloader(downloader Downloader) EngineOption {
	return func(r *SpiderEngine) {
		r.requestDownloader = downloader
		engineLog.Infoln("Set downloader to ", downloader)

	}
}

// EngineWithAllowStatusCode set request response allow status
func EngineWithAllowStatusCode(allowStatusCode []uint64) EngineOption {
	return func(r *SpiderEngine) {
		r.allowStatusCode = allowStatusCode
		engineLog.Infoln("Set request response allow status to ", allowStatusCode)

	}
}

// EngineWithUniqueReq set request unique flag
func EngineWithUniqueReq(uniqueReq bool) EngineOption {
	return func(r *SpiderEngine) {
		r.filterDuplicateReq = uniqueReq
		engineLog.Infoln("Set request unique flag to ", uniqueReq)

	}
}

// EngineWithSchedulerNum set engine scheduler number
// default to cpu number
func EngineWithSchedulerNum(schedulerNum uint) EngineOption {
	return func(r *SpiderEngine) {
		r.schedulerNum = schedulerNum
		engineLog.Infoln("Set engine scheduler number to ", schedulerNum)

	}
}

// EngineWithReadCacheNum set cache reader number
func EngineWithReadCacheNum(cacheReadNum uint) EngineOption {
	return func(r *SpiderEngine) {
		r.cacheReadNum = cacheReadNum
		engineLog.Infoln("Set engine cache reader to ", cacheReadNum)

	}
}

// EngineWithRequestNum set request channel buffer size
// request channel buffer size default to 1024
func EngineWithRequestNum(requestNum int64) EngineOption {
	return func(r *SpiderEngine) {
		r.downloadLimit = requestNum
		engineLog.Infoln("Set request channel buffer size ", requestNum)

	}
}
