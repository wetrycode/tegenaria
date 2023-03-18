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
	"bytes"
	"encoding/gob"

	"github.com/wetrycode/tegenaria"
)

// newRdbCache 构建待缓存的数据
func newRdbCache(request *tegenaria.Request, ctxID string, spiderName string) (rdbCacheData, error) {
	r, err := request.ToMap()
	if err != nil {
		return nil, err
	}
	r["ctxId"] = ctxID
	name := request.Parser
	if name == "func1" {
		logger.Panicf("请求%s,%s, 获取到非法的解析函数", request.Url, ctxID)
	}
	r["parser"] = name
	if request.Proxy != nil {
		r["proxyUrl"] = request.Proxy.ProxyUrl

	}
	r["spiderName"] = spiderName
	return r, nil
}

// loads 从缓存队列中加载请求并反序列化
func (s *serialize) loads(request []byte) error {
	decoder := gob.NewDecoder(bytes.NewReader(request))
	return decoder.Decode(&s.val)

}

// dumps 序列化操作
func (s *serialize) dumps() error {
	enc := gob.NewEncoder(&s.buf)
	return enc.Encode(s.val)
}

// unserialize 对从rdb中读取到的二进制数据进行反序列化
// 返回一个rdbCacheData对象
func unserialize(data []byte) (rdbCacheData, error) {
	s := &serialize{
		buf: bytes.Buffer{},
		val: make(rdbCacheData),
	}
	err := s.loads(data)
	return s.val, err
}

// newSerialize 获取序列化组件
func newSerialize(r rdbCacheData) *serialize {
	return &serialize{
		val: r,
	}
}
