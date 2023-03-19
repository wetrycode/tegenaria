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

// EngineWithUniqueReq 是否进行去重处理
func EngineWithUniqueReq(uniqueReq bool) EngineOption {
	return func(r *CrawlEngine) {
		r.filterDuplicateReq = uniqueReq

	}
}

// EngineWithReqChannelSize
func EngineWithReqChannelSize(size int) EngineOption {
	return func(r *CrawlEngine) {
		r.reqChannelSize = size
		r.requestsChan = make(chan *Context, size)
	}
}
