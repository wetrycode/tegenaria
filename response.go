package tegenaria

import (
	"bytes"
	"encoding/json"
	"sync"
	"unsafe"

	"github.com/sirupsen/logrus"
)

// Response the Request download response data
type Response struct {
	Status        int                 // Status request response status code
	Body          []byte              // Body response body
	Header        map[string][]string // Header response header
	Req           *Request            // req the Request object
	Delay         float64             // Delay the time of handle download request
	ContentLength int                 // ContentLength response content length
	URL           string              // URL of request url
	buffer        *bytes.Buffer       // buffer read response buffer
}

// ResponseBufferPool a buffer poll of request response object
// it is used by downloader
var ResponseBufferPool *sync.Pool = &sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 4096))
	},
}

// responsePool a buffer poll of Response object
var responsePool *sync.Pool = &sync.Pool{
	New: func() interface{} {
		return new(Response)
	},
}
var respLog *logrus.Entry = GetLogger("response")

// Json deserialize the response body to json
func (r *Response) Json() map[string]interface{} {
	defer func() {
		if p := recover(); p != nil {
			respLog.Errorf("panic recover! p: %v", p)
		}

	}()
	jsonResp := map[string]interface{}{}
	err := json.Unmarshal(r.Body, &jsonResp)
	if err != nil {
		respLog.Errorf("Get json response error %s", err.Error())
	}
	return jsonResp
}

// String get response text from response body
func (r *Response) String() string {
	defer func() {
		if p := recover(); p != nil {
			respLog.Errorf("panic recover! p: %v", p)
		}

	}()
	return *(*string)(unsafe.Pointer(&r.Body))
}

// NewResponse create a new Response from responsePool
func NewResponse() *Response {
	response := responsePool.Get().(*Response)
	response.buffer = bufferPool.Get().(*bytes.Buffer)
	response.buffer.Reset()
	if len(response.Body) != 0 || response.Req != nil {
		respLog.Infof("Get response object not empty")
	}
	return response

}
func (r *Response) write() {
	if r.buffer != nil {
		r.Body = r.buffer.Bytes()
	}
}

// freeResponse reset Response and the put it into responsePool
func freeResponse(r *Response) {
	r.Status = -1
	r.Body = r.Body[:0]
	r.Header = nil
	r.Delay = 0
	bufferPool.Put(r.buffer)
	freeRequest(r.Req)
	responsePool.Put(r)
	r = nil
}
