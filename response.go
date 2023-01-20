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
