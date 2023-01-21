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
	"encoding/json"
	"sync"

	"github.com/sirupsen/logrus"
)

// Response 请求响应体的结构
type Response struct {
	// Status状态码
	Status        int                 
	// Header 响应头
	Header        map[string][]string // Header response header
	// Delay 请求延迟
	Delay         float64             // Delay the time of handle download request
	// ContentLength 响应体大小
	ContentLength uint64                 // ContentLength response content length
	// URL 请求url
	URL           string              // URL of request url
	// Buffer 响应体缓存
	Buffer        *bytes.Buffer       // buffer read response buffer
}

// responsePool Response 对象内存池
var responsePool *sync.Pool = &sync.Pool{
	New: func() interface{} {
		return new(Response)
	},
}
var respLog *logrus.Entry = GetLogger("response")

// Json 将响应数据转为json
func (r *Response) Json() (map[string]interface{},error) {
	jsonResp := map[string]interface{}{}
	err := json.Unmarshal(r.Buffer.Bytes(), &jsonResp)
	if err != nil {
		respLog.Errorf("Get json response error %s", err.Error())
		
		return nil, err
	}
	return jsonResp,nil
}

// String 将响应数据转为string
func (r *Response) String() string {
	return r.Buffer.String()
}

// NewResponse 从内存池中创建新的response对象
func NewResponse() *Response {
	response := responsePool.Get().(*Response)
	response.Buffer = bufferPool.Get().(*bytes.Buffer)
	response.Buffer.Reset()
	return response

}

// freeResponse 重置response对象并放回对象池
func freeResponse(r *Response) {
	r.Status = -1
	r.Header = nil
	r.Delay = 0
	responsePool.Put(r)
	r = nil
}
