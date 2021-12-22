package tegenaria

import (
	"encoding/json"
	"sync"
	"unsafe"

	"github.com/sirupsen/logrus"
)

type Response struct {
	Text          string
	Status        int
	Body          []byte
	Header        map[string][]byte
	Req           *Request
	Delay         float64
	ContentLength int
}

var responsePool *sync.Pool = &sync.Pool{
	New: func() interface{} {
		return new(Response)
	},
}
var respLog *logrus.Entry = GetLogger("response")

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

func (r *Response) String() string {
	defer func() {
		if p := recover(); p != nil {
			respLog.Errorf("panic recover! p: %v", p)
		}

	}()
	return *(*string)(unsafe.Pointer(&r.Body))
}
func NewResponse() *Response {
	response := responsePool.Get().(*Response)
	return response
	// return new(Response)

}
func (r *Response) freeResponse() {
	r.Text = ""
	r.Status = -1
	r.Body = r.Body[:0]
	r.Header = nil
	r.Delay = 0
	r.Req.freeRequest()
	responsePool.Put(r)
}
