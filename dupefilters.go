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

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	bloom "github.com/bits-and-blooms/bloom/v3"
	"github.com/spaolacci/murmur3"
)

// RFPDupeFilterInterface request 对象指纹计算和布隆过滤器去重
type RFPDupeFilterInterface interface {
	// Fingerprint request指纹计算
	Fingerprint(ctx *Context) ([]byte, error)

	// DoDupeFilter request去重
	DoDupeFilter(ctx *Context) (bool, error)
}
// RFPDupeFilter 去重组件
type RFPDupeFilter struct {
	bloomFilter *bloom.BloomFilter
}
// NewRFPDupeFilter 新建去重组件
// bloomP容错率
// bloomN数据规模
func NewRFPDupeFilter(bloomP float64, bloomN uint) *RFPDupeFilter {
	// 计算最佳的bit set大小
	bloomM := OptimalNumOfBits(int64(bloomN), bloomP)
	// 计算最佳的哈希函数大小
	bloomK := OptimalNumOfHashFunctions(int64(bloomN), bloomM)
	return &RFPDupeFilter{
		bloomFilter: bloom.New(uint(bloomM), uint(bloomK)),
	}
}

// canonicalizeUrl request 规整化处理
func (f *RFPDupeFilter) canonicalizetionUrl(request *Request, keepFragment bool) url.URL {
	u, _ := url.ParseRequestURI(request.Url)
	u.RawQuery = u.Query().Encode()
	u.ForceQuery = true
	if !keepFragment {
		u.Fragment = ""
	}
	return *u
}

// encodeHeader 请求头序列化
func (f *RFPDupeFilter) encodeHeader(request *Request) string {
	h := request.Header
	if h == nil {
		return ""
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	// Sort by Header key
	sort.Strings(keys)
	for _, k := range keys {
		// Sort by value
		buf.WriteString(fmt.Sprintf("%s:%s;\n", strings.ToUpper(k), strings.ToUpper(h[k])))
	}
	return buf.String()
}
// Fingerprint 计算指纹
func (f *RFPDupeFilter) Fingerprint(ctx *Context) ([]byte, error) {
	request:=ctx.Request
	if request.Url == "" {
		return nil, fmt.Errorf("request is nil,maybe it had been free")
	}
	// get sha128
	sha := murmur3.New128()
	method:=string(request.Method)
	_, err := io.WriteString(sha, method)
	if err != nil {
		return nil, err
	}
	// canonical request url
	u := f.canonicalizetionUrl(request, false)
	_, err = io.WriteString(sha, u.String())
	if err != nil {
		return nil, err
	}
	// read request body
	if request.Body != nil {

		sha.Write(request.Body)
	}
	// to handle request header
	if len(request.Header) != 0 {
		_, err := io.WriteString(sha, f.encodeHeader(request))
		if err != nil {
			return nil, err
		}
	}
	res := sha.Sum(nil)
	return res, nil
}

// DoDupeFilter 通过布隆过滤器对request对象进行去重处理
func (f *RFPDupeFilter) DoDupeFilter(ctx *Context) (bool, error) {
	// Use bloom filter to do fingerprint deduplication
	data, err := f.Fingerprint(ctx)
	if err != nil {
		return false, err
	}
	return f.bloomFilter.TestOrAdd(data), nil
}
