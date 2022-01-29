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

// Response the Request download response data
type Response struct {
	Status        int                 // Status request response status code
	Header        map[string][]string // Header response header
	Delay         float64             // Delay the time of handle download request
	ContentLength int                 // ContentLength response content length
	URL           string              // URL of request url
	Buffer        *bytes.Buffer       // buffer read response buffer
}

// responsePool a buffer poll of Response object
var responsePool *sync.Pool = &sync.Pool{
	New: func() interface{} {
		return new(Response)
	},
}
var respLog *logrus.Entry = GetLogger("response")

// Json deserialize the response body to json
func (r *Response) Json() (map[string]interface{},error) {
	jsonResp := map[string]interface{}{}
	err := json.Unmarshal(r.Buffer.Bytes(), &jsonResp)
	if err != nil {
		respLog.Errorf("Get json response error %s", err.Error())
		
		return nil, err
	}
	return jsonResp,nil
}

// String get response text from response body
func (r *Response) String() string {
	return r.Buffer.String()
}

// NewResponse create a new Response from responsePool
func NewResponse() *Response {
	response := responsePool.Get().(*Response)
	response.Buffer = bufferPool.Get().(*bytes.Buffer)
	response.Buffer.Reset()
	return response

}

// freeResponse reset Response and the put it into responsePool
func freeResponse(r *Response) {
	r.Status = -1
	r.Header = nil
	r.Delay = 0
	responsePool.Put(r)
	r = nil
}
